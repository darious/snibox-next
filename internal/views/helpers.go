package views

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/darious1472/snibox-next/internal/store"
)

type View struct {
	Slug, Label, Icon string
}

var NavViews = []View{
	{"all", "All", IconAll},
	{"snippets", "Snippets", IconSnippet},
	{"notes", "Notes", IconNote},
	{"links", "Links", IconLink},
	{"pinned", "Pinned", IconPin},
	{"archive", "Archive", IconArchive},
}

// ViewLabel returns the human label for the current view scope.
func ViewLabel(v string) string {
	for _, vv := range NavViews {
		if vv.Slug == v {
			return vv.Label
		}
	}
	return "All"
}

// CountFor extracts a view's count from store.Counts by slug.
func CountFor(c store.Counts, slug string) int {
	switch slug {
	case "all":
		return c.All
	case "snippets":
		return c.Snippets
	case "notes":
		return c.Notes
	case "links":
		return c.Links
	case "pinned":
		return c.Pinned
	case "archive":
		return c.Archive
	}
	return 0
}

// Relative renders a "5m ago" style timestamp.
func Relative(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	if d < 30*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Format("Jan 2 2006")
}

// PreviewText returns a single-line plain preview for the list row.
func PreviewText(it *store.Item) string {
	switch it.Type {
	case store.TypeLink:
		if it.URL != nil {
			return *it.URL
		}
	}
	body := it.Body
	if len(body) > 160 {
		body = body[:160]
	}
	body = strings.ReplaceAll(body, "\n", " ")
	body = strings.ReplaceAll(body, "\t", " ")
	return body
}

func IsMono(t store.ItemType) bool {
	return t == store.TypeSnippet || t == store.TypeLink
}

// CommonLanguages is the dropdown content for snippet language selection.
var CommonLanguages = []string{
	"bash", "css", "diff", "dockerfile", "go", "html", "java", "javascript",
	"json", "markdown", "nginx", "php", "python", "ruby", "rust", "sql",
	"systemd", "toml", "typescript", "yaml",
}

func DerefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// BuildListURL returns "/partials/list?…" given the filter params.
func BuildListURL(view, q string, tags []string) string {
	v := url.Values{}
	if view != "" {
		v.Set("view", view)
	}
	if q != "" {
		v.Set("q", q)
	}
	for _, t := range tags {
		v.Add("tag", t)
	}
	if len(v) == 0 {
		return "/partials/list"
	}
	return "/partials/list?" + v.Encode()
}

func BuildViewURL(view string) string {
	if view == "" {
		return "/all"
	}
	return "/" + view
}
