package mcpserver

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aellingwood/forge/internal/build"
	"github.com/aellingwood/forge/internal/content"
	"github.com/aellingwood/forge/internal/scaffold"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (fs *ForgeServer) registerTools() {
	mcp.AddTool(fs.server, &mcp.Tool{
		Name:        "query_content",
		Description: "Filter and sort content pages by section, tags, date, draft status, and more",
	}, fs.handleQueryContent)

	mcp.AddTool(fs.server, &mcp.Tool{
		Name:        "get_page",
		Description: "Get full detail for a single page by path or URL",
	}, fs.handleGetPage)

	mcp.AddTool(fs.server, &mcp.Tool{
		Name:        "list_drafts",
		Description: "List all draft content across all sections",
	}, fs.handleListDrafts)

	mcp.AddTool(fs.server, &mcp.Tool{
		Name:        "validate_frontmatter",
		Description: "Validate a frontmatter YAML string against the Forge schema",
	}, fs.handleValidateFrontmatter)

	mcp.AddTool(fs.server, &mcp.Tool{
		Name:        "get_template_context",
		Description: "Show what data a specific template receives at render time",
	}, fs.handleGetTemplateContext)

	mcp.AddTool(fs.server, &mcp.Tool{
		Name:        "resolve_layout",
		Description: "Show which layout file a given content page will use",
	}, fs.handleResolveLayout)

	mcp.AddTool(fs.server, &mcp.Tool{
		Name:        "create_content",
		Description: "Scaffold a new content file with valid frontmatter",
	}, fs.handleCreateContent)

	mcp.AddTool(fs.server, &mcp.Tool{
		Name:        "build_site",
		Description: "Trigger a full site build and return structured results",
	}, fs.handleBuildSite)

	mcp.AddTool(fs.server, &mcp.Tool{
		Name:        "deploy_site",
		Description: "Deploy the site to S3 + CloudFront",
	}, fs.handleDeploySite)
}

func (fs *ForgeServer) handleQueryContent(ctx context.Context, req *mcp.CallToolRequest, input QueryContentInput) (*mcp.CallToolResult, QueryContentOutput, error) {
	sc, err := fs.ctx.Load()
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, QueryContentOutput{}, nil
	}

	sc.mu.RLock()
	pages := sc.pages
	sc.mu.RUnlock()

	// Validate section
	if input.Section != "" && !sc.HasSection(input.Section) {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Unknown section %q. Available sections: %s", input.Section, strings.Join(sc.SectionNames(), ", ")),
			}},
		}, QueryContentOutput{}, nil
	}

	// Apply filters
	filtered := filterPages(pages, input)

	// Sort
	sortBy := cmp.Or(input.SortBy, "date")
	sortOrder := cmp.Or(input.SortOrder, "desc")
	sortPages(filtered, sortBy, sortOrder)

	// Paginate
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	total := len(filtered)
	filtered = paginatePages(filtered, input.Offset, limit)

	briefs := make([]PageBrief, len(filtered))
	for i, p := range filtered {
		briefs[i] = toPageBrief(p)
	}

	return nil, QueryContentOutput{
		TotalMatches: total,
		Offset:       input.Offset,
		Limit:        limit,
		Pages:        briefs,
	}, nil
}

func (fs *ForgeServer) handleGetPage(ctx context.Context, req *mcp.CallToolRequest, input GetPageInput) (*mcp.CallToolResult, PageDetail, error) {
	if input.Path == "" && input.URL == "" {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "either path or url is required"}}}, PageDetail{}, nil
	}

	sc, err := fs.ctx.Load()
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, PageDetail{}, nil
	}

	sc.mu.RLock()
	pages := sc.pages
	sc.mu.RUnlock()

	for _, p := range pages {
		if (input.Path != "" && matchPagePath(p.SourcePath, input.Path)) ||
			(input.URL != "" && p.URL == input.URL) {
			return nil, toPageDetail(p), nil
		}
	}

	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("page not found: %s", cmp.Or(input.Path, input.URL))}},
	}, PageDetail{}, nil
}

