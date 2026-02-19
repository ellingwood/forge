// Package build orchestrates the full static site generation pipeline.
// It coordinates content discovery, markdown rendering, template execution,
// and file output to produce a complete static site.
package build

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/aellingwood/forge/internal/config"
	"github.com/aellingwood/forge/internal/content"
	tmpl "github.com/aellingwood/forge/internal/template"
)

// BuildOptions controls the behaviour of the build pipeline.
type BuildOptions struct {
	IncludeDrafts  bool
	IncludeFuture  bool
	IncludeExpired bool
	OutputDir      string
	Verbose        bool
	Minify         bool
	BaseURL        string
	ProjectRoot    string
}

// BuildResult contains statistics about the completed build.
type BuildResult struct {
	PagesRendered int
	FilesWritten  int
	FilesCopied   int
	Duration      time.Duration
	OutputSize    int64
}

// renderer is the interface used for rendering pages to HTML.
// This decouples the build package from the render package so that
// the build pipeline can be tested independently.
type renderer interface {
	RenderPage(page *content.Page, allPages []*content.Page) ([]byte, error)
}

// Builder coordinates the full static site generation pipeline.
type Builder struct {
	config  *config.SiteConfig
	options BuildOptions
}

// NewBuilder creates a new Builder with the given site configuration and options.
func NewBuilder(cfg *config.SiteConfig, opts BuildOptions) *Builder {
	return &Builder{
		config:  cfg,
		options: opts,
	}
}

