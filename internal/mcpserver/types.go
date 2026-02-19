// Package mcpserver implements an MCP (Model Context Protocol) server for Forge,
// exposing the site's content graph, build system, and configuration as
// structured, queryable data to MCP clients.
package mcpserver

import "time"

// PageBrief is a lightweight page summary without body content.
type PageBrief struct {
	Path        string    `json:"path"`
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Date        time.Time `json:"date"`
	Lastmod     time.Time `json:"lastmod,omitempty"`
	Draft       bool      `json:"draft"`
	Section     string    `json:"section"`
	Tags        []string  `json:"tags,omitempty"`
	Categories  []string  `json:"categories,omitempty"`
	Series      string    `json:"series,omitempty"`
	Project     string    `json:"project,omitempty"`
	Summary     string    `json:"summary,omitempty"`
	Description string    `json:"description,omitempty"`
	ReadingTime int       `json:"readingTime"`
	WordCount   int       `json:"wordCount"`
	HasCover    bool      `json:"hasCover"`
	IsBundle    bool      `json:"isPageBundle"`
}

// PageDetail is a full page representation including rendered content.
type PageDetail struct {
	PageBrief
	Slug            string            `json:"slug,omitempty"`
	Permalink       string            `json:"permalink,omitempty"`
	Weight          int               `json:"weight"`
	Layout          string            `json:"layout,omitempty"`
	Aliases         []string          `json:"aliases,omitempty"`
	Params          map[string]any    `json:"params,omitempty"`
	Cover           *CoverImageDetail `json:"cover,omitempty"`
	RawMarkdown     string            `json:"rawMarkdown"`
	RenderedHTML    string            `json:"renderedHTML"`
	TableOfContents string            `json:"tableOfContents,omitempty"`
	BundleAssets    []string          `json:"bundleAssets,omitempty"`
	PrevPage        *PageRef          `json:"prevPage,omitempty"`
	NextPage        *PageRef          `json:"nextPage,omitempty"`
}

// PageRef is a minimal reference to a related page.
type PageRef struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// CoverImageDetail holds cover image metadata.
type CoverImageDetail struct {
	Image   string `json:"image"`
	Alt     string `json:"alt,omitempty"`
	Caption string `json:"caption,omitempty"`
}

// SectionInfo describes a content section.
type SectionInfo struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	PageCount  int       `json:"pageCount"`
	DraftCount int       `json:"draftCount"`
	HasIndex   bool      `json:"hasIndex"`
	LatestDate time.Time `json:"latestDate,omitempty"`
	OldestDate time.Time `json:"oldestDate,omitempty"`
}

// TaxonomyOverview holds all taxonomies with term counts.
type TaxonomyOverview struct {
	Taxonomies []TaxonomySummary `json:"taxonomies"`
}

// TaxonomySummary is a lightweight taxonomy descriptor.
type TaxonomySummary struct {
	Name             string      `json:"name"`
	Singular         string      `json:"singular"`
	URLBase          string      `json:"urlBase"`
	TermCount        int         `json:"termCount"`
	TotalAssignments int         `json:"totalAssignments"`
	Terms            []TermBrief `json:"terms"`
}

// TermBrief is a lightweight taxonomy term.
type TermBrief struct {
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Count int    `json:"count"`
}

// TaxonomyDetail is the full detail for one taxonomy.
type TaxonomyDetail struct {
	Name     string       `json:"name"`
	Singular string       `json:"singular"`
	URLBase  string       `json:"urlBase"`
	Terms    []TermDetail `json:"terms"`
}

// TermDetail is a taxonomy term with its associated pages.
type TermDetail struct {
	Name  string    `json:"name"`
	Slug  string    `json:"slug"`
	URL   string    `json:"url"`
	Count int       `json:"count"`
	Pages []PageRef `json:"pages"`
}

// TemplateInventory lists all available templates.
type TemplateInventory struct {
	Layouts  []TemplateEntry `json:"layouts"`
	Partials []TemplateEntry `json:"partials"`
}

// TemplateEntry describes a single template file.
type TemplateEntry struct {
	Path         string `json:"path"`
	Source       string `json:"source"` // "theme" or "user"
	Type         string `json:"type,omitempty"`
	Section      string `json:"section,omitempty"`
	OverriddenBy string `json:"overriddenBy,omitempty"`
	Overrides    string `json:"overrides,omitempty"`
}

// BuildStatus holds the last build result.
type BuildStatus struct {
	LastBuild *BuildResultDetail `json:"lastBuild"`
}

// BuildResultDetail describes a completed build.
type BuildResultDetail struct {
	Timestamp       time.Time    `json:"timestamp"`
	DurationMs      int64        `json:"durationMs"`
	Success         bool         `json:"success"`
	PagesRendered   int          `json:"pagesRendered"`
	OutputDir       string       `json:"outputDir"`
	OutputSizeBytes int64        `json:"outputSizeBytes"`
	Errors          []BuildIssue `json:"errors"`
	Warnings        []BuildIssue `json:"warnings"`
}