func (fs *ForgeServer) handleListDrafts(ctx context.Context, req *mcp.CallToolRequest, input ListDraftsInput) (*mcp.CallToolResult, ListDraftsOutput, error) {
	sc, err := fs.ctx.Load()
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, ListDraftsOutput{}, nil
	}

	sc.mu.RLock()
	pages := sc.pages
	sc.mu.RUnlock()

	var drafts []PageBrief
	for _, p := range pages {
		if !p.Draft {
			continue
		}
		if input.Section != "" && p.Section != input.Section {
			continue
		}
		drafts = append(drafts, toPageBrief(p))
	}
	if drafts == nil {
		drafts = []PageBrief{}
	}
	return nil, ListDraftsOutput{TotalDrafts: len(drafts), Drafts: drafts}, nil
}

func (fs *ForgeServer) handleValidateFrontmatter(ctx context.Context, req *mcp.CallToolRequest, input ValidateFrontmatterInput) (*mcp.CallToolResult, ValidateFrontmatterOutput, error) {
	sc, err := fs.ctx.Load()
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, ValidateFrontmatterOutput{}, nil
	}

	sc.mu.RLock()
	existingTags := sc.AllTags()
	existingCats := sc.AllCategories()
	sc.mu.RUnlock()

	result := validateFrontmatter(input.Frontmatter, existingTags, existingCats)
	return nil, result, nil
}

func (fs *ForgeServer) handleGetTemplateContext(ctx context.Context, req *mcp.CallToolRequest, input GetTemplateContextInput) (*mcp.CallToolResult, GetTemplateContextOutput, error) {
	sc, err := fs.ctx.Load()
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, GetTemplateContextOutput{}, nil
	}

	sc.mu.RLock()
	pages := sc.pages
	sc.mu.RUnlock()

	// Find the page
	var target *content.Page
	for _, p := range pages {
		if matchPagePath(p.SourcePath, input.PagePath) {
			target = p
			break
		}
	}
	if target == nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("page not found: %s", input.PagePath)}},
		}, GetTemplateContextOutput{}, nil
	}

	// Resolve layout
	layoutInfo := resolveLayout(target, fs.siteDir)

	out := GetTemplateContextOutput{
		ResolvedTemplate: layoutInfo.Resolved,
		BaseTemplate:     layoutInfo.BaseTemplate,
		Partials:         layoutInfo.Blocks,
		Context: map[string]any{
			"Title":       target.Title,
			"Date":        target.Date,
			"Draft":       target.Draft,
			"Tags":        target.Tags,
			"Categories":  target.Categories,
			"Series":      target.Series,
			"ReadingTime": target.ReadingTime,
			"WordCount":   target.WordCount,
			"URL":         target.URL,
			"Section":     target.Section,
		},
		AvailableFunctions: []string{
			"markdownify", "plainify", "truncate", "slugify", "highlight",
			"safeHTML", "where", "sort", "first", "last", "shuffle", "group",
			"dateFormat", "now", "readingTime", "relURL", "absURL", "ref",
		},
	}
	return nil, out, nil
}

func (fs *ForgeServer) handleResolveLayout(ctx context.Context, req *mcp.CallToolRequest, input ResolveLayoutInput) (*mcp.CallToolResult, ResolveLayoutOutput, error) {
	sc, err := fs.ctx.Load()
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, ResolveLayoutOutput{}, nil
	}

	sc.mu.RLock()
	pages := sc.pages
	sc.mu.RUnlock()

	var target *content.Page
	for _, p := range pages {
		if matchPagePath(p.SourcePath, input.PagePath) {
			target = p
			break
		}
	}
	if target == nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("page not found: %s", input.PagePath)}},
		}, ResolveLayoutOutput{}, nil
	}

	return nil, resolveLayout(target, fs.siteDir), nil
}

