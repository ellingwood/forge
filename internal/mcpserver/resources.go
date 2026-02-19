package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aellingwood/forge/internal/config"
	"github.com/aellingwood/forge/internal/content"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (fs *ForgeServer) registerResources() {
	// Static resources
	fs.server.AddResource(&mcp.Resource{
		URI:         "forge://config",
		Name:        "Site Configuration",
		Description: "Resolved site configuration from forge.yaml",
		MIMEType:    "application/json",
	}, fs.handleConfigResource)

	fs.server.AddResource(&mcp.Resource{
		URI:         "forge://content/pages",
		Name:        "Content Inventory",
		Description: "All content pages with metadata (no body)",
		MIMEType:    "application/json",
	}, fs.handlePagesResource)

	fs.server.AddResource(&mcp.Resource{
		URI:         "forge://content/sections",
		Name:        "Sections",
		Description: "All content sections with page counts",
		MIMEType:    "application/json",
	}, fs.handleSectionsResource)

	fs.server.AddResource(&mcp.Resource{
		URI:         "forge://taxonomies",
		Name:        "Taxonomies Overview",
		Description: "All taxonomies with their terms and counts",
		MIMEType:    "application/json",
	}, fs.handleTaxonomiesResource)

	fs.server.AddResource(&mcp.Resource{
		URI:         "forge://templates",
		Name:        "Template Inventory",
		Description: "All available layouts and partials with file paths",
		MIMEType:    "application/json",
	}, fs.handleTemplatesResource)

	fs.server.AddResource(&mcp.Resource{
		URI:         "forge://build/status",
		Name:        "Build Status",
		Description: "Last build result — timestamp, duration, errors, warnings",
		MIMEType:    "application/json",
	}, fs.handleBuildStatusResource)

	fs.server.AddResource(&mcp.Resource{
		URI:         "forge://schema/frontmatter",
		Name:        "Frontmatter Schema",
		Description: "Valid frontmatter fields, types, defaults, and constraints",
		MIMEType:    "application/json",
	}, fs.handleFrontmatterSchemaResource)

	// Resource templates (parameterized URIs)
	fs.server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "forge://content/page/{path}",
		Name:        "Page Detail",
		Description: "Full detail for a single content page",
		MIMEType:    "application/json",
	}, fs.handlePageDetailResource)

	fs.server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "forge://taxonomies/{name}",
		Name:        "Taxonomy Detail",
		Description: "Terms and pages for a specific taxonomy",
		MIMEType:    "application/json",
	}, fs.handleTaxonomyDetailResource)
}

func jsonResource(uri, data string) *mcp.ReadResourceResult {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: uri, MIMEType: "application/json", Text: data},
		},
	}
}

func marshalResource(uri string, v any) (*mcp.ReadResourceResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return jsonResource(uri, string(b)), nil
}

func (fs *ForgeServer) handleConfigResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	sc, err := fs.ctx.Load()
	if err != nil {
		return nil, err
	}
	sc.mu.RLock()
	cfg := sc.cfg
	sc.mu.RUnlock()
	return marshalResource(req.Params.URI, cfg)
}

func (fs *ForgeServer) handlePagesResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	sc, err := fs.ctx.Load()
	if err != nil {
		return nil, err
	}
	sc.mu.RLock()
	pages := sc.pages
	sc.mu.RUnlock()

	briefs := make([]PageBrief, len(pages))
	for i, p := range pages {
		briefs[i] = toPageBrief(p)
	}
	result := map[string]any{
		"totalPages": len(briefs),
		"pages":      briefs,
	}
	return marshalResource(req.Params.URI, result)
}

func (fs *ForgeServer) handleSectionsResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	sc, err := fs.ctx.Load()
	if err != nil {
		return nil, err
	}
	sc.mu.RLock()
	pages := sc.pages
	sc.mu.RUnlock()

	sections := buildSections(pages, fs.siteDir)
	result := map[string]any{"sections": sections}
	return marshalResource(req.Params.URI, result)
}

func (fs *ForgeServer) handleTaxonomiesResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	sc, err := fs.ctx.Load()
	if err != nil {
		return nil, err
	}
	sc.mu.RLock()
	pages := sc.pages
	cfg := sc.cfg
	sc.mu.RUnlock()

	overview := buildTaxonomyOverview(pages, cfg)
	return marshalResource(req.Params.URI, overview)
}

