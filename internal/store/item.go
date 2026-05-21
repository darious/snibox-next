package store

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type ItemType string

const (
	TypeSnippet ItemType = "snippet"
	TypeNote    ItemType = "note"
	TypeLink    ItemType = "link"
)

func (t ItemType) Valid() bool {
	switch t {
	case TypeSnippet, TypeNote, TypeLink:
		return true
	}
	return false
}

type Item struct {
	ID        string
	Title     string
	Body      string
	Type      ItemType
	Language  *string
	URL       *string
	Tags      []string
	Pinned    bool
	Archived  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

var tagRegexp = regexp.MustCompile(`^[a-z0-9_\-]+$`)

const (
	MaxTitle   = 200
	MaxTagsLen = 32
)

func NormaliseTag(t string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(t, "#")))
}

func ValidateTag(t string) error {
	if t == "" {
		return errors.New("tag empty")
	}
	if !tagRegexp.MatchString(t) {
		return fmt.Errorf("tag %q must match [a-z0-9_-]+", t)
	}
	return nil
}

func (i *Item) Validate() error {
	if i.Title == "" {
		return errors.New("title required")
	}
	if len(i.Title) > MaxTitle {
		return fmt.Errorf("title exceeds %d chars", MaxTitle)
	}
	if !i.Type.Valid() {
		return fmt.Errorf("invalid type %q", i.Type)
	}
	if i.Type != TypeSnippet && i.Language != nil && *i.Language != "" {
		return errors.New("language only allowed for snippets")
	}
	if i.Type != TypeLink && i.URL != nil && *i.URL != "" {
		return errors.New("url only allowed for links")
	}
	if i.Type == TypeLink {
		if i.URL == nil || *i.URL == "" {
			return errors.New("link requires url")
		}
		u, err := url.Parse(*i.URL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("invalid url %q", *i.URL)
		}
	}
	if len(i.Tags) > MaxTagsLen {
		return fmt.Errorf("too many tags (max %d)", MaxTagsLen)
	}
	seen := make(map[string]bool, len(i.Tags))
	for idx, t := range i.Tags {
		n := NormaliseTag(t)
		if err := ValidateTag(n); err != nil {
			return err
		}
		if seen[n] {
			return fmt.Errorf("duplicate tag %q", n)
		}
		seen[n] = true
		i.Tags[idx] = n
	}
	return nil
}
