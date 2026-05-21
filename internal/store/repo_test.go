package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/darious1472/snibox-next/internal/id"
)

func newTestRepo(t *testing.T) *Repo {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewRepo(db)
}

func strp(s string) *string { return &s }

func mkSnippet(title, body, lang string, tags []string) *Item {
	now := time.Now().UTC()
	return &Item{
		ID:        id.New(),
		Title:     title,
		Body:      body,
		Type:      TypeSnippet,
		Language:  strp(lang),
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func mkNote(title, body string, tags []string) *Item {
	now := time.Now().UTC()
	return &Item{
		ID:        id.New(),
		Title:     title,
		Body:      body,
		Type:      TypeNote,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func mkLink(title, urlStr string, tags []string) *Item {
	now := time.Now().UTC()
	return &Item{
		ID:        id.New(),
		Title:     title,
		Body:      "",
		Type:      TypeLink,
		URL:       strp(urlStr),
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestCreateGet(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	it := mkSnippet("hello.go", "package main", "go", []string{"go", "demo"})
	if err := r.Create(ctx, it); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := r.Get(ctx, it.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "hello.go" || got.Type != TypeSnippet {
		t.Errorf("unexpected item: %+v", got)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "demo" || got.Tags[1] != "go" {
		t.Errorf("tags wrong: %v", got.Tags)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		it      *Item
		wantErr bool
	}{
		{"no title", &Item{Type: TypeNote}, true},
		{"snippet with lang", mkSnippet("a", "b", "go", nil), false},
		{"note with lang", &Item{Title: "n", Type: TypeNote, Language: strp("go")}, true},
		{"link no url", &Item{Title: "x", Type: TypeLink}, true},
		{"link bad url", &Item{Title: "x", Type: TypeLink, URL: strp("not-a-url")}, true},
		{"link ok", mkLink("x", "https://example.com", nil), false},
		{"bad tag", &Item{Title: "x", Type: TypeNote, Tags: []string{"has spaces"}}, true},
		{"upper tag normalised", &Item{Title: "x", Type: TypeNote, Tags: []string{"Go"}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.it.Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("want err=%v, got %v", c.wantErr, err)
			}
		})
	}
}

func TestSearchAcrossFields(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	mustCreate(t, r, mkSnippet("Jellyfin compose", "services:\n  jellyfin:", "yaml", []string{"docker", "media"}))
	mustCreate(t, r, mkNote("Window functions", "SELECT row_number() OVER (PARTITION BY x)", []string{"sql"}))
	mustCreate(t, r, mkLink("htmx docs", "https://htmx.org", []string{"frontend"}))

	cases := []struct {
		q    string
		want int
	}{
		{"jellyfin", 1}, // title
		{"row_number", 1}, // body
		{"docker", 1},    // tag
		{"yaml", 1},      // language
		{"htmx.org", 1},  // url
		{"nope", 0},
	}
	for _, c := range cases {
		t.Run(c.q, func(t *testing.T) {
			items, _, err := r.List(ctx, Filter{View: "search", Query: c.q})
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			if len(items) != c.want {
				t.Errorf("q=%q want %d, got %d", c.q, c.want, len(items))
			}
		})
	}
}

func TestListFiltersAndPinFloat(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	a := mkSnippet("A", "x", "go", []string{"go"})
	b := mkSnippet("B", "y", "go", []string{"go", "pinned-stuff"})
	b.Pinned = true
	c := mkNote("C note", "stuff", []string{"go"})
	d := mkNote("D archived", "stuff", nil)
	d.Archived = true
	for _, it := range []*Item{a, b, c, d} {
		mustCreate(t, r, it)
	}

	t.Run("all view excludes archived", func(t *testing.T) {
		items, _, err := r.List(ctx, Filter{View: "all"})
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 3 {
			t.Errorf("want 3, got %d", len(items))
		}
		if !items[0].Pinned {
			t.Errorf("pinned should sort first; got %s", items[0].Title)
		}
	})

	t.Run("type=note filter", func(t *testing.T) {
		items, _, err := r.List(ctx, Filter{View: "all", Type: "note"})
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Title != "C note" {
			t.Errorf("unexpected: %+v", items)
		}
	})

	t.Run("archive view", func(t *testing.T) {
		items, _, err := r.List(ctx, Filter{View: "archive"})
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Title != "D archived" {
			t.Errorf("unexpected: %+v", items)
		}
	})

	t.Run("tag filter AND", func(t *testing.T) {
		items, _, err := r.List(ctx, Filter{View: "all", Tags: []string{"go", "pinned-stuff"}})
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Title != "B" {
			t.Errorf("want only B, got %+v", titles(items))
		}
	})
}

func TestUpdateAndTagsTriggerSearchBlob(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	it := mkNote("Title", "body content", nil)
	mustCreate(t, r, it)

	// Tag added via AddTag should make search find by tag.
	if err := r.AddTag(ctx, it.ID, "homelab"); err != nil {
		t.Fatalf("add tag: %v", err)
	}
	items, _, err := r.List(ctx, Filter{View: "search", Query: "homelab"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected to find item by added tag, got %d", len(items))
	}

	// Update body should refresh search_blob. Re-fetch to keep tags.
	fresh, err := r.Get(ctx, it.ID)
	if err != nil {
		t.Fatal(err)
	}
	fresh.Body = "totally new prose about kubernetes"
	if err := r.Update(ctx, fresh); err != nil {
		t.Fatalf("update: %v", err)
	}
	items, _, err = r.List(ctx, Filter{View: "search", Query: "kubernetes"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("body update should be searchable, got %d", len(items))
	}
	items, _, _ = r.List(ctx, Filter{View: "search", Query: "body content"})
	if len(items) != 0 {
		t.Errorf("old body should be gone, got %d", len(items))
	}

	if err := r.RemoveTag(ctx, it.ID, "homelab"); err != nil {
		t.Fatalf("remove tag: %v", err)
	}
	items, _, _ = r.List(ctx, Filter{View: "search", Query: "homelab"})
	if len(items) != 0 {
		t.Errorf("removed tag should not match, got %d", len(items))
	}
}

func TestTogglePinArchive(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	it := mkNote("toggle", "x", nil)
	mustCreate(t, r, it)

	got, err := r.TogglePin(ctx, it.ID)
	if err != nil || !got.Pinned {
		t.Fatalf("pin1 err=%v pinned=%v", err, got.Pinned)
	}
	got, err = r.TogglePin(ctx, it.ID)
	if err != nil || got.Pinned {
		t.Fatalf("pin2 err=%v pinned=%v", err, got.Pinned)
	}
	got, err = r.ToggleArchive(ctx, it.ID)
	if err != nil || !got.Archived {
		t.Fatalf("arch err=%v arch=%v", err, got.Archived)
	}
}

func TestDeleteCascadesTags(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	it := mkSnippet("k", "v", "go", []string{"a", "b"})
	mustCreate(t, r, it)

	if err := r.Delete(ctx, it.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := r.Get(ctx, it.ID); err != ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
	var n int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM tags WHERE item_id = ?`, it.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("tags should cascade, found %d", n)
	}
}

func TestPaginationCursor(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		it := mkNote("note "+string(rune('A'+i)), "body", nil)
		// Vary updated_at so order is stable.
		it.UpdatedAt = time.Now().UTC().Add(time.Duration(i) * time.Second)
		mustCreate(t, r, it)
	}
	page1, cur, err := r.List(ctx, Filter{View: "all", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 || cur == "" {
		t.Fatalf("page1 len=%d cur=%q", len(page1), cur)
	}
	page2, cur2, err := r.List(ctx, Filter{View: "all", Limit: 2, Cursor: cur})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 2 || cur2 == "" {
		t.Fatalf("page2 len=%d cur=%q", len(page2), cur2)
	}
	page3, cur3, err := r.List(ctx, Filter{View: "all", Limit: 2, Cursor: cur2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page3) != 1 || cur3 != "" {
		t.Fatalf("page3 len=%d cur=%q", len(page3), cur3)
	}
}

func TestCountsAndTagCounts(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	mustCreate(t, r, mkSnippet("s1", "x", "go", []string{"go"}))
	mustCreate(t, r, mkNote("n1", "y", []string{"go", "homelab"}))
	mustCreate(t, r, mkLink("l1", "https://example.com", []string{"homelab"}))
	pinned := mkNote("pinned", "z", nil)
	pinned.Pinned = true
	mustCreate(t, r, pinned)
	arch := mkNote("arch", "z", []string{"hidden"})
	arch.Archived = true
	mustCreate(t, r, arch)

	c, err := r.Counts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if c.All != 4 || c.Snippets != 1 || c.Notes != 2 || c.Links != 1 || c.Pinned != 1 || c.Archive != 1 {
		t.Errorf("unexpected counts: %+v", c)
	}

	tcs, err := r.TagCounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, tc := range tcs {
		got[tc.Tag] = tc.Count
	}
	if got["go"] != 2 || got["homelab"] != 2 {
		t.Errorf("unexpected tag counts: %+v", got)
	}
	if _, ok := got["hidden"]; ok {
		t.Errorf("archived tags should be excluded")
	}
}

func mustCreate(t *testing.T, r *Repo, it *Item) {
	t.Helper()
	if err := r.Create(context.Background(), it); err != nil {
		t.Fatalf("create %s: %v", it.Title, err)
	}
}

func titles(items []*Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Title
	}
	return out
}

func TestSortByCreatedAndTitle(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	base := time.Now().UTC()
	items := []*Item{
		mkNote("Charlie", "x", nil),
		mkNote("Alpha", "x", nil),
		mkNote("Bravo", "x", nil),
	}
	for i, it := range items {
		it.CreatedAt = base.Add(time.Duration(i) * time.Second)
		it.UpdatedAt = it.CreatedAt
		mustCreate(t, r, it)
	}
	got, _, err := r.List(ctx, Filter{View: "all", Sort: "title"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Title != "Alpha" || got[2].Title != "Charlie" {
		t.Errorf("title sort wrong: %v", titles(got))
	}
	got, _, err = r.List(ctx, Filter{View: "all", Sort: "created"})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Title != "Bravo" { // newest first
		t.Errorf("created sort wrong: %v", titles(got))
	}
}

func TestUpsertAndReplaceAll(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	mustCreate(t, r, mkNote("keep", "k", nil))

	imported := []*Item{
		mkNote("imp1", "b1", []string{"x"}),
		mkSnippet("imp2", "code", "go", []string{"y"}),
	}
	if err := r.Upsert(ctx, imported); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	all, err := r.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Errorf("upsert merge: want 3, got %d", len(all))
	}

	// Now mutate one and upsert by ID.
	imported[0].Title = "imp1-updated"
	if err := r.Upsert(ctx, []*Item{imported[0]}); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	got, err := r.Get(ctx, imported[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "imp1-updated" {
		t.Errorf("upsert update missed: %s", got.Title)
	}

	// Replace.
	replacement := []*Item{mkNote("only", "x", nil)}
	if err := r.ReplaceAll(ctx, replacement); err != nil {
		t.Fatalf("replace: %v", err)
	}
	all, err = r.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Title != "only" {
		t.Errorf("replace failed: %v", titles(all))
	}
}

func TestShortQueryFallback(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	mustCreate(t, r, mkNote("Apple", "body has 'a' in it", nil))
	mustCreate(t, r, mkNote("Other", "definitely no apples here, just the body", nil))

	// 1-char query: title-only LIKE. Both titles contain 'a'? "Apple" yes, "Other" no.
	items, _, err := r.List(ctx, Filter{View: "search", Query: "a"})
	if err != nil {
		t.Fatal(err)
	}
	gotTitles := titles(items)
	wantOnly := []string{"Apple"}
	if len(gotTitles) != len(wantOnly) || gotTitles[0] != wantOnly[0] {
		t.Errorf("short query should hit title only; got %v", gotTitles)
	}
}

func TestAddTagDuplicateIgnored(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	it := mkNote("dup", "x", nil)
	mustCreate(t, r, it)
	if err := r.AddTag(ctx, it.ID, "x"); err != nil {
		t.Fatal(err)
	}
	if err := r.AddTag(ctx, it.ID, "X"); err != nil { // normalised dup
		t.Fatal(err)
	}
	got, _ := r.Get(ctx, it.ID)
	if len(got.Tags) != 1 {
		t.Errorf("expected dedup, got %v", got.Tags)
	}
}

func TestFilterValidation(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	if _, _, err := r.List(ctx, Filter{View: "nope"}); err == nil {
		t.Error("expected error on bad view")
	}
	if _, _, err := r.List(ctx, Filter{View: "all", Sort: "nope"}); err == nil {
		t.Error("expected error on bad sort")
	}
	if _, _, err := r.List(ctx, Filter{View: "all", Cursor: "###"}); err == nil {
		t.Error("expected error on bad cursor")
	}
}

func TestDBAndOpenErrors(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Error("expected error on empty dsn")
	}
	r := newTestRepo(t)
	if r.DB() == nil {
		t.Error("DB() returned nil")
	}
}

func TestNotFoundPaths(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	if err := r.Delete(ctx, "missing"); err != ErrNotFound {
		t.Errorf("delete missing: %v", err)
	}
	if _, err := r.TogglePin(ctx, "missing"); err != ErrNotFound {
		t.Errorf("toggle pin missing: %v", err)
	}
	if _, err := r.ToggleArchive(ctx, "missing"); err != ErrNotFound {
		t.Errorf("toggle arch missing: %v", err)
	}
	if err := r.RemoveTag(ctx, "missing", "x"); err != ErrNotFound {
		t.Errorf("remove tag missing: %v", err)
	}
	if err := r.AddTag(ctx, "missing", "x"); err != ErrNotFound {
		t.Errorf("add tag missing: %v", err)
	}
}

// ensure imports are used in some builds
var _ = strings.Join
