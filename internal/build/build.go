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
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aellingwood/forge/internal/config"
	"github.com/aellingwood/forge/internal/content"
	"github.com/aellingwood/forge/internal/feed"
	"github.com/aellingwood/forge/internal/search"
	"github.com/aellingwood/forge/internal/seo"
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
	PagesRendered  int
	FilesWritten   int
	FilesCopied    int
	StaticFiles    int
	Duration       time.Duration
	OutputSize     int64
	Pages          []string // URL paths of all rendered pages
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
//  11. Build Tailwind CSS
//  12. Copy page bundle assets
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

	// Load data files from data/ directory.
	dataDir := filepath.Join(projectRoot, "data")
	dataFiles, err := content.LoadDataFiles(dataDir)
	if err != nil {
		return nil, fmt.Errorf("loading data files: %w", err)
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

	// Inject a virtual home page if none was discovered (i.e., no content/_index.md).
	// This ensures public/index.html is always generated.
	if !hasHomePage(pages) {
		pages = append(pages, &content.Page{
			Type: content.PageTypeHome,
			URL:  "/",
		})
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

	// Step 4b: Generate summaries, word counts, and reading times.
	for _, p := range pages {
		// Calculate word count and reading time from plain text content.
		plainText := content.StripHTMLTags(p.Content)
		p.WordCount = content.CalculateWordCount(plainText)
		p.ReadingTime = content.CalculateReadingTime(plainText)

		// Generate summary if not already set from frontmatter.
		if p.Summary == "" {
			p.Summary = content.GenerateSummary(p.RawContent, p.Content, 300)
		}
	}

	// Step 5: Build taxonomy maps.
	tags, categories := buildTaxonomyMaps(pages)

	// Step 5b: Generate taxonomy virtual pages.
	if b.config.Taxonomies != nil {
		taxonomies := content.BuildTaxonomies(pages, b.config.Taxonomies)
		taxPages := content.GenerateTaxonomyPages(taxonomies)
		// Set permalinks on taxonomy pages.
		for _, tp := range taxPages {
			tp.Permalink = strings.TrimRight(baseURL, "/") + tp.URL
		}
		pages = append(pages, taxPages...)
	}

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
	siteCtx := b.buildSiteContext(pages, tags, categories, baseURL, dataFiles)

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
		result.Pages = append(result.Pages, r.url)
	}
	result.PagesRendered = len(results)

	// Step 10b: Generate 404.html using theme template if available.
	notFoundTemplate := engine.Resolve("404", "", "")
	if notFoundTemplate != "" {
		notFoundCtx := &tmpl.PageContext{
			Title: "Page Not Found",
			Site:  siteCtx,
		}
		rendered404, err := engine.Execute(notFoundTemplate, notFoundCtx)
		if err != nil {
			return nil, fmt.Errorf("rendering 404 page: %w", err)
		}
		if err := WriteFile(outputDir, "/404.html", rendered404); err != nil {
			return nil, fmt.Errorf("writing 404.html: %w", err)
		}
		result.FilesWritten++
	}

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

	// Step 11: Build Tailwind CSS.
	cssInput := filepath.Join(themePath, "static", "css", "globals.css")
	if _, err := os.Stat(cssInput); err == nil {
		cssOutput := filepath.Join(outputDir, "css", "style.css")
		contentPaths := []string{
			filepath.Join(themePath, "layouts", "**", "*.html"),
			filepath.Join(projectRoot, "layouts", "**", "*.html"),
			filepath.Join(contentDir, "**", "*.md"),
		}
		tb := &TailwindBuilder{}
		twConfig := filepath.Join(themePath, "tailwind.config.js")
		if _, err := os.Stat(twConfig); err == nil {
			tb.ConfigPath = twConfig
		}
		if _, binErr := tb.EnsureBinary(TailwindVersion); binErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not download Tailwind CSS binary: %v (skipping CSS compilation)\n", binErr)
		} else {
			if err := os.MkdirAll(filepath.Dir(cssOutput), 0o755); err != nil {
				return nil, fmt.Errorf("creating CSS output directory: %w", err)
			}
			if err := tb.Build(cssInput, cssOutput, contentPaths); err != nil {
				return nil, fmt.Errorf("building Tailwind CSS: %w", err)
			}
			result.StaticFiles++
		}
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

	// Step 13: Generate ancillary files (sitemap, robots, feeds, search index, aliases).

	// Collect non-draft pages for sitemap and search.
	var nonDraftPages []*content.Page
	for _, p := range pages {
		if !p.Draft {
			nonDraftPages = append(nonDraftPages, p)
		}
	}

	// Generate sitemap.xml.
	sitemapEntries := make([]seo.SitemapEntry, 0, len(nonDraftPages))
	for _, p := range nonDraftPages {
		sitemapEntries = append(sitemapEntries, seo.SitemapEntry{
			URL:     p.Permalink,
			Lastmod: p.Lastmod,
		})
	}
	sitemapData, err := seo.GenerateSitemap(sitemapEntries)
	if err != nil {
		return nil, fmt.Errorf("generating sitemap: %w", err)
	}
	if err := writeDirectFile(outputDir, "sitemap.xml", sitemapData); err != nil {
		return nil, fmt.Errorf("writing sitemap.xml: %w", err)
	}
	result.StaticFiles++

	// Generate robots.txt.
	sitemapURL := strings.TrimRight(baseURL, "/") + "/sitemap.xml"
	robotsData := seo.GenerateRobotsTxt(sitemapURL)
	if err := writeDirectFile(outputDir, "robots.txt", robotsData); err != nil {
		return nil, fmt.Errorf("writing robots.txt: %w", err)
	}
	result.StaticFiles++

	// Collect blog posts for feeds (non-draft, section == "blog" or configured sections, sorted by date desc).
	feedSections := b.config.Feeds.Sections
	if len(feedSections) == 0 {
		feedSections = []string{"blog"}
	}
	var feedPages []*content.Page
	for _, p := range nonDraftPages {
		if slices.Contains(feedSections, p.Section) {
			feedPages = append(feedPages, p)
		}
	}
	sort.SliceStable(feedPages, func(i, j int) bool {
		return feedPages[i].Date.After(feedPages[j].Date)
	})

	// Convert pages to FeedItems.
	feedItems := make([]feed.FeedItem, 0, len(feedPages))
	for _, p := range feedPages {
		feedItems = append(feedItems, feed.FeedItem{
			Title:       p.Title,
			Link:        p.Permalink,
			Description: p.Summary,
			Content:     p.Content,
			Author:      p.Author,
			PubDate:     p.Date,
			GUID:        p.Permalink,
			Categories:  append(p.Tags, p.Categories...),
		})
	}

	feedOpts := feed.FeedOptions{
		Title:       b.config.Title,
		Description: b.config.Description,
		Link:        strings.TrimRight(baseURL, "/"),
		Language:    b.config.Language,
		Author:      b.config.Author.Name,
		MaxItems:    b.config.Feeds.Limit,
		FullContent: b.config.Feeds.FullContent,
	}

	// Generate RSS feed (index.xml).
	if b.config.Feeds.RSS {
		feedOpts.FeedLink = strings.TrimRight(baseURL, "/") + "/index.xml"
		rssData, err := feed.GenerateRSS(feedItems, feedOpts)
		if err != nil {
			return nil, fmt.Errorf("generating RSS feed: %w", err)
		}
		if err := writeDirectFile(outputDir, "index.xml", rssData); err != nil {
			return nil, fmt.Errorf("writing index.xml: %w", err)
		}
		result.StaticFiles++
	}

	// Generate Atom feed (atom.xml).
	if b.config.Feeds.Atom {
		feedOpts.FeedLink = strings.TrimRight(baseURL, "/") + "/atom.xml"
		atomData, err := feed.GenerateAtom(feedItems, feedOpts)
		if err != nil {
			return nil, fmt.Errorf("generating Atom feed: %w", err)
		}
		if err := writeDirectFile(outputDir, "atom.xml", atomData); err != nil {
			return nil, fmt.Errorf("writing atom.xml: %w", err)
		}
		result.StaticFiles++
	}

	// Generate search index (search-index.json).
	if b.config.Search.Enabled {
		maxContentLen := b.config.Search.ContentLength
		if maxContentLen <= 0 {
			maxContentLen = 5000
		}
		indexEntries := make([]search.IndexEntry, 0, len(nonDraftPages))
		for _, p := range nonDraftPages {
			strippedContent := search.StripHTML(p.Content)
			indexEntries = append(indexEntries, search.IndexEntry{
				Title:      p.Title,
				URL:        p.URL,
				Tags:       p.Tags,
				Categories: p.Categories,
				Summary:    content.StripHTMLTags(p.Summary),
				Content:    strippedContent,
			})
		}
		searchData, err := search.GenerateIndex(indexEntries, maxContentLen)
		if err != nil {
			return nil, fmt.Errorf("generating search index: %w", err)
		}
		if err := writeDirectFile(outputDir, "search-index.json", searchData); err != nil {
			return nil, fmt.Errorf("writing search-index.json: %w", err)
		}
		result.StaticFiles++
	}

	// Generate alias redirect pages.
	var aliases []AliasPage
	for _, p := range pages {
		for _, alias := range p.Aliases {
			aliases = append(aliases, AliasPage{
				AliasURL:     alias,
				CanonicalURL: p.URL,
			})
		}
	}
	if len(aliases) > 0 {
		aliasFiles := GenerateAliasPages(aliases)
		for filePath, htmlData := range aliasFiles {
			fullPath := filepath.Join(outputDir, filePath)
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("creating alias directory %s: %w", dir, err)
			}
			if err := os.WriteFile(fullPath, htmlData, 0o644); err != nil {
				return nil, fmt.Errorf("writing alias file %s: %w", fullPath, err)
			}
			result.StaticFiles++
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

// writeDirectFile writes data to a named file directly in the output directory.
func writeDirectFile(outputDir, filename string, data []byte) error {
	filePath := filepath.Join(outputDir, filename)
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	return os.WriteFile(filePath, data, 0o644)
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
	dataFiles map[string]any,
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
		Data:       dataFiles,
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

// hasHomePage reports whether any page in the slice has PageTypeHome.
func hasHomePage(pages []*content.Page) bool {
	for _, p := range pages {
		if p.Type == content.PageTypeHome {
			return true
		}
	}
	return false
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
