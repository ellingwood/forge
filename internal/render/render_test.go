package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aellingwood/forge/internal/config"
	"github.com/aellingwood/forge/internal/content"
	tmpl "github.com/aellingwood/forge/internal/template"
)

// testConfig returns a minimal SiteConfig for testing.
func testConfig() *config.SiteConfig {
	return &config.SiteConfig{
		Title:       "Test Site",
		BaseURL:     "https://example.com",
		Description: "A test site",
		Language:    "en",
		Author: config.AuthorConfig{
			Name:  "Test Author",
			Email: "test@example.com",
			Bio:   "A test bio",
			Social: config.SocialConfig{
				GitHub:  "testuser",
				Twitter: "@testuser",
			},
		},
		Menu: config.MenuConfig{
			Main: []config.MenuItem{
				{Name: "Home", URL: "/", Weight: 1},
				{Name: "Blog", URL: "/blog/", Weight: 2},
			},
		},
		Params: map[string]any{
			"color": "blue",
		},
	}
}

// testPage returns a sample content.Page for testing.
func testPage() *content.Page {
	return &content.Page{
		Title:       "Test Post",
		Slug:        "test-post",
		URL:         "/blog/test-post/",
		Permalink:   "https://example.com/blog/test-post/",
		Description: "A test post description",
		Summary:     "<p>Test summary</p>",
		Date:        time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		Lastmod:     time.Date(2025, 6, 20, 12, 0, 0, 0, time.UTC),
		RawContent:  "# Hello\n\nThis is a **test** post.",
		Content:     "<h1>Hello</h1>\n<p>This is a <strong>test</strong> post.</p>",
		WordCount:   7,
		ReadingTime: 1,
		Draft:       false,
		Type:        content.PageTypeSingle,
		Section:     "blog",
		Tags:        []string{"go", "testing"},
		Categories:  []string{"tutorial"},
		Series:      "learn-go",
		Cover: &content.CoverImage{
			Image:   "/images/cover.jpg",
			Alt:     "Cover image",
			Caption: "A nice cover",
		},
		Author: "Test Author",
		Params: map[string]any{
			"custom": "value",
		},
		Aliases: []string{"/old-url/"},
	}
}

func TestNewRenderer(t *testing.T) {
	cfg := testConfig()
	md := content.NewMarkdownRenderer()

	// Create a minimal theme for the engine.
	themeDir := t.TempDir()
	layoutDir := filepath.Join(themeDir, "layouts", "_default")
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layoutDir, "single.html"), []byte("<h1>{{ .Title }}</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, err := tmpl.NewEngine(themeDir, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	r := NewRenderer(engine, md, cfg)
	if r == nil {
		t.Fatal("NewRenderer returned nil")
	}
	if r.engine != engine {
		t.Error("engine not set correctly")
	}
	if r.markdown != md {
		t.Error("markdown not set correctly")
	}
	if r.config != cfg {
		t.Error("config not set correctly")
	}
}

