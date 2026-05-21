package markdown

import (
	"bytes"
	"html/template"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	chromalexers "github.com/alecthomas/chroma/v2/lexers"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Strikethrough,
		extension.Table,
		extension.Linkify,
	),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithXHTML(),
	),
)

// RenderMarkdown converts markdown body to HTML. Safe to embed via templ.Raw.
func RenderMarkdown(body string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(body), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

// HighlightCode runs chroma in HTML "classes" mode. The corresponding CSS is
// emitted via WriteStylesheet at startup so the page itself is JS-free.
func HighlightCode(body, lexerName string) (template.HTML, error) {
	lexer := chromalexers.Get(lexerName)
	if lexer == nil {
		lexer = chromalexers.Fallback
	}
	iterator, err := lexer.Tokenise(nil, body)
	if err != nil {
		return "", err
	}
	formatter := chromahtml.New(
		chromahtml.WithClasses(true),
		chromahtml.PreventSurroundingPre(false),
		chromahtml.LineNumbersInTable(false),
	)
	var buf bytes.Buffer
	if err := formatter.Format(&buf, chromaStyle, iterator); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

// StripMarkdown returns a one-line plain-text preview of a markdown body.
// Used by list-row preview, not by the main pane.
func StripMarkdown(body string) string {
	out := bytes.Buffer{}
	skip := false
	for _, r := range body {
		switch r {
		case '`':
			skip = !skip
		case '#', '*', '_', '>', '[', ']', '(', ')':
			// drop
		case '\n':
			out.WriteByte(' ')
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}
