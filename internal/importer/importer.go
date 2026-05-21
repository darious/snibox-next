package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/darious1472/snibox-next/internal/id"
	"github.com/darious1472/snibox-next/internal/store"
)

type Mode string

const (
	ModeMerge   Mode = "merge"
	ModeAppend  Mode = "append"
	ModeReplace Mode = "replace"
)

// Wire is the JSON shape used by both export and import.
type Wire struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Type      string    `json:"type"`
	Language  *string   `json:"language"`
	URL       *string   `json:"url"`
	Tags      []string  `json:"tags"`
	Pinned    bool      `json:"pinned"`
	Archived  bool      `json:"archived"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Report is returned to the caller of POST /import.
type Report struct {
	Mode      Mode             `json:"mode"`
	Imported  int              `json:"imported"`
	Errors    []ImportError    `json:"errors,omitempty"`
}

type ImportError struct {
	Index int    `json:"index"`
	ID    string `json:"id,omitempty"`
	Error string `json:"error"`
}

// ToWire converts internal items to the JSON-wire representation.
func ToWire(items []*store.Item) []Wire {
	out := make([]Wire, 0, len(items))
	for _, it := range items {
		out = append(out, Wire{
			ID:        it.ID,
			Title:     it.Title,
			Body:      it.Body,
			Type:      string(it.Type),
			Language:  it.Language,
			URL:       it.URL,
			Tags:      append([]string(nil), it.Tags...),
			Pinned:    it.Pinned,
			Archived:  it.Archived,
			CreatedAt: it.CreatedAt,
			UpdatedAt: it.UpdatedAt,
		})
	}
	return out
}

func fromWire(w Wire) *store.Item {
	return &store.Item{
		ID:        w.ID,
		Title:     w.Title,
		Body:      w.Body,
		Type:      store.ItemType(w.Type),
		Language:  w.Language,
		URL:       w.URL,
		Tags:      append([]string(nil), w.Tags...),
		Pinned:    w.Pinned,
		Archived:  w.Archived,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}

// Run reads JSON from r and applies it to repo per mode. Returns a report.
// Per SPEC.md §6.2:
//   replace: preserve imported created_at/updated_at; wipe table first.
//   merge:   preserve imported created_at/updated_at; upsert by id.
//   append:  regenerate id/created_at/updated_at; insert all.
//
// The entire write is wrapped in a single transaction (via the repo methods).
// Validation errors abort the import.
func Run(ctx context.Context, repo *store.Repo, r io.Reader, mode Mode) (Report, error) {
	switch mode {
	case "", ModeMerge:
		mode = ModeMerge
	case ModeAppend, ModeReplace:
	default:
		return Report{Mode: mode}, fmt.Errorf("invalid mode %q", mode)
	}

	var raw []Wire
	dec := json.NewDecoder(r)
	if err := dec.Decode(&raw); err != nil {
		return Report{Mode: mode}, fmt.Errorf("decode: %w", err)
	}

	items := make([]*store.Item, 0, len(raw))
	report := Report{Mode: mode}
	now := time.Now().UTC()
	for i, w := range raw {
		it := fromWire(w)
		if mode == ModeAppend {
			it.ID = id.New()
			it.CreatedAt = now
			it.UpdatedAt = now
		}
		// Normalise + validate.
		for j, t := range it.Tags {
			it.Tags[j] = store.NormaliseTag(t)
		}
		if it.CreatedAt.IsZero() {
			it.CreatedAt = now
		}
		if it.UpdatedAt.IsZero() {
			it.UpdatedAt = it.CreatedAt
		}
		if err := it.Validate(); err != nil {
			report.Errors = append(report.Errors, ImportError{Index: i, ID: w.ID, Error: err.Error()})
			continue
		}
		if mode != ModeAppend && it.ID == "" {
			report.Errors = append(report.Errors, ImportError{Index: i, Error: "id required for merge/replace"})
			continue
		}
		items = append(items, it)
	}
	if len(report.Errors) > 0 {
		return report, fmt.Errorf("%d records invalid", len(report.Errors))
	}

	switch mode {
	case ModeReplace:
		if err := repo.ReplaceAll(ctx, items); err != nil {
			return report, err
		}
	case ModeMerge:
		if err := repo.Upsert(ctx, items); err != nil {
			return report, err
		}
	case ModeAppend:
		for _, it := range items {
			if err := repo.Create(ctx, it); err != nil {
				return report, err
			}
		}
	}
	report.Imported = len(items)
	return report, nil
}
