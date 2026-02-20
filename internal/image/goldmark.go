package image

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// DefaultSizes is the default responsive sizes attribute for images rendered
// at the content width (max-w-3xl = 768px).
const DefaultSizes = "(max-width: 768px) 100vw, 768px"

// ResponsiveImageExtension implements goldmark.Extender. It replaces standard
// <img> tags with responsive <picture> elements for images that have been
// processed by the image Processor.
type ResponsiveImageExtension struct {
	processor *Processor
}

// NewResponsiveImageExtension creates a goldmark extension that renders
// processed images as responsive <picture> elements.
func NewResponsiveImageExtension(proc *Processor) *ResponsiveImageExtension {
	return &ResponsiveImageExtension{processor: proc}
}

// Extend registers the responsive image renderer with the goldmark instance.
func (e *ResponsiveImageExtension) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&responsiveImageRenderer{processor: e.processor}, 100),
		),
	)
}

// responsiveImageRenderer renders ast.Image nodes as responsive <picture>
// elements when the image has been processed, or as plain <img> tags otherwise.
type responsiveImageRenderer struct {
	processor *Processor
}

// RegisterFuncs registers the image node renderer.
func (r *responsiveImageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

// renderImage renders an ast.Image node. For processed images it outputs a
// <picture> element with WebP <source> and original-format fallback <img>.
// For unprocessed, external, or SVG images it outputs a plain <img>.
func (r *responsiveImageRenderer) renderImage(
	w util.BufWriter, source []byte, node ast.Node, entering bool,
) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*ast.Image)
	src := string(n.Destination)
	alt := nodeAltText(n, source)

	// Check if this image was processed.
	pi := r.processor.GetImage(src)

	if pi == nil || isExternalURL(src) || isSVG(src) {
		// Render a plain <img> with lazy loading.
		_, _ = fmt.Fprintf(w, `<img src="%s" alt="%s" loading="lazy" decoding="async"`,
			util.EscapeHTML([]byte(src)), util.EscapeHTML([]byte(alt)))
		if n.Title != nil {
			_, _ = fmt.Fprintf(w, ` title="%s"`, util.EscapeHTML(n.Title))
		}
		_, _ = w.WriteString(">")
		return ast.WalkSkipChildren, nil
	}

	// Group variants by format.
	webpVariants := filterVariants(pi.Variants, "webp")
	origVariants := filterVariantsExcluding(pi.Variants, "webp")

	sizes := DefaultSizes

	_, _ = w.WriteString("<picture>\n")

	// WebP <source> if we have WebP variants.
	if len(webpVariants) > 0 {
		_, _ = fmt.Fprintf(w, `  <source type="image/webp" srcset="%s" sizes="%s">`+"\n",
			buildSrcsetFromVariants(webpVariants), sizes)
	}

	// Fallback <img> with original format srcset.
	_, _ = fmt.Fprintf(w, `  <img src="%s"`, util.EscapeHTML([]byte(src)))
	if len(origVariants) > 0 {
		_, _ = fmt.Fprintf(w, ` srcset="%s" sizes="%s"`,
			buildSrcsetFromVariants(origVariants), sizes)
	}
	_, _ = fmt.Fprintf(w, ` alt="%s" width="%d" height="%d" loading="lazy" decoding="async"`,
		util.EscapeHTML([]byte(alt)), pi.Width, pi.Height)
	if n.Title != nil {
		_, _ = fmt.Fprintf(w, ` title="%s"`, util.EscapeHTML(n.Title))
	}
	_, _ = w.WriteString(">\n")
	_, _ = w.WriteString("</picture>")

	return ast.WalkSkipChildren, nil
}

// nodeAltText extracts the alt text from an image node by collecting text from
// its child nodes.
func nodeAltText(n *ast.Image, source []byte) string {
	var buf strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			buf.Write(t.Segment.Value(source))
		}
	}
	return buf.String()
}

// filterVariants returns only those variants matching the given format.
func filterVariants(variants []Variant, format string) []Variant {
	var result []Variant
	for _, v := range variants {
		if v.Format == format {
			result = append(result, v)
		}
	}
	return result
}

// filterVariantsExcluding returns variants that do NOT match the given format.
func filterVariantsExcluding(variants []Variant, excludeFormat string) []Variant {
	var result []Variant
	for _, v := range variants {
		if v.Format != excludeFormat {
			result = append(result, v)
		}
	}
	return result
}

// buildSrcsetFromVariants builds a srcset string from a slice of Variant.
// Format: "url1 320w, url2 640w, ..."
func buildSrcsetFromVariants(variants []Variant) string {
	parts := make([]string, 0, len(variants))
	for _, v := range variants {
		parts = append(parts, fmt.Sprintf("%s %dw", v.URL, v.Width))
	}
	return strings.Join(parts, ", ")
}

// isExternalURL reports whether u is an absolute URL (http or https).
func isExternalURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

// isSVG reports whether the URL points to an SVG file.
func isSVG(u string) bool {
	lower := strings.ToLower(u)
	return strings.HasSuffix(lower, ".svg")
}

// BuildSrcset builds a srcset string from a ProcessedImage, filtering
// variants by format. If format is empty, all non-webp variants are included.
// This is exported for use by the build pipeline.
func BuildSrcset(pi *ProcessedImage, format string) string {
	if pi == nil {
		return ""
	}
	var variants []Variant
	if format == "" {
		variants = filterVariantsExcluding(pi.Variants, "webp")
	} else {
		variants = filterVariants(pi.Variants, format)
	}
	return buildSrcsetFromVariants(variants)
}
