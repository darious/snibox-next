package importer

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/darious1472/snibox-next/internal/id"
	"github.com/darious1472/snibox-next/internal/store"
)

func newRepo(t *testing.T) *store.Repo {
	t.Helper()
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return store.NewRepo(db)
}

func sample() []Wire {
	t1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	lang := "go"
	urlS := "https://example.com"
	return []Wire{
		{ID: id.New(), Title: "snip1", Body: "package main", Type: "snippet", Language: &lang, Tags: []string{"go"}, CreatedAt: t1, UpdatedAt: t2},
		{ID: id.New(), Title: "note1", Body: "# hi", Type: "note", Tags: []string{"a", "b"}, Pinned: true, CreatedAt: t1, UpdatedAt: t2},
		{ID: id.New(), Title: "link1", Body: "", Type: "link", URL: &urlS, CreatedAt: t1, UpdatedAt: t2},
	}
}

func encode(t *testing.T, w []Wire) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(w); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestReplaceRoundTrip(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	wire := sample()
	rep, err := Run(ctx, r, bytes.NewReader(encode(t, wire)), ModeReplace)
	if err != nil {
		t.Fatalf("import: %v %+v", err, rep)
	}
	if rep.Imported != 3 {
		t.Errorf("want 3 imported, got %d", rep.Imported)
	}
	out, err := r.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("want 3, got %d", len(out))
	}
	// Timestamps preserved.
	want := wire[0].UpdatedAt
	for _, it := range out {
		if !it.UpdatedAt.Equal(want) && !it.UpdatedAt.Equal(wire[1].UpdatedAt) && !it.UpdatedAt.Equal(wire[2].UpdatedAt) {
			t.Errorf("updated_at not preserved: %v", it.UpdatedAt)
		}
	}
}

func TestMergePreservesTimestamps(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	wire := sample()
	_, err := Run(ctx, r, bytes.NewReader(encode(t, wire)), ModeMerge)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	// Update the title on the wire and merge again — created_at should stay.
	wire[0].Title = "snip1-renamed"
	_, err = Run(ctx, r, bytes.NewReader(encode(t, wire[:1])), ModeMerge)
	if err != nil {
		t.Fatalf("import 2: %v", err)
	}
	got, err := r.Get(ctx, wire[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "snip1-renamed" {
		t.Errorf("title not updated: %q", got.Title)
	}
	if !got.CreatedAt.Equal(wire[0].CreatedAt) {
		t.Errorf("created_at not preserved: %v vs %v", got.CreatedAt, wire[0].CreatedAt)
	}
}

func TestAppendRegeneratesIDAndTimes(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	wire := sample()
	origID := wire[0].ID
	origTime := wire[0].UpdatedAt
	rep, err := Run(ctx, r, bytes.NewReader(encode(t, wire)), ModeAppend)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if rep.Imported != 3 {
		t.Errorf("want 3, got %d", rep.Imported)
	}
	out, err := r.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range out {
		if it.ID == origID {
			t.Errorf("append should assign new IDs, found original %s", origID)
		}
		if it.UpdatedAt.Equal(origTime) {
			t.Errorf("append should regenerate timestamps, got %v", it.UpdatedAt)
		}
	}
}

func TestInvalidRecordsAborted(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	bad := []Wire{
		{ID: id.New(), Title: "", Type: "note"}, // no title
	}
	rep, err := Run(ctx, r, bytes.NewReader(encode(t, bad)), ModeReplace)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(rep.Errors) != 1 {
		t.Errorf("expected 1 record err, got %+v", rep.Errors)
	}
}

func TestRoundTripExportImport(t *testing.T) {
	src := newRepo(t)
	ctx := context.Background()
	// Seed via direct store API.
	lang := "yaml"
	for i, title := range []string{"a", "b", "c"} {
		now := time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC)
		it := &store.Item{
			ID:        id.New(),
			Title:     title,
			Body:      "body " + title,
			Type:      store.TypeSnippet,
			Language:  &lang,
			Tags:      []string{"x"},
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := src.Create(ctx, it); err != nil {
			t.Fatal(err)
		}
	}
	items, err := src.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wire := ToWire(items)
	body, _ := json.Marshal(wire)

	dst := newRepo(t)
	_, err = Run(ctx, dst, strings.NewReader(string(body)), ModeReplace)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	got, err := dst.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(items) {
		t.Errorf("len mismatch: %d vs %d", len(got), len(items))
	}
	for i := range got {
		if got[i].Title != items[i].Title || !got[i].CreatedAt.Equal(items[i].CreatedAt) {
			t.Errorf("mismatch at %d: %+v vs %+v", i, got[i], items[i])
		}
	}
}

func TestInvalidMode(t *testing.T) {
	r := newRepo(t)
	if _, err := Run(context.Background(), r, strings.NewReader("[]"), Mode("nope")); err == nil {
		t.Error("expected error on bad mode")
	}
}

func TestDecodeError(t *testing.T) {
	r := newRepo(t)
	if _, err := Run(context.Background(), r, strings.NewReader("not-json"), ModeMerge); err == nil {
		t.Error("expected decode error")
	}
}