func (fs *ForgeServer) handleCreateContent(ctx context.Context, req *mcp.CallToolRequest, input CreateContentInput) (*mcp.CallToolResult, CreateContentOutput, error) {
	if input.Title == "" {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "title is required"}}}, CreateContentOutput{}, nil
	}

	sc, err := fs.ctx.Load()
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, CreateContentOutput{}, nil
	}

	sc.mu.RLock()
	existingTags := sc.AllTags()
	existingCats := sc.AllCategories()
	sc.mu.RUnlock()

	slug := input.Slug
	if slug == "" {
		slug = scaffold.Slugify(input.Title)
	}

	// Determine draft status
	isDraft := true
	if input.Draft != nil {
		isDraft = *input.Draft
	}

	// Determine file path
	now := time.Now()
	var relPath, url string
	switch input.Type {
	case "post":
		datePrefix := now.Format("2006-01-02")
		if input.PageBundle {
			relPath = fmt.Sprintf("content/blog/%s-%s/index.md", datePrefix, slug)
		} else {
			relPath = fmt.Sprintf("content/blog/%s-%s.md", datePrefix, slug)
		}
		url = fmt.Sprintf("/blog/%s/", slug)
	case "page":
		relPath = fmt.Sprintf("content/%s.md", slug)
		url = fmt.Sprintf("/%s/", slug)
	case "project":
		if input.PageBundle {
			relPath = fmt.Sprintf("content/projects/%s/index.md", slug)
		} else {
			relPath = fmt.Sprintf("content/projects/%s.md", slug)
		}
		url = fmt.Sprintf("/projects/%s/", slug)
	default:
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("unknown content type %q; must be post, page, or project", input.Type)}},
		}, CreateContentOutput{}, nil
	}

	absPath := filepath.Join(fs.siteDir, relPath)

	// Check if file already exists
	if _, err := os.Stat(absPath); err == nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("file already exists: %s", relPath)}},
		}, CreateContentOutput{}, nil
	}

	// Validate tags and categories
	var warnings []string
	for _, t := range input.Tags {
		similar := findSimilarTerms(t, existingTags, 2)
		for _, s := range similar {
			if s != t {
				warnings = append(warnings, fmt.Sprintf("Tag %q is similar to existing tag %q", t, s))
			}
		}
		if !containsStr(existingTags, t) {
			warnings = append(warnings, fmt.Sprintf("Tag %q is new and will create a new taxonomy term", t))
		}
	}
	for _, c := range input.Categories {
		similar := findSimilarTerms(c, existingCats, 2)
		for _, s := range similar {
			if s != c {
				warnings = append(warnings, fmt.Sprintf("Category %q is similar to existing category %q", c, s))
			}
		}
		if !containsStr(existingCats, c) {
			warnings = append(warnings, fmt.Sprintf("Category %q is new and will create a new taxonomy term", c))
		}
	}

	// Build frontmatter
	fm := buildFrontmatterYAML(input, slug, isDraft, now)

	// Build file content
	body := input.Body
	if body == "" {
		body = "\n"
	}
	fileContent := "---\n" + fm + "---\n\n" + body

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return nil, CreateContentOutput{}, fmt.Errorf("creating directories: %w", err)
	}

	// Write file
	if err := os.WriteFile(absPath, []byte(fileContent), 0644); err != nil {
		return nil, CreateContentOutput{}, fmt.Errorf("writing file: %w", err)
	}

	// Mark context dirty so it reloads
	fs.ctx.MarkDirty()

	return nil, CreateContentOutput{
		Created:     true,
		FilePath:    relPath,
		URL:         url,
		Frontmatter: fm,
		Warnings:    warnings,
	}, nil
}