func TestBuildPageContext(t *testing.T) {
	cfg := testConfig()
	md := content.NewMarkdownRenderer()

	// Create a minimal engine (not used for BuildPageContext but required by NewRenderer).
	themeDir := t.TempDir()
	layoutDir := filepath.Join(themeDir, "layouts", "_default")
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layoutDir, "single.html"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	engine, err := tmpl.NewEngine(themeDir, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	r := NewRenderer(engine, md, cfg)
	page := testPage()

	// Set up PrevPage and NextPage.
	prevPage := &content.Page{
		Title:   "Previous Post",
		URL:     "/blog/prev-post/",
		Section: "blog",
		Type:    content.PageTypeSingle,
	}
	nextPage := &content.Page{
		Title:   "Next Post",
		URL:     "/blog/next-post/",
		Section: "blog",
		Type:    content.PageTypeSingle,
	}
	page.PrevPage = prevPage
	page.NextPage = nextPage

	ctx := r.BuildPageContext(page, nil)

	// Verify all fields are mapped correctly.
	t.Run("basic fields", func(t *testing.T) {
		if ctx.Title != "Test Post" {
			t.Errorf("Title = %q, want %q", ctx.Title, "Test Post")
		}
		if ctx.Slug != "test-post" {
			t.Errorf("Slug = %q, want %q", ctx.Slug, "test-post")
		}
		if ctx.URL != "/blog/test-post/" {
			t.Errorf("URL = %q, want %q", ctx.URL, "/blog/test-post/")
		}
		if ctx.Permalink != "https://example.com/blog/test-post/" {
			t.Errorf("Permalink = %q, want %q", ctx.Permalink, "https://example.com/blog/test-post/")
		}
		if ctx.Description != "A test post description" {
			t.Errorf("Description = %q, want %q", ctx.Description, "A test post description")
		}
		if string(ctx.Summary) != "<p>Test summary</p>" {
			t.Errorf("Summary = %q, want %q", ctx.Summary, "<p>Test summary</p>")
		}
	})

	t.Run("dates", func(t *testing.T) {
		if !ctx.Date.Equal(time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)) {
			t.Errorf("Date = %v, want 2025-06-15", ctx.Date)
		}
		if !ctx.Lastmod.Equal(time.Date(2025, 6, 20, 12, 0, 0, 0, time.UTC)) {
			t.Errorf("Lastmod = %v, want 2025-06-20", ctx.Lastmod)
		}
	})

	t.Run("content fields", func(t *testing.T) {
		if string(ctx.Content) != "<h1>Hello</h1>\n<p>This is a <strong>test</strong> post.</p>" {
			t.Errorf("Content = %q", ctx.Content)
		}
		if ctx.WordCount != 7 {
			t.Errorf("WordCount = %d, want 7", ctx.WordCount)
		}
		if ctx.ReadingTime != 1 {
			t.Errorf("ReadingTime = %d, want 1", ctx.ReadingTime)
		}
	})

	t.Run("classification", func(t *testing.T) {
		if ctx.Draft != false {
			t.Errorf("Draft = %v, want false", ctx.Draft)
		}
		if ctx.Section != "blog" {
			t.Errorf("Section = %q, want %q", ctx.Section, "blog")
		}
		if ctx.Type != "single" {
			t.Errorf("Type = %q, want %q", ctx.Type, "single")
		}
	})

	t.Run("taxonomies", func(t *testing.T) {
		if len(ctx.Tags) != 2 || ctx.Tags[0] != "go" || ctx.Tags[1] != "testing" {
			t.Errorf("Tags = %v, want [go testing]", ctx.Tags)
		}
		if len(ctx.Categories) != 1 || ctx.Categories[0] != "tutorial" {
			t.Errorf("Categories = %v, want [tutorial]", ctx.Categories)
		}
		if ctx.Series != "learn-go" {
			t.Errorf("Series = %q, want %q", ctx.Series, "learn-go")
		}
	})

	t.Run("cover image", func(t *testing.T) {
		if ctx.Cover == nil {
			t.Fatal("Cover is nil")
		}
		if ctx.Cover.Image != "/images/cover.jpg" {
			t.Errorf("Cover.Image = %q, want %q", ctx.Cover.Image, "/images/cover.jpg")
		}
		if ctx.Cover.Alt != "Cover image" {
			t.Errorf("Cover.Alt = %q, want %q", ctx.Cover.Alt, "Cover image")
		}
		if ctx.Cover.Caption != "A nice cover" {
			t.Errorf("Cover.Caption = %q, want %q", ctx.Cover.Caption, "A nice cover")
		}
	})

	t.Run("params", func(t *testing.T) {
		if ctx.Params["custom"] != "value" {
			t.Errorf("Params[custom] = %v, want %q", ctx.Params["custom"], "value")
		}
	})

	t.Run("prev/next pages", func(t *testing.T) {
		if ctx.PrevPage == nil {
			t.Fatal("PrevPage is nil")
		}
		if ctx.PrevPage.Title != "Previous Post" {
			t.Errorf("PrevPage.Title = %q, want %q", ctx.PrevPage.Title, "Previous Post")
		}
		if ctx.PrevPage.URL != "/blog/prev-post/" {
			t.Errorf("PrevPage.URL = %q, want %q", ctx.PrevPage.URL, "/blog/prev-post/")
		}
		// PrevPage should NOT have its own PrevPage/NextPage (one level deep).
		if ctx.PrevPage.PrevPage != nil {
			t.Error("PrevPage.PrevPage should be nil (one level deep)")
		}

		if ctx.NextPage == nil {
			t.Fatal("NextPage is nil")
		}
		if ctx.NextPage.Title != "Next Post" {
			t.Errorf("NextPage.Title = %q, want %q", ctx.NextPage.Title, "Next Post")
		}
		if ctx.NextPage.NextPage != nil {
			t.Error("NextPage.NextPage should be nil (one level deep)")
		}
	})

	t.Run("nil cover image", func(t *testing.T) {
		pageNoCover := &content.Page{
			Title:   "No Cover",
			Type:    content.PageTypeSingle,
			Section: "blog",
		}
		ctxNoCover := r.BuildPageContext(pageNoCover, nil)
		if ctxNoCover.Cover != nil {
			t.Error("Cover should be nil when page has no cover")
		}
	})

	t.Run("nil prev/next", func(t *testing.T) {
		pageNoNav := &content.Page{
			Title:   "No Nav",
			Type:    content.PageTypeSingle,
			Section: "blog",
		}
		ctxNoNav := r.BuildPageContext(pageNoNav, nil)
		if ctxNoNav.PrevPage != nil {
			t.Error("PrevPage should be nil")
		}
		if ctxNoNav.NextPage != nil {
			t.Error("NextPage should be nil")
		}
	})
}

