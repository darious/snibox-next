package handlers

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/darious1472/snibox-next/internal/id"
	"github.com/darious1472/snibox-next/internal/importer"
	"github.com/darious1472/snibox-next/internal/store"
	"github.com/darious1472/snibox-next/internal/views"
)

type Server struct {
	Repo     *store.Repo
	Static   embed.FS
	ChromaCSS []byte
	ReadOnly bool
}

func New(repo *store.Repo, static embed.FS, chromaCSS []byte, readOnly bool) *Server {
	return &Server{Repo: repo, Static: static, ChromaCSS: chromaCSS, ReadOnly: readOnly}
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(loggingMW)
	r.Use(s.readOnlyMW)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200); w.Write([]byte("ok")) })

	r.Get("/static/css/chroma.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(s.ChromaCSS)
	})
	r.Handle("/static/*", http.FileServer(http.FS(s.Static)))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/all", http.StatusFound)
	})

	r.Get("/partials/list", s.partialList)
	r.Get("/partials/item/{id}", s.partialItem)
	r.Get("/partials/new", s.partialNewModal)

	r.Get("/tag/{tag}", s.pageTag)
	r.Get("/item/{id}", s.pageItem)
	r.Get("/{view}", s.pageView)

	r.Post("/items", s.createItem)
	r.Patch("/items/{id}", s.updateItem)
	r.Delete("/items/{id}", s.deleteItem)
	r.Post("/items/{id}/pin", s.togglePin)
	r.Post("/items/{id}/archive", s.toggleArchive)
	r.Post("/items/{id}/tags", s.addTag)
	r.Delete("/items/{id}/tags/{tag}", s.removeTag)

	r.Get("/export.json", s.exportJSON)
	r.Post("/import", s.importJSON)

	return r
}

func loggingMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func (s *Server) readOnlyMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.ReadOnly {
			next.ServeHTTP(w, r)
			return
		}
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			http.Error(w, "read-only", http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) buildPage(ctx context.Context, view, tag, q string, active *store.Item) (views.PageData, error) {
	filter := store.Filter{View: view, Query: q}
	if tag != "" {
		filter.Tags = []string{tag}
		filter.View = "all"
	}
	if q != "" {
		filter.View = "search"
	}
	items, cur, err := s.Repo.List(ctx, filter)
	if err != nil {
		return views.PageData{}, err
	}
	counts, err := s.Repo.Counts(ctx)
	if err != nil {
		return views.PageData{}, err
	}
	tagCounts, err := s.Repo.TagCounts(ctx)
	if err != nil {
		return views.PageData{}, err
	}
	return views.PageData{
		View:       view,
		Tag:        tag,
		Query:      q,
		Counts:     counts,
		TagCounts:  tagCounts,
		Items:      items,
		NextCursor: cur,
		Active:     active,
		ReadOnly:   s.ReadOnly,
	}, nil
}

func (s *Server) pageView(w http.ResponseWriter, r *http.Request) {
	view := chi.URLParam(r, "view")
	if !validView(view) {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query().Get("q")
	d, err := s.buildPage(r.Context(), view, "", q, nil)
	if err != nil {
		httpErr(w, err)
		return
	}
	views.Page(d).Render(r.Context(), w)
}

func (s *Server) pageTag(w http.ResponseWriter, r *http.Request) {
	tag := store.NormaliseTag(chi.URLParam(r, "tag"))
	d, err := s.buildPage(r.Context(), "all", tag, "", nil)
	if err != nil {
		httpErr(w, err)
		return
	}
	views.Page(d).Render(r.Context(), w)
}

func (s *Server) pageItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	it, err := s.Repo.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		httpErr(w, err)
		return
	}
	d, err := s.buildPage(r.Context(), viewFor(it), "", "", it)
	if err != nil {
		httpErr(w, err)
		return
	}
	views.Page(d).Render(r.Context(), w)
}

func viewFor(it *store.Item) string {
	if it.Archived {
		return "archive"
	}
	switch it.Type {
	case store.TypeSnippet:
		return "snippets"
	case store.TypeNote:
		return "notes"
	case store.TypeLink:
		return "links"
	}
	return "all"
}

func validView(v string) bool {
	switch v {
	case "all", "snippets", "notes", "links", "pinned", "archive", "search":
		return true
	}
	return false
}

// Partials -------------------------------------------------------------------

func (s *Server) partialList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	view := q.Get("view")
	if !validView(view) {
		view = "all"
	}
	query := q.Get("q")
	tags := q["tag"]
	d, err := s.buildPage(r.Context(), view, "", query, nil)
	if err != nil {
		httpErr(w, err)
		return
	}
	if len(tags) > 0 {
		filter := store.Filter{View: view, Query: query, Tags: tags}
		items, _, err := s.Repo.List(r.Context(), filter)
		if err != nil {
			httpErr(w, err)
			return
		}
		d.Items = items
	}
	views.ListPane(d).Render(r.Context(), w)
}

func (s *Server) partialItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	it, err := s.Repo.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		httpErr(w, err)
		return
	}
	if r.URL.Query().Get("edit") == "1" && it.Type == store.TypeSnippet {
		views.SnippetEditor(it).Render(r.Context(), w)
		return
	}
	views.Item(it, s.ReadOnly).Render(r.Context(), w)
}