func (fs *ForgeServer) handleBuildSite(ctx context.Context, req *mcp.CallToolRequest, input BuildSiteInput) (*mcp.CallToolResult, BuildSiteOutput, error) {
	sc, err := fs.ctx.Load()
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, BuildSiteOutput{}, nil
	}

	sc.mu.RLock()
	cfg := sc.cfg
	sc.mu.RUnlock()

	outputDir := input.OutputDir
	if outputDir == "" {
		outputDir = "public"
	}

	opts := build.BuildOptions{
		IncludeDrafts: input.Drafts,
		IncludeFuture: input.Future,
		OutputDir:     outputDir,
		Verbose:       input.Verbose,
		ProjectRoot:   fs.siteDir,
	}
	if input.BaseURL != "" {
		opts.BaseURL = input.BaseURL
	}

	builder := build.NewBuilder(cfg, opts)
	start := time.Now()
	result, buildErr := builder.Build()

	var out BuildSiteOutput
	if buildErr != nil {
		out = BuildSiteOutput{
			Success: false,
			Errors:  []BuildIssue{{Message: buildErr.Error()}},
		}
	} else {
		out = BuildSiteOutput{
			Success:           true,
			DurationMs:        time.Since(start).Milliseconds(),
			PagesRendered:     result.PagesRendered,
			StaticFilesCopied: result.StaticFiles,
			OutputDir:         outputDir + "/",
			OutputSizeBytes:   result.OutputSize,
			Errors:            []BuildIssue{},
			Warnings:          []BuildIssue{},
		}
	}

	// Store last build result
	fs.lastBuild = &BuildResultDetail{
		Timestamp:       time.Now(),
		DurationMs:      out.DurationMs,
		Success:         out.Success,
		PagesRendered:   out.PagesRendered,
		OutputDir:       out.OutputDir,
		OutputSizeBytes: out.OutputSizeBytes,
		Errors:          out.Errors,
		Warnings:        out.Warnings,
	}

	// Notify resource updated
	_ = fs.server.ResourceUpdated(ctx, &mcp.ResourceUpdatedNotificationParams{URI: "forge://build/status"})

	return nil, out, nil
}

func (fs *ForgeServer) handleDeploySite(ctx context.Context, req *mcp.CallToolRequest, input DeploySiteInput) (*mcp.CallToolResult, DeploySiteOutput, error) {
	sc, err := fs.ctx.Load()
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, DeploySiteOutput{}, nil
	}

	sc.mu.RLock()
	cfg := sc.cfg
	sc.mu.RUnlock()

	if cfg.Deploy.S3.Bucket == "" {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "deploy.s3.bucket is not configured in forge.yaml"}},
		}, DeploySiteOutput{}, nil
	}

	// Note: Real S3/CF clients need AWS SDK; for now return a meaningful error
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: "deploy_site requires AWS credentials; use 'forge deploy' CLI command instead"}},
	}, DeploySiteOutput{
		DryRun: input.DryRun,
		Bucket: cfg.Deploy.S3.Bucket,
		Region: cfg.Deploy.S3.Region,
	}, nil
}

// --- Query filter helpers ---

func filterPages(pages []*content.Page, input QueryContentInput) []*content.Page {
	var result []*content.Page
	for _, p := range pages {
		if input.Section != "" && p.Section != input.Section {
			continue
		}
		if input.Draft != nil && p.Draft != *input.Draft {
			continue
		}
		if len(input.Tags) > 0 && !hasAllTags(p, input.Tags) {
			continue
		}
		if len(input.Categories) > 0 && !hasAnyCategory(p, input.Categories) {
			continue
		}
		if input.Series != "" && p.Series != input.Series {
			continue
		}
		if input.DateAfter != "" {
			t, err := time.Parse(time.RFC3339, input.DateAfter)
			if err == nil && !p.Date.After(t) {
				continue
			}
		}
		if input.DateBefore != "" {
			t, err := time.Parse(time.RFC3339, input.DateBefore)
			if err == nil && !p.Date.Before(t) {
				continue
			}
		}
		if input.Search != "" {
			q := strings.ToLower(input.Search)
			if !strings.Contains(strings.ToLower(p.Title), q) &&
				!strings.Contains(strings.ToLower(p.Summary), q) &&
				!strings.Contains(strings.ToLower(p.RawContent), q) {
				continue
			}
		}
		result = append(result, p)
	}
	return result
}