func TestBuildSiteContext(t *testing.T) {
	cfg := testConfig()
	md := content.NewMarkdownRenderer()

	themeDir := t.TempDir()
	layoutDir := filepath.Join(themeDir, "layouts", "_default")
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layoutDir, "single.html"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	engine, err := tmpl.NewEngine(themeDir, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	r := NewRenderer(engine, md, cfg)

	allPages := []*content.Page{
		{
			Title:      "Go Basics",
			Section:    "blog",
			Type:       content.PageTypeSingle,
			Tags:       []string{"go", "basics"},
			Categories: []string{"tutorial"},
		},
		{
			Title:      "Go Advanced",
			Section:    "blog",
			Type:       content.PageTypeSingle,
			Tags:       []string{"go", "advanced"},
			Categories: []string{"tutorial"},
		},
		{
			Title:      "Rust Intro",
			Section:    "blog",
			Type:       content.PageTypeSingle,
			Tags:       []string{"rust"},
			Categories: []string{"language"},
		},
		{
			Title:   "About Me",
			Section: "pages",
			Type:    content.PageTypeSingle,
		},
	}

	siteCtx := r.BuildSiteContext(allPages)

	t.Run("site metadata", func(t *testing.T) {
		if siteCtx.Title != "Test Site" {
			t.Errorf("Title = %q, want %q", siteCtx.Title, "Test Site")
		}
		if siteCtx.BaseURL != "https://example.com" {
			t.Errorf("BaseURL = %q, want %q", siteCtx.BaseURL, "https://example.com")
		}
		if siteCtx.Description != "A test site" {
			t.Errorf("Description = %q, want %q", siteCtx.Description, "A test site")
		}
		if siteCtx.Language != "en" {
			t.Errorf("Language = %q, want %q", siteCtx.Language, "en")
		}
	})

	t.Run("author", func(t *testing.T) {
		if siteCtx.Author.Name != "Test Author" {
			t.Errorf("Author.Name = %q, want %q", siteCtx.Author.Name, "Test Author")
		}
		if siteCtx.Author.Email != "test@example.com" {
			t.Errorf("Author.Email = %q, want %q", siteCtx.Author.Email, "test@example.com")
		}
		if siteCtx.Author.Social.GitHub != "testuser" {
			t.Errorf("Author.Social.GitHub = %q, want %q", siteCtx.Author.Social.GitHub, "testuser")
		}
		if siteCtx.Author.Social.Twitter != "@testuser" {
			t.Errorf("Author.Social.Twitter = %q, want %q", siteCtx.Author.Social.Twitter, "@testuser")
		}
	})

	t.Run("menu", func(t *testing.T) {
		if len(siteCtx.Menu) != 2 {
			t.Fatalf("Menu has %d items, want 2", len(siteCtx.Menu))
		}
		if siteCtx.Menu[0].Name != "Home" || siteCtx.Menu[0].URL != "/" {
			t.Errorf("Menu[0] = %+v, want Home /", siteCtx.Menu[0])
		}
		if siteCtx.Menu[1].Name != "Blog" || siteCtx.Menu[1].URL != "/blog/" {
			t.Errorf("Menu[1] = %+v, want Blog /blog/", siteCtx.Menu[1])
		}
	})

	t.Run("params", func(t *testing.T) {
		if siteCtx.Params["color"] != "blue" {
			t.Errorf("Params[color] = %v, want %q", siteCtx.Params["color"], "blue")
		}
	})

	t.Run("pages", func(t *testing.T) {
		if len(siteCtx.Pages) != 4 {
			t.Fatalf("Pages has %d items, want 4", len(siteCtx.Pages))
		}
		if siteCtx.Pages[0].Title != "Go Basics" {
			t.Errorf("Pages[0].Title = %q, want %q", siteCtx.Pages[0].Title, "Go Basics")
		}
	})

	t.Run("sections", func(t *testing.T) {
		blogPages, ok := siteCtx.Sections["blog"]
		if !ok {
			t.Fatal("Sections[blog] not found")
		}
		if len(blogPages) != 3 {
			t.Errorf("Sections[blog] has %d pages, want 3", len(blogPages))
		}

		pagesSection, ok := siteCtx.Sections["pages"]
		if !ok {
			t.Fatal("Sections[pages] not found")
		}
		if len(pagesSection) != 1 {
			t.Errorf("Sections[pages] has %d pages, want 1", len(pagesSection))
		}
		if pagesSection[0].Title != "About Me" {
			t.Errorf("Sections[pages][0].Title = %q, want %q", pagesSection[0].Title, "About Me")
		}
	})

	t.Run("taxonomies tags", func(t *testing.T) {
		tags, ok := siteCtx.Taxonomies["tags"]
		if !ok {
			t.Fatal("Taxonomies[tags] not found")
		}

		goPages, ok := tags["go"]
		if !ok {
			t.Fatal("tags[go] not found")
		}
		if len(goPages) != 2 {
			t.Errorf("tags[go] has %d pages, want 2", len(goPages))
		}

		rustPages, ok := tags["rust"]
		if !ok {
			t.Fatal("tags[rust] not found")
		}
		if len(rustPages) != 1 {
			t.Errorf("tags[rust] has %d pages, want 1", len(rustPages))
		}

		basicPages, ok := tags["basics"]
		if !ok {
			t.Fatal("tags[basics] not found")
		}
		if len(basicPages) != 1 {
			t.Errorf("tags[basics] has %d pages, want 1", len(basicPages))
		}
	})

	t.Run("taxonomies categories", func(t *testing.T) {
		cats, ok := siteCtx.Taxonomies["categories"]
		if !ok {
			t.Fatal("Taxonomies[categories] not found")
		}

		tutorialPages, ok := cats["tutorial"]
		if !ok {
			t.Fatal("categories[tutorial] not found")
		}
		if len(tutorialPages) != 2 {
			t.Errorf("categories[tutorial] has %d pages, want 2", len(tutorialPages))
		}

		langPages, ok := cats["language"]
		if !ok {
			t.Fatal("categories[language] not found")
		}
		if len(langPages) != 1 {
			t.Errorf("categories[language] has %d pages, want 1", len(langPages))
		}
	})

	t.Run("data initialized", func(t *testing.T) {
		if siteCtx.Data == nil {
			t.Error("Data should be initialized, not nil")
		}
	})

	t.Run("build date set", func(t *testing.T) {
		if siteCtx.BuildDate.IsZero() {
			t.Error("BuildDate should be set")
		}
		if time.Since(siteCtx.BuildDate) > 5*time.Second {
			t.Error("BuildDate should be approximately now")
		}
	})
}

func TestBuildSiteContextNoTaxonomies(t *testing.T) {
	cfg := testConfig()
	md := content.NewMarkdownRenderer()

	themeDir := t.TempDir()
	layoutDir := filepath.Join(themeDir, "layouts", "_default")
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layoutDir, "single.html"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	engine, err := tmpl.NewEngine(themeDir, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	r := NewRenderer(engine, md, cfg)

	// Pages with no tags or categories.
	allPages := []*content.Page{
		{Title: "Plain Page", Section: "blog", Type: content.PageTypeSingle},
	}

	siteCtx := r.BuildSiteContext(allPages)

	if _, ok := siteCtx.Taxonomies["tags"]; ok {
		t.Error("Taxonomies[tags] should not exist when no pages have tags")
	}
	if _, ok := siteCtx.Taxonomies["categories"]; ok {
		t.Error("Taxonomies[categories] should not exist when no pages have categories")
	}
}

func TestRenderPage(t *testing.T) {
	cfg := testConfig()
	md := content.NewMarkdownRenderer()

	// Create a temporary theme directory with a simple single template.
	themeDir := t.TempDir()
	layoutDir := filepath.Join(themeDir, "layouts", "_default")
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		t.Fatal(err)
	}

	singleTemplate := `<article><h1>{{ .Title }}</h1><div>{{ .Content }}</div><p>Site: {{ .Site.Title }}</p></article>`
	if err := os.WriteFile(filepath.Join(layoutDir, "single.html"), []byte(singleTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, err := tmpl.NewEngine(themeDir, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	r := NewRenderer(engine, md, cfg)

	page := &content.Page{
		Title:      "Render Test",
		RawContent: "Hello **world**!\n\nThis is a test.",
		Type:       content.PageTypeSingle,
		Section:    "blog",
		Tags:       []string{"test"},
	}

	allPages := []*content.Page{page}

	output, err := r.RenderPage(page, allPages)
	if err != nil {
		t.Fatalf("RenderPage failed: %v", err)
	}

	result := string(output)

	// Verify the template was rendered with content.
	if !strings.Contains(result, "<h1>Render Test</h1>") {
		t.Errorf("output should contain title, got: %s", result)
	}
	if !strings.Contains(result, "<strong>world</strong>") {
		t.Errorf("output should contain rendered markdown bold, got: %s", result)
	}
	if !strings.Contains(result, "Site: Test Site") {
		t.Errorf("output should contain site title, got: %s", result)
	}
	if !strings.Contains(result, "<article>") {
		t.Errorf("output should contain article wrapper from template, got: %s", result)
	}

	// Verify the page's Content field was updated.
	if page.Content == "" {
		t.Error("page.Content should be populated after RenderPage")
	}
	if !strings.Contains(page.Content, "<strong>world</strong>") {
		t.Errorf("page.Content should contain rendered HTML, got: %s", page.Content)
	}
}

func TestRenderPageWithTOC(t *testing.T) {
	cfg := testConfig()
	md := content.NewMarkdownRenderer()

	themeDir := t.TempDir()
	layoutDir := filepath.Join(themeDir, "layouts", "_default")
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tocTemplate := `<nav>{{ .TableOfContents }}</nav><div>{{ .Content }}</div>`
	if err := os.WriteFile(filepath.Join(layoutDir, "single.html"), []byte(tocTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, err := tmpl.NewEngine(themeDir, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	r := NewRenderer(engine, md, cfg)

	page := &content.Page{
		Title:      "TOC Test",
		RawContent: "# First Heading\n\nContent one.\n\n## Second Heading\n\nContent two.\n\n## Third Heading\n\nContent three.\n",
		Type:       content.PageTypeSingle,
		Section:    "blog",
	}

	output, err := r.RenderPage(page, []*content.Page{page})
	if err != nil {
		t.Fatalf("RenderPage failed: %v", err)
	}

	result := string(output)

	// The TOC should contain nav element with list items for the headings.
	if !strings.Contains(result, "<nav>") {
		t.Errorf("output should contain nav wrapper, got: %s", result)
	}
	// The page content should contain the rendered headings.
	if !strings.Contains(result, "First Heading") {
		t.Errorf("output should contain first heading, got: %s", result)
	}

	// Verify the page's TableOfContents was populated.
	if page.TableOfContents == "" {
		t.Error("page.TableOfContents should be populated after RenderPage")
	}
}

func TestRenderPageNoTemplate(t *testing.T) {
	cfg := testConfig()
	md := content.NewMarkdownRenderer()

	// Create a theme with no matching templates.
	themeDir := t.TempDir()
	layoutDir := filepath.Join(themeDir, "layouts", "_default")
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Only create a list template, not a single template.
	if err := os.WriteFile(filepath.Join(layoutDir, "list.html"), []byte("<ul></ul>"), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, err := tmpl.NewEngine(themeDir, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	r := NewRenderer(engine, md, cfg)

	page := &content.Page{
		Title:      "No Template",
		RawContent: "Some content.",
		Type:       content.PageTypeSingle,
		Section:    "blog",
	}

	_, err = r.RenderPage(page, []*content.Page{page})
	if err == nil {
		t.Fatal("RenderPage should fail when no matching template exists")
	}
	if !strings.Contains(err.Error(), "no template found") {
		t.Errorf("error should mention no template found, got: %v", err)
	}
}

func TestRenderPageHomeType(t *testing.T) {
	cfg := testConfig()
	md := content.NewMarkdownRenderer()

	themeDir := t.TempDir()
	layoutDir := filepath.Join(themeDir, "layouts")
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		t.Fatal(err)
	}

	indexTemplate := `<main>{{ .Title }} - {{ len .Site.Pages }} pages</main>`
	if err := os.WriteFile(filepath.Join(layoutDir, "index.html"), []byte(indexTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, err := tmpl.NewEngine(themeDir, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	r := NewRenderer(engine, md, cfg)

	homePage := &content.Page{
		Title:      "Welcome",
		RawContent: "Home page content.",
		Type:       content.PageTypeHome,
	}
	blogPage := &content.Page{
		Title:   "Blog Post",
		Section: "blog",
		Type:    content.PageTypeSingle,
	}

	allPages := []*content.Page{homePage, blogPage}

	output, err := r.RenderPage(homePage, allPages)
	if err != nil {
		t.Fatalf("RenderPage for home failed: %v", err)
	}

	result := string(output)
	if !strings.Contains(result, "Welcome") {
		t.Errorf("output should contain home page title, got: %s", result)
	}
	if !strings.Contains(result, "2 pages") {
		t.Errorf("output should contain page count, got: %s", result)
	}
}

func TestRenderPageListType(t *testing.T) {
	cfg := testConfig()
	md := content.NewMarkdownRenderer()

	themeDir := t.TempDir()
	layoutDir := filepath.Join(themeDir, "layouts", "_default")
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		t.Fatal(err)
	}

	listTemplate := `<h1>{{ .Title }}</h1><p>{{ .Type }}</p>`
	if err := os.WriteFile(filepath.Join(layoutDir, "list.html"), []byte(listTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, err := tmpl.NewEngine(themeDir, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	r := NewRenderer(engine, md, cfg)

	page := &content.Page{
		Title:      "Blog Posts",
		RawContent: "",
		Type:       content.PageTypeList,
		Section:    "blog",
	}

	output, err := r.RenderPage(page, []*content.Page{page})
	if err != nil {
		t.Fatalf("RenderPage for list failed: %v", err)
	}

	result := string(output)
	if !strings.Contains(result, "Blog Posts") {
		t.Errorf("output should contain list title, got: %s", result)
	}
	if !strings.Contains(result, "list") {
		t.Errorf("output should contain type 'list', got: %s", result)
	}
}

func TestPageTypeMapping(t *testing.T) {
	// Verify all PageType values map to the correct strings.
	tests := []struct {
		pageType content.PageType
		want     string
	}{
		{content.PageTypeSingle, "single"},
		{content.PageTypeList, "list"},
		{content.PageTypeHome, "home"},
		{content.PageTypeTaxonomy, "taxonomy"},
		{content.PageTypeTaxonomyList, "taxonomylist"},
	}

	for _, tt := range tests {
		got := tt.pageType.String()
		if got != tt.want {
			t.Errorf("PageType(%d).String() = %q, want %q", tt.pageType, got, tt.want)
		}
	}
}
