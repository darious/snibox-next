package handlers

import (
	"context"
	"embed"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/darious1472/snibox-next/internal/id"
	"github.com/darious1472/snibox-next/internal/store"
)

var emptyFS embed.FS

func newServer(t *testing.T, readOnly bool) (*Server, *httptest.Server) {
	t.Helper()
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s := New(store.NewRepo(db), emptyFS, []byte("/* test */"), readOnly)
	ts := httptest.NewServer(s.Routes())
	t.Cleanup(ts.Close)
	return s, ts
}

func seed(t *testing.T, s *Server) (snippet, note, link *store.Item) {
	t.Helper()
	now := time.Now().UTC()
	lang := "go"
	urlS := "https://example.com"
	snippet = &store.Item{ID: id.New(), Title: "hello.go", Body: "package main", Type: store.TypeSnippet, Language: &lang, Tags: []string{"go"}, CreatedAt: now, UpdatedAt: now}
	note = &store.Item{ID: id.New(), Title: "a note", Body: "## hi", Type: store.TypeNote, Tags: []string{"prose"}, CreatedAt: now, UpdatedAt: now}
	link = &store.Item{ID: id.New(), Title: "ex", Body: "", Type: store.TypeLink, URL: &urlS, CreatedAt: now, UpdatedAt: now}
	for _, it := range []*store.Item{snippet, note, link} {
		if err := s.Repo.Create(context.Background(), it); err != nil {
			t.Fatal(err)
		}
	}
	return
}

func TestHealthz(t *testing.T) {
	_, ts := newServer(t, false)
	res, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Errorf("status %d", res.StatusCode)
	}
}

func TestPageViewRoutes(t *testing.T) {
	s, ts := newServer(t, false)
	seed(t, s)
	for _, p := range []string{"/all", "/snippets", "/notes", "/links", "/pinned", "/archive"} {
		res, err := http.Get(ts.URL + p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if res.StatusCode != 200 {
			t.Errorf("%s status %d", p, res.StatusCode)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if !strings.Contains(strings.ToLower(string(body)), "<!doctype html>") {
			t.Errorf("%s did not return full HTML", p)
		}
	}
}

func TestRedirectRoot(t *testing.T) {
	_, ts := newServer(t, false)
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := c.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 302 || res.Header.Get("Location") != "/all" {
		t.Errorf("expected redirect to /all, got %d %s", res.StatusCode, res.Header.Get("Location"))
	}
}

func TestCreateUpdateDelete(t *testing.T) {
	_, ts := newServer(t, false)
	// Create
	form := url.Values{"title": {"new-snippet"}, "type": {"snippet"}, "language": {"go"}, "body": {"package x"}, "tags": {"a b"}}
	res, err := http.PostForm(ts.URL+"/items", form)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("create %d: %s", res.StatusCode, string(b))
	}
	// Find the new item via export.
	res, err = http.Get(ts.URL + "/export.json")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var wire []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&wire); err != nil {
		t.Fatal(err)
	}
	if len(wire) != 1 {
		t.Fatalf("expected 1 item exported, got %d", len(wire))
	}
	idStr := wire[0]["id"].(string)
	// Update title.
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/items/"+idStr,
		strings.NewReader(url.Values{"title": {"renamed"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Errorf("patch %d", res.StatusCode)
	}
	// Pin.
	res, err = http.Post(ts.URL+"/items/"+idStr+"/pin", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Errorf("pin %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "active") {
		t.Errorf("expected active class after pin, got %s", string(body))
	}
	// Add tag.
	res, err = http.PostForm(ts.URL+"/items/"+idStr+"/tags", url.Values{"tag": {"extra"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Errorf("addTag %d", res.StatusCode)
	}
	// Remove tag.
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/items/"+idStr+"/tags/extra", nil)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Errorf("rmTag %d", res.StatusCode)
	}
	// Delete.
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/items/"+idStr, nil)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Errorf("delete %d", res.StatusCode)
	}
}

func TestInvalidCreate(t *testing.T) {
	_, ts := newServer(t, false)
	// Empty title.
	res, _ := http.PostForm(ts.URL+"/items", url.Values{"title": {""}, "type": {"note"}})
	if res.StatusCode != 400 {
		t.Errorf("want 400, got %d", res.StatusCode)
	}
	// Link without URL.
	res, _ = http.PostForm(ts.URL+"/items", url.Values{"title": {"x"}, "type": {"link"}})
	if res.StatusCode != 400 {
		t.Errorf("want 400, got %d", res.StatusCode)
	}
}

func TestPartialList(t *testing.T) {
	s, ts := newServer(t, false)
	snip, _, _ := seed(t, s)
	res, err := http.Get(ts.URL + "/partials/list?view=snippets")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), snip.Title) {
		t.Errorf("snippet not in list output: %s", string(body))
	}
}

func TestPartialItem(t *testing.T) {
	s, ts := newServer(t, false)
	_, note, _ := seed(t, s)
	res, err := http.Get(ts.URL + "/partials/item/" + note.ID)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "note-grid") {
		t.Errorf("note body missing: %s", string(body))
	}
	// 404 on missing.
	res, _ = http.Get(ts.URL + "/partials/item/nonexistent")
	if res.StatusCode != 404 {
		t.Errorf("want 404 got %d", res.StatusCode)
	}
}

func TestReadOnlyBlocksMutations(t *testing.T) {
	_, ts := newServer(t, true)
	res, _ := http.PostForm(ts.URL+"/items", url.Values{"title": {"x"}, "type": {"note"}})
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", res.StatusCode)
	}
	// Reads still work.
	res, _ = http.Get(ts.URL + "/all")
	if res.StatusCode != 200 {
		t.Errorf("read failed in read-only: %d", res.StatusCode)
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	s, ts := newServer(t, false)
	seed(t, s)
	res, _ := http.Get(ts.URL + "/export.json")
	exported, _ := io.ReadAll(res.Body)

	// Fresh server.
	_, ts2 := newServer(t, false)
	res, err := http.Post(ts2.URL+"/import?mode=replace", "application/json", strings.NewReader(string(exported)))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		t.Errorf("import status %d: %s", res.StatusCode, string(body))
	}
	var rep map[string]any
	json.Unmarshal(body, &rep)
	if int(rep["imported"].(float64)) != 3 {
		t.Errorf("expected 3 imported, got %v", rep["imported"])
	}
}

func TestTagPage(t *testing.T) {
	s, ts := newServer(t, false)
	seed(t, s)
	res, err := http.Get(ts.URL + "/tag/go")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Errorf("status %d", res.StatusCode)
	}
}

func TestNewModalPartial(t *testing.T) {
	_, ts := newServer(t, false)
	res, _ := http.Get(ts.URL + "/partials/new")
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "modal") {
		t.Errorf("modal partial empty: %s", string(body))
	}
}