func (fs *ForgeServer) handleTemplatesResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	inv := buildTemplateInventory(fs.siteDir)
	return marshalResource(req.Params.URI, inv)
}

func (fs *ForgeServer) handleBuildStatusResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	status := BuildStatus{LastBuild: fs.lastBuild}
	return marshalResource(req.Params.URI, status)
}

func (fs *ForgeServer) handleFrontmatterSchemaResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	sc, err := fs.ctx.Load()
	if err != nil {
		return nil, err
	}
	sc.mu.RLock()
	tags := sc.AllTags()
	cats := sc.AllCategories()
	series := sc.AllSeries()
	projects := sc.AllProjectSlugs()
	sc.mu.RUnlock()

	schema := buildFrontmatterSchema(tags, cats, series, projects)
	return marshalResource(req.Params.URI, schema)
}

func (fs *ForgeServer) handlePageDetailResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// Extract {path} from URI: "forge://content/page/{path}"
	uri := req.Params.URI
	prefix := "forge://content/page/"
	if !strings.HasPrefix(uri, prefix) {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	path := strings.TrimPrefix(uri, prefix)

	sc, err := fs.ctx.Load()
	if err != nil {
		return nil, err
	}
	sc.mu.RLock()
	pages := sc.pages
	sc.mu.RUnlock()

	for _, p := range pages {
		if matchPagePath(p.SourcePath, path) {
			detail := toPageDetail(p)
			return marshalResource(uri, detail)
		}
	}
	return nil, mcp.ResourceNotFoundError(uri)
}

func (fs *ForgeServer) handleTaxonomyDetailResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// Extract {name} from URI: "forge://taxonomies/{name}"
	uri := req.Params.URI
	prefix := "forge://taxonomies/"
	if !strings.HasPrefix(uri, prefix) {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	name := strings.TrimPrefix(uri, prefix)

	sc, err := fs.ctx.Load()
	if err != nil {
		return nil, err
	}
	sc.mu.RLock()
	pages := sc.pages
	sc.mu.RUnlock()

	detail, ok := buildTaxonomyDetail(name, pages)
	if !ok {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	return marshalResource(uri, detail)
}

// --- Helper functions ---

// matchPagePath checks if sourcePath (content-dir-relative, e.g. "blog/my-post.md")
// matches an API path (site-root-relative, e.g. "content/blog/my-post.md", or
// content-dir-relative "blog/my-post.md").
func matchPagePath(sourcePath, apiPath string) bool {
	// Direct match (content-relative)
	if sourcePath == apiPath {
		return true
	}
	// Strip "content/" prefix from API path (site-root → content-relative)
	contentRelative := strings.TrimPrefix(apiPath, "content/")
	if sourcePath == contentRelative {
		return true
	}
	// Suffix match for any remaining cases
	return strings.HasSuffix(sourcePath, "/"+apiPath) || strings.HasSuffix(sourcePath, "/"+contentRelative)
}

func toPageBrief(p *content.Page) PageBrief {
	b := PageBrief{
		Path:        p.SourcePath,
		URL:         p.URL,
		Title:       p.Title,
		Date:        p.Date,
		Lastmod:     p.Lastmod,
		Draft:       p.Draft,
		Section:     p.Section,
		Tags:        p.Tags,
		Categories:  p.Categories,
		Series:      p.Series,
		Project:     p.Project,
		Summary:     p.Summary,
		Description: p.Description,
		ReadingTime: p.ReadingTime,
		WordCount:   p.WordCount,
		HasCover:    p.Cover != nil,
		IsBundle:    p.IsBundle,
	}
	if b.Tags == nil {
		b.Tags = []string{}
	}
	if b.Categories == nil {
		b.Categories = []string{}
	}
	return b
}

func toPageDetail(p *content.Page) PageDetail {
	d := PageDetail{
		PageBrief:       toPageBrief(p),
		Slug:            p.Slug,
		Permalink:       p.Permalink,
		Weight:          p.Weight,
		Layout:          p.Layout,
		Aliases:         p.Aliases,
		Params:          p.Params,
		RawMarkdown:     p.RawContent,
		RenderedHTML:    p.Content,
		TableOfContents: p.TableOfContents,
		BundleAssets:    p.BundleFiles,
	}
	if p.Cover != nil {
		d.Cover = &CoverImageDetail{
			Image:   p.Cover.Image,
			Alt:     p.Cover.Alt,
			Caption: p.Cover.Caption,
		}
	}
	if p.PrevPage != nil {
		d.PrevPage = &PageRef{Title: p.PrevPage.Title, URL: p.PrevPage.URL}
	}
	if p.NextPage != nil {
		d.NextPage = &PageRef{Title: p.NextPage.Title, URL: p.NextPage.URL}
	}
	return d
}

func buildSections(pages []*content.Page, siteDir string) []SectionInfo {
	type sectionData struct {
		count      int
		draftCount int
		latest     time.Time
		oldest     time.Time
		hasIndex   bool
	}
	data := make(map[string]*sectionData)

	for _, p := range pages {
		if p.Section == "" {
			continue
		}
		d, ok := data[p.Section]
		if !ok {
			d = &sectionData{}
			data[p.Section] = d
		}
		d.count++
		if p.Draft {
			d.draftCount++
		}
		if !p.Date.IsZero() {
			if d.latest.IsZero() || p.Date.After(d.latest) {
				d.latest = p.Date
			}
			if d.oldest.IsZero() || p.Date.Before(d.oldest) {
				d.oldest = p.Date
			}
		}
		// Check if this page is an index page
		if strings.HasSuffix(p.SourcePath, "_index.md") {
			d.hasIndex = true
		}
	}

	sections := make([]SectionInfo, 0, len(data))
	for name, d := range data {
		sections = append(sections, SectionInfo{
			Name:       name,
			Path:       fmt.Sprintf("content/%s/", name),
			PageCount:  d.count,
			DraftCount: d.draftCount,
			HasIndex:   d.hasIndex,
			LatestDate: d.latest,
			OldestDate: d.oldest,
		})
	}
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Name < sections[j].Name
	})
	return sections
}