func hasAllTags(p *content.Page, tags []string) bool {
	for _, t := range tags {
		found := false
		for _, pt := range p.Tags {
			if strings.EqualFold(pt, t) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func hasAnyCategory(p *content.Page, cats []string) bool {
	for _, c := range cats {
		for _, pc := range p.Categories {
			if strings.EqualFold(pc, c) {
				return true
			}
		}
	}
	return false
}

func sortPages(pages []*content.Page, by, order string) {
	sort.Slice(pages, func(i, j int) bool {
		var less bool
		switch by {
		case "title":
			less = pages[i].Title < pages[j].Title
		case "weight":
			less = pages[i].Weight < pages[j].Weight
		case "readingTime":
			less = pages[i].ReadingTime < pages[j].ReadingTime
		case "wordCount":
			less = pages[i].WordCount < pages[j].WordCount
		default: // date
			less = pages[i].Date.Before(pages[j].Date)
		}
		if order == "asc" {
			return less
		}
		return !less
	})
}

func paginatePages(pages []*content.Page, offset, limit int) []*content.Page {
	if offset >= len(pages) {
		return []*content.Page{}
	}
	end := offset + limit
	if end > len(pages) {
		end = len(pages)
	}
	return pages[offset:end]
}

func resolveLayout(p *content.Page, siteDir string) ResolveLayoutOutput {
	section := p.Section
	layout := p.Layout
	if layout == "" {
		layout = "single"
	}

	type candidate struct {
		path   string
		source string
	}

	themePath := filepath.Join(siteDir, "embedded", "themes", "default", "layouts")
	userPath := filepath.Join(siteDir, "layouts")

	candidates := []candidate{}
	if section != "" {
		candidates = append(candidates,
			candidate{filepath.Join(userPath, section, layout+".html"), "user"},
			candidate{filepath.Join(themePath, section, layout+".html"), "theme"},
			candidate{filepath.Join(userPath, section, "single.html"), "user"},
			candidate{filepath.Join(themePath, section, "single.html"), "theme"},
		)
	}
	candidates = append(candidates,
		candidate{filepath.Join(userPath, "_default", layout+".html"), "user"},
		candidate{filepath.Join(themePath, "_default", layout+".html"), "theme"},
		candidate{filepath.Join(userPath, "_default", "single.html"), "user"},
		candidate{filepath.Join(themePath, "_default", "single.html"), "theme"},
	)

	var resolved, resolvedSource string
	lookupOrder := make([]LayoutLookup, 0, len(candidates))
	for _, c := range candidates {
		rel, _ := filepath.Rel(siteDir, c.path)
		_, err := os.Stat(c.path)
		exists := err == nil
		if exists && resolved == "" {
			resolved = rel
			resolvedSource = c.source
		}
		lookupOrder = append(lookupOrder, LayoutLookup{Path: rel, Exists: exists, Source: c.source})
	}

	baseof := filepath.Join(themePath, "_default", "baseof.html")
	baseoRel, _ := filepath.Rel(siteDir, baseof)
	if _, err := os.Stat(filepath.Join(userPath, "_default", "baseof.html")); err == nil {
		baseoRel, _ = filepath.Rel(siteDir, filepath.Join(userPath, "_default", "baseof.html"))
	}

	return ResolveLayoutOutput{
		Resolved:     resolved,
		Source:       resolvedSource,
		LookupOrder:  lookupOrder,
		BaseTemplate: baseoRel,
		Blocks:       []string{"head", "main", "scripts"},
	}
}

func buildFrontmatterYAML(input CreateContentInput, slug string, isDraft bool, now time.Time) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("title: %q\n", input.Title))
	sb.WriteString(fmt.Sprintf("date: %s\n", now.Format(time.RFC3339)))
	if isDraft {
		sb.WriteString("draft: true\n")
	}
	if len(input.Tags) > 0 {
		sb.WriteString("tags:\n")
		for _, t := range input.Tags {
			sb.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}
	if len(input.Categories) > 0 {
		sb.WriteString("categories:\n")
		for _, c := range input.Categories {
			sb.WriteString(fmt.Sprintf("  - %s\n", c))
		}
	}
	if input.Series != "" {
		sb.WriteString(fmt.Sprintf("series: %q\n", input.Series))
	}
	if input.Description != "" {
		sb.WriteString(fmt.Sprintf("description: %q\n", input.Description))
	}
	if input.Slug != "" && input.Slug != slug {
		sb.WriteString(fmt.Sprintf("slug: %q\n", input.Slug))
	}
	return sb.String()
}

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
