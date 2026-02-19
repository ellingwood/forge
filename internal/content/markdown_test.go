package content

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderBasicMarkdown(t *testing.T) {
	r := NewMarkdownRenderer()

	input := []byte(`# Hello World

This is a **bold** and *italic* paragraph.

[Click here](https://example.com)
`)

	out, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	html := string(out)

	checks := []struct {
		desc    string
		contain string
	}{
		{"h1 heading", "<h1"},
		{"bold text", "<strong>bold</strong>"},
		{"italic text", "<em>italic</em>"},
		{"link href", `href="https://example.com"`},
		{"link tag", "<a "},
		{"paragraph", "<p>"},
	}

	for _, c := range checks {
		if !strings.Contains(html, c.contain) {
			t.Errorf("expected HTML to contain %s (%q), got:\n%s", c.desc, c.contain, html)
		}
	}
}

func TestRenderGFMTables(t *testing.T) {
	r := NewMarkdownRenderer()

	input := []byte(`| Name  | Age |
|-------|-----|
| Alice | 30  |
| Bob   | 25  |
`)

	out, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	html := string(out)

	for _, tag := range []string{"<table>", "<thead>", "<tbody>", "<tr>", "<th>", "<td>"} {
		if !strings.Contains(html, tag) {
			t.Errorf("expected HTML to contain %q, got:\n%s", tag, html)
		}
	}
}

func TestRenderTaskLists(t *testing.T) {
	r := NewMarkdownRenderer()

	input := []byte(`- [x] Done task
- [ ] Pending task
`)

	out, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	html := string(out)

	// Task list checkboxes should be rendered as <input> elements.
	if !strings.Contains(html, `<input`) {
		t.Errorf("expected HTML to contain checkbox <input>, got:\n%s", html)
	}
	if !strings.Contains(html, `checked`) {
		t.Errorf("expected HTML to contain 'checked' for done task, got:\n%s", html)
	}
	if !strings.Contains(html, "type=\"checkbox\"") {
		t.Errorf("expected HTML to contain type=\"checkbox\", got:\n%s", html)
	}
}

func TestRenderCodeBlockHighlighting(t *testing.T) {
	r := NewMarkdownRenderer()

	input := []byte("```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```\n")

	out, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	html := string(out)

	// Syntax highlighting with CSS classes should produce chroma class names.
	if !strings.Contains(html, "chroma") {
		t.Errorf("expected HTML to contain 'chroma' class, got:\n%s", html)
	}
	if !strings.Contains(html, "<pre") {
		t.Errorf("expected HTML to contain <pre>, got:\n%s", html)
	}
	if !strings.Contains(html, "<code") {
		t.Errorf("expected HTML to contain <code>, got:\n%s", html)
	}
}

func TestRenderFootnotes(t *testing.T) {
	r := NewMarkdownRenderer()

	input := []byte(`This has a footnote[^1].

[^1]: This is the footnote content.
`)

	out, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	html := string(out)

	// Footnotes should generate a footnote section with links.
	if !strings.Contains(html, "footnote") {
		t.Errorf("expected HTML to contain 'footnote', got:\n%s", html)
	}
	// The footnote reference should be a link.
	if !strings.Contains(html, "<a") {
		t.Errorf("expected HTML to contain footnote link <a>, got:\n%s", html)
	}
}

func TestRenderHeadingIDs(t *testing.T) {
	r := NewMarkdownRenderer()

	input := []byte(`## My Section

Some content.

### Another Heading
`)

	out, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	html := string(out)

	if !strings.Contains(html, `id="my-section"`) {
		t.Errorf("expected heading to have id=\"my-section\", got:\n%s", html)
	}
	if !strings.Contains(html, `id="another-heading"`) {
		t.Errorf("expected heading to have id=\"another-heading\", got:\n%s", html)
	}
}

func TestRenderWithTOC(t *testing.T) {
	r := NewMarkdownRenderer()

	input := []byte(`# Introduction

Some intro text.

## Getting Started

Setup instructions.

## Configuration

Config details.

### Advanced Options

More details.
`)

	content, tocHTML, err := r.RenderWithTOC(input)
	if err != nil {
		t.Fatalf("RenderWithTOC() error: %v", err)
	}

	// Content should have headings with IDs.
	contentStr := string(content)
	if !strings.Contains(contentStr, `id="introduction"`) {
		t.Errorf("expected content to have id=\"introduction\", got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `id="getting-started"`) {
		t.Errorf("expected content to have id=\"getting-started\", got:\n%s", contentStr)
	}

	// TOC should be non-empty and contain links.
	if len(tocHTML) == 0 {
		t.Fatal("expected non-empty TOC HTML")
	}

	tocStr := string(tocHTML)
	if !strings.Contains(tocStr, "<ul>") {
		t.Errorf("expected TOC to contain <ul>, got:\n%s", tocStr)
	}
	if !strings.Contains(tocStr, "<li>") {
		t.Errorf("expected TOC to contain <li>, got:\n%s", tocStr)
	}
	if !strings.Contains(tocStr, "<a") {
		t.Errorf("expected TOC to contain links, got:\n%s", tocStr)
	}
	if !strings.Contains(tocStr, "#getting-started") {
		t.Errorf("expected TOC to contain link to #getting-started, got:\n%s", tocStr)
	}
}

func TestGenerateChromaCSS(t *testing.T) {
	lightCSS, darkCSS, err := GenerateChromaCSS("monokai", "dracula")
	if err != nil {
		t.Fatalf("GenerateChromaCSS() error: %v", err)
	}

	if len(lightCSS) == 0 {
		t.Error("expected non-empty light CSS")
	}
	if len(darkCSS) == 0 {
		t.Error("expected non-empty dark CSS")
	}

	// Light CSS should contain .chroma selectors.
	if !strings.Contains(lightCSS, ".chroma") {
		t.Errorf("expected light CSS to contain '.chroma', got:\n%s", lightCSS[:min(200, len(lightCSS))])
	}

	// Dark CSS should have .dark prefix on .chroma selectors.
	if !strings.Contains(darkCSS, ".dark") {
		t.Errorf("expected dark CSS to contain '.dark', got:\n%s", darkCSS[:min(200, len(darkCSS))])
	}
	if !strings.Contains(darkCSS, ".dark .chroma") {
		t.Errorf("expected dark CSS to contain '.dark .chroma', got:\n%s", darkCSS[:min(200, len(darkCSS))])
	}

	// Light CSS should NOT have .dark prefix.
	if strings.Contains(lightCSS, ".dark") {
		t.Errorf("expected light CSS to NOT contain '.dark'")
	}
}

func TestRenderRawHTML(t *testing.T) {
	r := NewMarkdownRenderer()

	input := []byte(`Some text before.

<div class="custom">
  <p>Raw HTML content</p>
</div>

Some text after.
`)

	out, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !bytes.Contains(out, []byte(`<div class="custom">`)) {
		t.Errorf("expected raw HTML <div> to pass through, got:\n%s", string(out))
	}
	if !bytes.Contains(out, []byte(`<p>Raw HTML content</p>`)) {
		t.Errorf("expected raw HTML <p> to pass through, got:\n%s", string(out))
	}
}