func buildTaxonomyOverview(pages []*content.Page, cfg *config.SiteConfig) TaxonomyOverview {
	// Build term counts per taxonomy
	tagCounts := make(map[string]int)
	catCounts := make(map[string]int)

	for _, p := range pages {
		for _, t := range p.Tags {
			tagCounts[t]++
		}
		for _, c := range p.Categories {
			catCounts[c]++
		}
	}

	var taxos []TaxonomySummary

	// Tags taxonomy
	if len(tagCounts) > 0 {
		terms := make([]TermBrief, 0, len(tagCounts))
		total := 0
		for name, count := range tagCounts {
			terms = append(terms, TermBrief{Name: name, Slug: normalizeTaxonomyName(name), Count: count})
			total += count
		}
		sort.Slice(terms, func(i, j int) bool { return terms[i].Count > terms[j].Count })
		taxos = append(taxos, TaxonomySummary{
			Name:             "tags",
			Singular:         "tag",
			URLBase:          "/tags/",
			TermCount:        len(terms),
			TotalAssignments: total,
			Terms:            terms,
		})
	}

	// Categories taxonomy
	if len(catCounts) > 0 {
		terms := make([]TermBrief, 0, len(catCounts))
		total := 0
		for name, count := range catCounts {
			terms = append(terms, TermBrief{Name: name, Slug: normalizeTaxonomyName(name), Count: count})
			total += count
		}
		sort.Slice(terms, func(i, j int) bool { return terms[i].Count > terms[j].Count })
		taxos = append(taxos, TaxonomySummary{
			Name:             "categories",
			Singular:         "category",
			URLBase:          "/categories/",
			TermCount:        len(terms),
			TotalAssignments: total,
			Terms:            terms,
		})
	}

	return TaxonomyOverview{Taxonomies: taxos}
}

