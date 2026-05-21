package markdown

import (
	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

// chromaStyle is used at runtime for tokenisation; it doesn't matter for the
// emitted HTML (classes mode), but Format requires one.
var chromaStyle = func() *chroma.Style {
	if s := styles.Get("onedark"); s != nil {
		return s
	}
	return styles.Fallback
}()

// WriteStylesheet returns the chroma CSS for the bundled style. Called once at
// boot and served from /static/chroma.css.
func WriteStylesheet() (string, error) {
	formatter := chromahtml.New(chromahtml.WithClasses(true))
	var sb stringBuilder
	if err := formatter.WriteCSS(&sb, chromaStyle); err != nil {
		return "", err
	}
	return sb.String(), nil
}

type stringBuilder struct{ s []byte }

func (b *stringBuilder) Write(p []byte) (int, error) { b.s = append(b.s, p...); return len(p), nil }
func (b *stringBuilder) String() string              { return string(b.s) }