// BuildIssue describes a build error or warning.
type BuildIssue struct {
	File    string `json:"file,omitempty"`
	Message string `json:"message"`
	Level   string `json:"level,omitempty"`
}

// FrontmatterSchema describes valid frontmatter fields.
type FrontmatterSchema struct {
	Required []string               `json:"required"`
	Fields   map[string]FieldSchema `json:"fields"`
}

// FieldSchema describes a single frontmatter field.
type FieldSchema struct {
	Type           string         `json:"type"`
	Description    string         `json:"description"`
	Default        any            `json:"default"`
	ExistingValues []string       `json:"existingValues,omitempty"`
	ValidValues    []string       `json:"validValues,omitempty"`
	Fields         map[string]any `json:"fields,omitempty"`
	KnownKeys      map[string]any `json:"knownKeys,omitempty"`
}

// QueryContentInput is the input for the query_content tool.
type QueryContentInput struct {
	Section    string   `json:"section,omitempty"    jsonschema:"Filter by content section (e.g. blog, projects)"`
	Tags       []string `json:"tags,omitempty"       jsonschema:"Filter pages that have ALL of these tags"`
	Categories []string `json:"categories,omitempty" jsonschema:"Filter pages that have ANY of these categories"`
	Draft      *bool    `json:"draft,omitempty"      jsonschema:"Filter by draft status; omit to include all"`
	DateAfter  string   `json:"dateAfter,omitempty"  jsonschema:"Only pages published after this date (ISO 8601)"`
	DateBefore string   `json:"dateBefore,omitempty" jsonschema:"Only pages published before this date (ISO 8601)"`
	Series     string   `json:"series,omitempty"     jsonschema:"Filter by series name"`
	Project    string   `json:"project,omitempty"    jsonschema:"Filter by project slug"`
	Search     string   `json:"search,omitempty"     jsonschema:"Full-text search across title, summary, and content"`
	SortBy     string   `json:"sortBy,omitempty"     jsonschema:"Sort field: date, title, weight, readingTime, wordCount (default: date)"`
	SortOrder  string   `json:"sortOrder,omitempty"  jsonschema:"Sort order: asc or desc (default: desc)"`
	Limit      int      `json:"limit,omitempty"      jsonschema:"Max results to return, 1-100 (default: 20)"`
	Offset     int      `json:"offset,omitempty"     jsonschema:"Pagination offset (default: 0)"`
}

// QueryContentOutput is the output from the query_content tool.
type QueryContentOutput struct {
	TotalMatches int         `json:"totalMatches"`
	Offset       int         `json:"offset"`
	Limit        int         `json:"limit"`
	Pages        []PageBrief `json:"pages"`
}

// GetPageInput is the input for the get_page tool.
type GetPageInput struct {
	Path string `json:"path,omitempty" jsonschema:"Content file path relative to site root (e.g. content/blog/my-post.md)"`
	URL  string `json:"url,omitempty"  jsonschema:"Page URL, e.g. /blog/my-post/ (alternative to path)"`
}

// ListDraftsInput is the input for the list_drafts tool.
type ListDraftsInput struct {
	Section string `json:"section,omitempty" jsonschema:"Optionally filter drafts by section"`
}

// ListDraftsOutput is the output from the list_drafts tool.
type ListDraftsOutput struct {
	TotalDrafts int         `json:"totalDrafts"`
	Drafts      []PageBrief `json:"drafts"`
}

// ValidateFrontmatterInput is the input for the validate_frontmatter tool.
type ValidateFrontmatterInput struct {
	Frontmatter string `json:"frontmatter"           jsonschema:"Raw YAML frontmatter without --- delimiters"`
	Section     string `json:"section,omitempty"     jsonschema:"Target section (affects layout validation)"`
}

// ValidateFrontmatterOutput is the output from the validate_frontmatter tool.
type ValidateFrontmatterOutput struct {
	Valid                 bool                `json:"valid"`
	Errors                []ValidationError   `json:"errors"`
	Warnings              []ValidationWarning `json:"warnings"`
	NormalizedFrontmatter string              `json:"normalizedFrontmatter,omitempty"`
}

// ValidationError describes a frontmatter validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
}