func buildTaxonomyDetail(name string, pages []*content.Page) (TaxonomyDetail, bool) {
	var getTerms func(*content.Page) []string
	var singular, urlBase string

	switch strings.ToLower(name) {
	case "tags":
		getTerms = func(p *content.Page) []string { return p.Tags }
		singular = "tag"
		urlBase = "/tags/"
	case "categories":
		getTerms = func(p *content.Page) []string { return p.Categories }
		singular = "category"
		urlBase = "/categories/"
	default:
		return TaxonomyDetail{}, false
	}

	termPages := make(map[string][]PageRef)
	for _, p := range pages {
		for _, t := range getTerms(p) {
			termPages[t] = append(termPages[t], PageRef{Title: p.Title, URL: p.URL})
		}
	}

	terms := make([]TermDetail, 0, len(termPages))
	for termName, refs := range termPages {
		slug := normalizeTaxonomyName(termName)
		terms = append(terms, TermDetail{
			Name:  termName,
			Slug:  slug,
			URL:   fmt.Sprintf("%s%s/", urlBase, slug),
			Count: len(refs),
			Pages: refs,
		})
	}
	sort.Slice(terms, func(i, j int) bool { return terms[i].Count > terms[j].Count })

	return TaxonomyDetail{
		Name:     name,
		Singular: singular,
		URLBase:  urlBase,
		Terms:    terms,
	}, true
}

func buildTemplateInventory(siteDir string) TemplateInventory {
	inv := TemplateInventory{}

	themePath := filepath.Join(siteDir, "embedded", "themes", "default", "layouts")
	userPath := filepath.Join(siteDir, "layouts")

	walkTemplates := func(root, source string) {
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".html") {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			entry := TemplateEntry{
				Path:   rel,
				Source: source,
			}
			if strings.Contains(rel, "partials") {
				inv.Partials = append(inv.Partials, entry)
			} else {
				if strings.HasSuffix(rel, "single.html") {
					entry.Type = "single"
				} else if strings.HasSuffix(rel, "list.html") {
					entry.Type = "list"
				} else if strings.HasSuffix(rel, "baseof.html") {
					entry.Type = "base"
				}
				parts := strings.SplitN(rel, string(filepath.Separator), 2)
				if len(parts) == 2 && parts[0] != "_default" {
					entry.Section = parts[0]
				}
				inv.Layouts = append(inv.Layouts, entry)
			}
			return nil
		})
	}

	walkTemplates(themePath, "theme")
	walkTemplates(userPath, "user")

	return inv
}

func buildFrontmatterSchema(tags, cats, series, projects []string) FrontmatterSchema {
	return FrontmatterSchema{
		Required: []string{"title"},
		Fields: map[string]FieldSchema{
			"title": {
				Type:        "string",
				Description: "Page title (required)",
				Default:     nil,
			},
			"date": {
				Type:        "datetime",
				Description: "Publish date (ISO 8601)",
				Default:     "now",
			},
			"draft": {
				Type:        "boolean",
				Description: "Exclude from production builds",
				Default:     true,
			},
			"tags": {
				Type:           "[]string",
				Description:    "Tag taxonomy terms",
				Default:        []string{},
				ExistingValues: tags,
			},
			"categories": {
				Type:           "[]string",
				Description:    "Category taxonomy terms",
				Default:        []string{},
				ExistingValues: cats,
			},
			"series": {
				Type:           "string",
				Description:    "Group related posts into a named series",
				Default:        nil,
				ExistingValues: series,
			},
			"project": {
				Type:           "string",
				Description:    "Associate this post with a project page by slug",
				Default:        nil,
				ExistingValues: projects,
			},
			"cover": {
				Type:        "object",
				Description: "Cover image configuration",
				Fields: map[string]any{
					"image":   map[string]string{"type": "string"},
					"alt":     map[string]string{"type": "string"},
					"caption": map[string]string{"type": "string"},
				},
			},
			"slug": {
				Type:        "string",
				Description: "URL slug override (default: derived from filename)",
			},
			"description": {
				Type:        "string",
				Description: "Meta description / OpenGraph description",
			},
			"summary": {
				Type:        "string",
				Description: "Explicit summary for listing pages",
			},
			"weight": {
				Type:        "integer",
				Description: "Sort order for non-date ordering",
				Default:     0,
			},
			"layout": {
				Type:        "string",
				Description: "Explicit layout override",
				ValidValues: []string{"post", "project", "page"},
			},
			"aliases": {
				Type:        "[]string",
				Description: "Redirect old URLs to this page",
			},
			"params": {
				Type:        "map[string]any",
				Description: "Arbitrary key-value pairs accessible in templates",
				KnownKeys: map[string]any{
					"toc":  map[string]any{"type": "boolean", "default": false},
					"math": map[string]any{"type": "boolean", "default": false},
				},
			},
		},
	}
}
