package content

import (
	"bytes"
	"fmt"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"go.abhg.dev/goldmark/toc"
)

// MarkdownRenderer converts Markdown source into HTML using goldmark with
// a rich set of extensions (GFM, footnotes, typographer, syntax highlighting,
// auto heading IDs, and attributes).
type MarkdownRenderer struct {
	md goldmark.Markdown
}

// NewMarkdownRenderer creates a MarkdownRenderer configured with all
// standard extensions enabled.
func NewMarkdownRenderer() *MarkdownRenderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.Typographer,
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	return &MarkdownRenderer{md: md}
}

// Render converts Markdown source bytes into HTML.
func (r *MarkdownRenderer) Render(source []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := r.md.Convert(source, &buf); err != nil {
		return nil, fmt.Errorf("markdown render: %w", err)
	}
	return buf.Bytes(), nil
}

// RenderWithTOC converts Markdown source bytes into HTML and also produces
// a table of contents as a nested HTML list. It returns the rendered content
// HTML and the TOC HTML separately.
func (r *MarkdownRenderer) RenderWithTOC(source []byte) (htmlOut []byte, tocOut []byte, err error) {
	// Parse the markdown into an AST.
	doc := r.md.Parser().Parse(text.NewReader(source))

	// Extract the TOC tree from the AST.
	tocTree, err := toc.Inspect(doc, source)
	if err != nil {
		return nil, nil, fmt.Errorf("toc inspect: %w", err)
	}

	// Render the TOC as an HTML list.
	tocList := toc.RenderList(tocTree)
	if tocList != nil {
		var tocBuf bytes.Buffer
		if err := r.md.Renderer().Render(&tocBuf, source, tocList); err != nil {
			return nil, nil, fmt.Errorf("toc render: %w", err)
		}
		tocOut = tocBuf.Bytes()
	}

	// Render the full document.
	var contentBuf bytes.Buffer
	if err := r.md.Renderer().Render(&contentBuf, source, doc); err != nil {
		return nil, nil, fmt.Errorf("markdown render: %w", err)
	}

	return contentBuf.Bytes(), tocOut, nil
}

// GenerateChromaCSS produces CSS for syntax-highlighted code blocks.
// It returns separate CSS strings for light and dark themes. The dark CSS
// has all .chroma selectors prefixed with .dark so it can be scoped to a
// dark mode class on the document.
func GenerateChromaCSS(lightStyle, darkStyle string) (lightCSS string, darkCSS string, err error) {
	formatter := chromahtml.New(chromahtml.WithClasses(true))

	// Generate light CSS.
	lightSty := styles.Get(lightStyle)
	var lightBuf bytes.Buffer
	if err := formatter.WriteCSS(&lightBuf, lightSty); err != nil {
		return "", "", fmt.Errorf("generate light CSS: %w", err)
	}
	lightCSS = lightBuf.String()

	// Generate dark CSS.
	darkSty := styles.Get(darkStyle)
	var darkBuf bytes.Buffer
	if err := formatter.WriteCSS(&darkBuf, darkSty); err != nil {
		return "", "", fmt.Errorf("generate dark CSS: %w", err)
	}

	// Prefix every .chroma selector with .dark to scope it.
	darkCSS = strings.ReplaceAll(darkBuf.String(), ".chroma", ".dark .chroma")

	return lightCSS, darkCSS, nil
}