// ValidationWarning describes a frontmatter validation warning.
type ValidationWarning struct {
	Field      string `json:"field"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// GetTemplateContextInput is the input for the get_template_context tool.
type GetTemplateContextInput struct {
	PagePath     string `json:"pagePath"               jsonschema:"Content file path to simulate rendering for"`
	TemplatePath string `json:"templatePath,omitempty" jsonschema:"Specific template to inspect; auto-resolved if omitted"`
}

// GetTemplateContextOutput is the output from the get_template_context tool.
type GetTemplateContextOutput struct {
	ResolvedTemplate   string   `json:"resolvedTemplate"`
	BaseTemplate       string   `json:"baseTemplate"`
	Partials           []string `json:"partials"`
	Context            any      `json:"context"`
	AvailableFunctions []string `json:"availableFunctions"`
}

// ResolveLayoutInput is the input for the resolve_layout tool.
type ResolveLayoutInput struct {
	PagePath string `json:"pagePath" jsonschema:"Content file path, e.g. content/blog/my-post.md"`
}

// LayoutLookup describes one entry in the layout resolution chain.
type LayoutLookup struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Source string `json:"source"`
}

// ResolveLayoutOutput is the output from the resolve_layout tool.
type ResolveLayoutOutput struct {
	Resolved     string         `json:"resolved"`
	Source       string         `json:"source"`
	LookupOrder  []LayoutLookup `json:"lookupOrder"`
	BaseTemplate string         `json:"baseTemplate"`
	Blocks       []string       `json:"blocks"`
}

// CreateContentInput is the input for the create_content tool.
type CreateContentInput struct {
	Type        string         `json:"type"                  jsonschema:"Content type: post, page, or project"`
	Title       string         `json:"title"                 jsonschema:"Page title (required)"`
	Slug        string         `json:"slug,omitempty"        jsonschema:"URL slug override (default: generated from title)"`
	Tags        []string       `json:"tags,omitempty"        jsonschema:"Tags to assign"`
	Categories  []string       `json:"categories,omitempty"  jsonschema:"Categories to assign"`
	Series      string         `json:"series,omitempty"      jsonschema:"Series name"`
	Project     string         `json:"project,omitempty"     jsonschema:"Project slug to associate this content with"`
	Draft       *bool          `json:"draft,omitempty"       jsonschema:"Mark as draft (default: true)"`
	Description string         `json:"description,omitempty" jsonschema:"Meta description"`
	Body        string         `json:"body,omitempty"        jsonschema:"Initial Markdown body content"`
	PageBundle  bool           `json:"pageBundle,omitempty"  jsonschema:"Create as a page bundle directory"`
	Params      map[string]any `json:"params,omitempty"      jsonschema:"Additional frontmatter params"`
}

// CreateContentOutput is the output from the create_content tool.
type CreateContentOutput struct {
	Created     bool     `json:"created"`
	FilePath    string   `json:"filePath"`
	URL         string   `json:"url"`
	Frontmatter string   `json:"frontmatter"`
	Warnings    []string `json:"warnings,omitempty"`
}

// BuildSiteInput is the input for the build_site tool.
type BuildSiteInput struct {
	Drafts    bool   `json:"drafts,omitempty"    jsonschema:"Include draft content (default: false)"`
	Future    bool   `json:"future,omitempty"    jsonschema:"Include future-dated content (default: false)"`
	BaseURL   string `json:"baseURL,omitempty"   jsonschema:"Override base URL"`
	OutputDir string `json:"outputDir,omitempty" jsonschema:"Override output directory (default: public/)"`
	Verbose   bool   `json:"verbose,omitempty"   jsonschema:"Include per-page timing in output (default: false)"`
}

// BuildSiteOutput is the output from the build_site tool.
type BuildSiteOutput struct {
	Success           bool         `json:"success"`
	DurationMs        int64        `json:"durationMs"`
	PagesRendered     int          `json:"pagesRendered"`
	StaticFilesCopied int          `json:"staticFilesCopied"`
	OutputDir         string       `json:"outputDir"`
	OutputSizeBytes   int64        `json:"outputSizeBytes"`
	Errors            []BuildIssue `json:"errors"`
	Warnings          []BuildIssue `json:"warnings"`
}

// DeploySiteInput is the input for the deploy_site tool.
type DeploySiteInput struct {
	DryRun           bool `json:"dryRun,omitempty"           jsonschema:"Show what would change without deploying (default: false)"`
	SkipInvalidation bool `json:"skipInvalidation,omitempty" jsonschema:"Skip CloudFront cache invalidation (default: false)"`
}

// DeploySiteOutput is the output from the deploy_site tool.
type DeploySiteOutput struct {
	Success          bool                `json:"success"`
	DryRun           bool                `json:"dryRun"`
	Bucket           string              `json:"bucket"`
	Region           string              `json:"region"`
	FilesUploaded    int                 `json:"filesUploaded"`
	FilesDeleted     int                 `json:"filesDeleted"`
	FilesUnchanged   int                 `json:"filesUnchanged"`
	BytesTransferred int64               `json:"bytesTransferred"`
	Invalidation     *InvalidationResult `json:"invalidation,omitempty"`
	ErrorMessage     string              `json:"errorMessage,omitempty"`
}

// InvalidationResult describes a CloudFront invalidation.
type InvalidationResult struct {
	DistributionID string   `json:"distributionId"`
	InvalidationID string   `json:"invalidationId"`
	Status         string   `json:"status"`
	Paths          []string `json:"paths"`
}