func (s *Server) partialNewModal(w http.ResponseWriter, r *http.Request) {
	views.NewModal().Render(r.Context(), w)
}

// Mutations ------------------------------------------------------------------

func (s *Server) createItem(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	it := &store.Item{
		ID:        id.New(),
		Title:     strings.TrimSpace(r.FormValue("title")),
		Body:      r.FormValue("body"),
		Type:      store.ItemType(r.FormValue("type")),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if it.Type == store.TypeSnippet {
		if l := strings.TrimSpace(r.FormValue("language")); l != "" {
			it.Language = &l
		}
	}
	if it.Type == store.TypeLink {
		if u := strings.TrimSpace(r.FormValue("url")); u != "" {
			it.URL = &u
		}
	}
	if raw := strings.TrimSpace(r.FormValue("tags")); raw != "" {
		for _, t := range strings.Fields(raw) {
			it.Tags = append(it.Tags, store.NormaliseTag(t))
		}
	}
	if err := s.Repo.Create(r.Context(), it); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Header().Set("HX-Trigger", `{"toast":"Created"}`)
	views.Item(it, s.ReadOnly).Render(r.Context(), w)
}

func (s *Server) updateItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	it, err := s.Repo.Get(r.Context(), itemID)
	if err != nil {
		httpErr(w, err)
		return
	}
	if v := r.FormValue("title"); v != "" {
		it.Title = strings.TrimSpace(v)
	}
	if r.Form.Has("body") {
		it.Body = r.FormValue("body")
	}
	if r.Form.Has("language") && it.Type == store.TypeSnippet {
		l := strings.TrimSpace(r.FormValue("language"))
		if l == "" {
			it.Language = nil
		} else {
			it.Language = &l
		}
	}
	if r.Form.Has("url") && it.Type == store.TypeLink {
		u := strings.TrimSpace(r.FormValue("url"))
		if u == "" {
			it.URL = nil
		} else {
			it.URL = &u
		}
	}
	if err := s.Repo.Update(r.Context(), it); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Header().Set("HX-Trigger", `{"toast":"Saved"}`)
	if r.Header.Get("HX-Target") == "main" || r.Header.Get("HX-Target") == "" {
		views.Item(it, s.ReadOnly).Render(r.Context(), w)
		return
	}
	w.WriteHeader(204)
}

func (s *Server) deleteItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	if err := s.Repo.Delete(r.Context(), itemID); err != nil {
		httpErr(w, err)
		return
	}
	w.Header().Set("HX-Trigger", `{"toast":"Deleted"}`)
	io.WriteString(w, `<div class="main-empty">Item deleted.</div>`)
}

func (s *Server) togglePin(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	it, err := s.Repo.TogglePin(r.Context(), itemID)
	if err != nil {
		httpErr(w, err)
		return
	}
	w.Header().Set("HX-Trigger", `{"toast":"Toggled pin"}`)
	io.WriteString(w, pinButtonHTML(it))
}

func pinButtonHTML(it *store.Item) string {
	cls := "icon-btn"
	if it.Pinned {
		cls = "icon-btn active"
	}
	return fmt.Sprintf(`<button id="pin-btn" class="%s" title="Pin (p)" hx-post="/items/%s/pin" hx-target="closest button" hx-swap="outerHTML">%s</button>`,
		cls, it.ID, views.IconPin)
}

func (s *Server) toggleArchive(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	it, err := s.Repo.ToggleArchive(r.Context(), itemID)
	if err != nil {
		httpErr(w, err)
		return
	}
	w.Header().Set("HX-Trigger", `{"toast":"Toggled archive"}`)
	views.Item(it, s.ReadOnly).Render(r.Context(), w)
}

func (s *Server) addTag(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	tag := store.NormaliseTag(r.FormValue("tag"))
	if err := s.Repo.AddTag(r.Context(), itemID, tag); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	it, _ := s.Repo.Get(r.Context(), itemID)
	views.Item(it, s.ReadOnly).Render(r.Context(), w)
}

func (s *Server) removeTag(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	tag := chi.URLParam(r, "tag")
	if err := s.Repo.RemoveTag(r.Context(), itemID, tag); err != nil {
		httpErr(w, err)
		return
	}
	it, _ := s.Repo.Get(r.Context(), itemID)
	views.Item(it, s.ReadOnly).Render(r.Context(), w)
}

// Import / export -----------------------------------------------------------

func (s *Server) exportJSON(w http.ResponseWriter, r *http.Request) {
	items, err := s.Repo.ListAll(r.Context())
	if err != nil {
		httpErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="snibox-export.json"`)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(importer.ToWire(items)); err != nil {
		httpErr(w, err)
	}
}

func (s *Server) importJSON(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "merge"
	}
	body := r.Body
	if ct := r.Header.Get("Content-Type"); strings.HasPrefix(ct, "multipart/") {
		f, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		defer f.Close()
		body = f
	}
	report, err := importer.Run(r.Context(), s.Repo, body, importer.Mode(mode))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(report)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func httpErr(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