// Build executes the full build pipeline and returns a BuildResult summarizing
// what was generated. The pipeline steps are:
//  1. Clean or create the output directory
//  2. Discover content files
//  3. Filter pages (drafts, future, expired)
//  4. Render markdown in parallel
//  5. Build taxonomy maps
//  6. Sort pages and set navigation links
//  7. Create template engine
//  8. Render pages to HTML in parallel
//  9. Write HTML files
//  10. Copy static files
//  11. Copy page bundle assets
func (b *Builder) Build() (*BuildResult, error) {
	start := time.Now()
	result := &BuildResult{}

	projectRoot := b.options.ProjectRoot
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("determining project root: %w", err)
		}
	}

	// Determine output directory.
	outputDir := b.options.OutputDir
	if outputDir == "" {
		outputDir = filepath.Join(projectRoot, "public")
	}
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(projectRoot, outputDir)
	}

	// Determine content directory.
	contentDir := filepath.Join(projectRoot, "content")

	// Determine base URL.
	baseURL := b.options.BaseURL
	if baseURL == "" {
		baseURL = b.config.BaseURL
	}

	// Step 1: Clean output directory.
	if err := CleanDir(outputDir); err != nil {
		return nil, fmt.Errorf("cleaning output directory: %w", err)
	}

	// Step 2: Discover content.
	pages, err := content.Discover(contentDir, b.config)
	if err != nil {
		return nil, fmt.Errorf("discovering content: %w", err)
	}

	// Set absolute permalinks.
	for _, p := range pages {
		p.Permalink = strings.TrimRight(baseURL, "/") + p.URL
	}

	// Step 3: Filter pages based on options.
	if !b.options.IncludeDrafts {
		pages = content.FilterDrafts(pages)
	}
	if !b.options.IncludeFuture {
		pages = content.FilterFuture(pages)
	}
	if !b.options.IncludeExpired {
		pages = content.FilterExpired(pages)
	}

	// Step 4: Render markdown in parallel.
	mdRenderer := content.NewMarkdownRenderer()
	numWorkers := runtime.NumCPU()

	err = renderParallel(pages, numWorkers, func(p *content.Page) error {
		htmlContent, tocHTML, err := mdRenderer.RenderWithTOC([]byte(p.RawContent))
		if err != nil {
			return fmt.Errorf("rendering markdown for %s: %w", p.SourcePath, err)
		}
		p.Content = string(htmlContent)
		p.TableOfContents = string(tocHTML)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("rendering markdown: %w", err)
	}

	// Step 5: Build taxonomy maps.
	tags, categories := buildTaxonomyMaps(pages)

	// Step 6: Sort pages by date (newest first) and set prev/next links.
	content.SortByDate(pages, false)
	setSectionNavigation(pages)

	// Step 7: Create template engine.
	themeName := b.config.Theme
	if themeName == "" {
		themeName = "default"
	}
	themePath := filepath.Join(projectRoot, "themes", themeName)
	userLayoutPath := filepath.Join(projectRoot, "layouts")

	engine, err := tmpl.NewEngine(themePath, userLayoutPath)
	if err != nil {
		return nil, fmt.Errorf("creating template engine: %w", err)
	}

	// Build site context for templates.
	siteCtx := b.buildSiteContext(pages, tags, categories, baseURL)

	// Build page contexts for all pages.
	pageContextMap := b.buildPageContexts(pages, siteCtx)

	// Step 8 & 9: Render pages to HTML in parallel and collect results.
	type renderResult struct {
		url  string
		data []byte
	}
	var mu sync.Mutex
	var results []renderResult

	err = renderParallel(pages, numWorkers, func(p *content.Page) error {
		ctx := pageContextMap[p]
		if ctx == nil {
			return fmt.Errorf("no context for page %s", p.SourcePath)
		}

		// Resolve template.
		templateName := engine.Resolve(p.Type.String(), p.Section, p.Layout)
		if templateName == "" {
			// Use a fallback: wrap content in baseof if available, or output raw content.
			templateName = engine.Resolve("single", "_default", "")
			if templateName == "" {
				// No template found at all, use raw rendered content.
				mu.Lock()
				results = append(results, renderResult{url: p.URL, data: []byte(p.Content)})
				mu.Unlock()
				return nil
			}
		}

		rendered, err := engine.Execute(templateName, ctx)
		if err != nil {
			return fmt.Errorf("executing template %s for %s: %w", templateName, p.SourcePath, err)
		}

		mu.Lock()
		results = append(results, renderResult{url: p.URL, data: rendered})
		mu.Unlock()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("rendering pages: %w", err)
	}

	// Step 10: Write HTML files.
	for _, r := range results {
		if err := WriteFile(outputDir, r.url, r.data); err != nil {
			return nil, fmt.Errorf("writing %s: %w", r.url, err)
		}
		result.FilesWritten++
	}
	result.PagesRendered = len(results)

	// Step 11: Copy static files from theme and site static directories.
	themeStaticDir := filepath.Join(themePath, "static")
	siteStaticDir := filepath.Join(projectRoot, "static")

	if info, err := os.Stat(themeStaticDir); err == nil && info.IsDir() {
		copied, err := copyDirCounting(themeStaticDir, outputDir)
		if err != nil {
			return nil, fmt.Errorf("copying theme static files: %w", err)
		}
		result.FilesCopied += copied
	}

	if info, err := os.Stat(siteStaticDir); err == nil && info.IsDir() {
		copied, err := copyDirCounting(siteStaticDir, outputDir)
		if err != nil {
			return nil, fmt.Errorf("copying site static files: %w", err)
		}
		result.FilesCopied += copied
	}

	// Step 12: Copy page bundle assets.
	for _, p := range pages {
		if !p.IsBundle || len(p.BundleFiles) == 0 {
			continue
		}
		// Determine output directory for this page's assets.
		pageOutputDir := filepath.Join(outputDir, strings.TrimPrefix(p.URL, "/"))
		for _, assetName := range p.BundleFiles {
			src := filepath.Join(p.BundleDir, assetName)
			dst := filepath.Join(pageOutputDir, assetName)
			if err := CopyFile(src, dst); err != nil {
				return nil, fmt.Errorf("copying bundle asset %s: %w", src, err)
			}
			result.FilesCopied++
		}
	}

	// Calculate output size.
	size, err := DirSize(outputDir)
	if err != nil {
		return nil, fmt.Errorf("calculating output size: %w", err)
	}
	result.OutputSize = size
	result.Duration = time.Since(start)

	return result, nil
}

// copyDirCounting copies a directory and returns the number of files copied.
func copyDirCounting(src, dst string) (int, error) {
	count := 0
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}

		if err := CopyFile(path, dstPath); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

// buildSiteContext creates a SiteContext for template rendering.
func (b *Builder) buildSiteContext(
	pages []*content.Page,
	tags map[string][]*content.Page,
	categories map[string][]*content.Page,
	baseURL string,
) *tmpl.SiteContext {
	// Build menu items.
	menuItems := make([]tmpl.MenuItemContext, len(b.config.Menu.Main))
	for i, item := range b.config.Menu.Main {
		menuItems[i] = tmpl.MenuItemContext{
			Name:   item.Name,
			URL:    item.URL,
			Weight: item.Weight,
		}
	}

	// Build section map.
	sections := make(map[string][]*tmpl.PageContext)

	// Build page contexts for site.
	sitePages := make([]*tmpl.PageContext, 0, len(pages))
	for _, p := range pages {
		pc := pageToContext(p, nil) // site will be set after
		sitePages = append(sitePages, pc)
		if p.Section != "" {
			sections[p.Section] = append(sections[p.Section], pc)
		}
	}

	// Build taxonomy contexts.
	taxonomies := make(map[string]map[string][]*tmpl.PageContext)
	if len(tags) > 0 {
		tagMap := make(map[string][]*tmpl.PageContext)
		for term, tagPages := range tags {
			for _, tp := range tagPages {
				tagMap[term] = append(tagMap[term], pageToContext(tp, nil))
			}
		}
		taxonomies["tags"] = tagMap
	}
	if len(categories) > 0 {
		catMap := make(map[string][]*tmpl.PageContext)
		for term, catPages := range categories {
			for _, cp := range catPages {
				catMap[term] = append(catMap[term], pageToContext(cp, nil))
			}
		}
		taxonomies["categories"] = catMap
	}

	return &tmpl.SiteContext{
		Title:       b.config.Title,
		Description: b.config.Description,
		BaseURL:     baseURL,
		Language:    b.config.Language,
		Author: tmpl.AuthorContext{
			Name:   b.config.Author.Name,
			Email:  b.config.Author.Email,
			Bio:    b.config.Author.Bio,
			Avatar: b.config.Author.Avatar,
			Social: tmpl.SocialContext{
				GitHub:   b.config.Author.Social.GitHub,
				LinkedIn: b.config.Author.Social.LinkedIn,
				Twitter:  b.config.Author.Social.Twitter,
				Mastodon: b.config.Author.Social.Mastodon,
				Email:    b.config.Author.Social.Email,
			},
		},
		Menu:       menuItems,
		Params:     b.config.Params,
		Data:       make(map[string]any),
		Pages:      sitePages,
		Sections:   sections,
		Taxonomies: taxonomies,
		BuildDate:  time.Now(),
	}
}

// buildPageContexts creates a map from Page to PageContext for all pages.
func (b *Builder) buildPageContexts(pages []*content.Page, siteCtx *tmpl.SiteContext) map[*content.Page]*tmpl.PageContext {
	m := make(map[*content.Page]*tmpl.PageContext, len(pages))
	for _, p := range pages {
		ctx := pageToContext(p, siteCtx)
		m[p] = ctx
	}

	// Wire up prev/next navigation on page contexts.
	for _, p := range pages {
		ctx := m[p]
		if p.PrevPage != nil {
			if prevCtx, ok := m[p.PrevPage]; ok {
				ctx.PrevPage = prevCtx
			}
		}
		if p.NextPage != nil {
			if nextCtx, ok := m[p.NextPage]; ok {
				ctx.NextPage = nextCtx
			}
		}
	}
	return m
}

// pageToContext converts a content.Page to a template.PageContext.
func pageToContext(p *content.Page, siteCtx *tmpl.SiteContext) *tmpl.PageContext {
	ctx := &tmpl.PageContext{
		Title:           p.Title,
		Description:     p.Description,
		Content:         template.HTML(p.Content),
		Summary:         template.HTML(p.Summary),
		Date:            p.Date,
		Lastmod:         p.Lastmod,
		Draft:           p.Draft,
		Slug:            p.Slug,
		URL:             p.URL,
		Permalink:       p.Permalink,
		ReadingTime:     p.ReadingTime,
		WordCount:       p.WordCount,
		Tags:            p.Tags,
		Categories:      p.Categories,
		Series:          p.Series,
		Params:          p.Params,
		TableOfContents: template.HTML(p.TableOfContents),
		Section:         p.Section,
		Type:            p.Type.String(),
		Site:            siteCtx,
	}

	if p.Cover != nil {
		ctx.Cover = &tmpl.CoverImage{
			Image:   p.Cover.Image,
			Alt:     p.Cover.Alt,
			Caption: p.Cover.Caption,
		}
	}

	return ctx
}
