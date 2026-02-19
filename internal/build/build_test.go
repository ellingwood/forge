package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aellingwood/forge/internal/config"
	"github.com/aellingwood/forge/internal/content"
)

// --- Writer utility tests ---

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name     string
		url      string
		data     string
		wantPath string
	}{
		{
			name:     "root URL",
			url:      "/",
			data:     "<html>home</html>",
			wantPath: "index.html",
		},
		{
			name:     "section URL with trailing slash",
			url:      "/blog/my-post/",
			data:     "<html>post</html>",
			wantPath: "blog/my-post/index.html",
		},
		{
			name:     "section URL without trailing slash",
			url:      "/about",
			data:     "<html>about</html>",
			wantPath: "about/index.html",
		},
		{
			name:     "deeply nested URL",
			url:      "/a/b/c/d/",
			data:     "<html>deep</html>",
			wantPath: "a/b/c/d/index.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subDir := filepath.Join(dir, tt.name)
			if err := os.MkdirAll(subDir, 0o755); err != nil {
				t.Fatal(err)
			}

			if err := WriteFile(subDir, tt.url, []byte(tt.data)); err != nil {
				t.Fatalf("WriteFile(%q, %q) error: %v", subDir, tt.url, err)
			}

			filePath := filepath.Join(subDir, tt.wantPath)
			got, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("reading written file %s: %v", filePath, err)
			}
			if string(got) != tt.data {
				t.Errorf("file content = %q, want %q", string(got), tt.data)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "source.txt")
	dstPath := filepath.Join(dir, "sub", "dest.txt")

	content := "hello world"
	if err := os.WriteFile(srcPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("reading copied file: %v", err)
	}
	if string(got) != content {
		t.Errorf("copied content = %q, want %q", string(got), content)
	}
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "output")

	// Create a directory structure.
	if err := os.MkdirAll(filepath.Join(src, "css"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "js"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "css", "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "js", "app.js"), []byte("alert(1)"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "robots.txt"), []byte("User-agent: *"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir error: %v", err)
	}

	// Verify files exist.
	wantFiles := []struct {
		path    string
		content string
	}{
		{"css/style.css", "body{}"},
		{"js/app.js", "alert(1)"},
		{"robots.txt", "User-agent: *"},
	}

	for _, wf := range wantFiles {
		got, err := os.ReadFile(filepath.Join(dst, wf.path))
		if err != nil {
			t.Errorf("reading %s: %v", wf.path, err)
			continue
		}
		if string(got) != wf.content {
			t.Errorf("%s content = %q, want %q", wf.path, string(got), wf.content)
		}
	}
}

func TestCleanDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "output")

	// Create a directory with some content.
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "nested.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CleanDir(dir); err != nil {
		t.Fatalf("CleanDir error: %v", err)
	}

	// Directory should exist but be empty.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading cleaned dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("cleaned dir has %d entries, want 0", len(entries))
	}
}

func TestCleanDir_NonExistent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")

	if err := CleanDir(dir); err != nil {
		t.Fatalf("CleanDir on nonexistent dir error: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat after CleanDir: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory after CleanDir")
	}
}

func TestDirSize(t *testing.T) {
	dir := t.TempDir()

	data1 := []byte("hello") // 5 bytes
	data2 := []byte("world!") // 6 bytes

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), data1, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "b.txt"), data2, 0o644); err != nil {
		t.Fatal(err)
	}

	size, err := DirSize(dir)
	if err != nil {
		t.Fatalf("DirSize error: %v", err)
	}

	want := int64(len(data1) + len(data2))
	if size != want {
		t.Errorf("DirSize = %d, want %d", size, want)
	}
}

func TestDirSize_NonExistent(t *testing.T) {
	size, err := DirSize(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("DirSize on nonexistent dir error: %v", err)
	}
	if size != 0 {
		t.Errorf("DirSize of nonexistent = %d, want 0", size)
	}
}

// --- Pipeline tests ---

func TestRenderParallel(t *testing.T) {
	pages := []*content.Page{
		{Title: "A", SourcePath: "a.md", RawContent: "alpha"},
		{Title: "B", SourcePath: "b.md", RawContent: "beta"},
		{Title: "C", SourcePath: "c.md", RawContent: "gamma"},
	}

	err := renderParallel(pages, 2, func(p *content.Page) error {
		p.Content = strings.ToUpper(p.RawContent)
		return nil
	})
	if err != nil {
		t.Fatalf("renderParallel error: %v", err)
	}

	for _, p := range pages {
		want := strings.ToUpper(p.RawContent)
		if p.Content != want {
			t.Errorf("page %s: Content = %q, want %q", p.Title, p.Content, want)
		}
	}
}

func TestRenderParallel_Empty(t *testing.T) {
	err := renderParallel(nil, 4, func(p *content.Page) error {
		return nil
	})
	if err != nil {
		t.Fatalf("renderParallel with empty pages: %v", err)
	}
}

func TestRenderParallel_Error(t *testing.T) {
	pages := []*content.Page{
		{Title: "A", SourcePath: "a.md"},
		{Title: "B", SourcePath: "b.md"},
	}

	err := renderParallel(pages, 1, func(p *content.Page) error {
		if p.Title == "A" {
			return os.ErrInvalid
		}
		return nil
	})
	if err == nil {
		t.Fatal("expected error from renderParallel, got nil")
	}
}

func TestSetSectionNavigation(t *testing.T) {
	p1 := &content.Page{Title: "Post 1", Type: content.PageTypeSingle, Section: "blog"}
	p2 := &content.Page{Title: "Post 2", Type: content.PageTypeSingle, Section: "blog"}
	p3 := &content.Page{Title: "Post 3", Type: content.PageTypeSingle, Section: "blog"}
	p4 := &content.Page{Title: "Project", Type: content.PageTypeSingle, Section: "projects"}
	// Pages already sorted newest first.
	pages := []*content.Page{p1, p2, p3, p4}

	setSectionNavigation(pages)

	// p1 (newest blog): no next (newer), prev = p2 (older)
	if p1.NextPage != nil {
		t.Error("p1 NextPage should be nil")
	}
	if p1.PrevPage != p2 {
		t.Error("p1 PrevPage should be p2")
	}

	// p2 (middle blog): next = p1 (newer), prev = p3 (older)
	if p2.NextPage != p1 {
		t.Error("p2 NextPage should be p1")
	}
	if p2.PrevPage != p3 {
		t.Error("p2 PrevPage should be p3")
	}

	// p3 (oldest blog): next = p2 (newer), prev = nil
	if p3.NextPage != p2 {
		t.Error("p3 NextPage should be p2")
	}
	if p3.PrevPage != nil {
		t.Error("p3 PrevPage should be nil")
	}

	// p4 (only project): no prev or next
	if p4.PrevPage != nil || p4.NextPage != nil {
		t.Error("p4 should have no navigation links (only page in section)")
	}
}

func TestBuildTaxonomyMaps(t *testing.T) {
	pages := []*content.Page{
		{
			Title: "Go Post",
			Tags:  []string{"go", "programming"},
			Categories: []string{"tech"},
		},
		{
			Title: "Rust Post",
			Tags:  []string{"rust", "programming"},
			Categories: []string{"tech"},
		},
		{
			Title: "Travel Post",
			Tags:  []string{"travel"},
			Categories: []string{"personal"},
		},
	}

	tags, categories := buildTaxonomyMaps(pages)

	if len(tags["go"]) != 1 {
		t.Errorf("tags[go] = %d, want 1", len(tags["go"]))
	}
	if len(tags["programming"]) != 2 {
		t.Errorf("tags[programming] = %d, want 2", len(tags["programming"]))
	}
	if len(tags["travel"]) != 1 {
		t.Errorf("tags[travel] = %d, want 1", len(tags["travel"]))
	}
	if len(categories["tech"]) != 2 {
		t.Errorf("categories[tech] = %d, want 2", len(categories["tech"]))
	}
	if len(categories["personal"]) != 1 {
		t.Errorf("categories[personal] = %d, want 1", len(categories["personal"]))
	}
}

// --- Full build pipeline test ---

// setupTestSite creates a temporary project directory with content, theme, and config.
func setupTestSite(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// Create directory structure.
	dirs := []string{
		"content/blog",
		"themes/default/layouts/_default",
		"themes/default/layouts/partials",
		"themes/default/static/css",
		"static/images",
		"layouts",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Create config file.
	configContent := `title: "Test Site"
baseURL: "https://example.com"
language: "en"
theme: "default"
`
	if err := os.WriteFile(filepath.Join(root, "forge.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create content files.
	// Home page.
	homeContent := `---
title: "Home"
---
Welcome to my site.
`
	if err := os.WriteFile(filepath.Join(root, "content", "_index.md"), []byte(homeContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Blog post 1 (published).
	post1 := `---
title: "First Post"
date: 2024-01-15
tags:
  - go
  - programming
categories:
  - tech
---
This is my **first** post.
`
	if err := os.WriteFile(filepath.Join(root, "content", "blog", "first-post.md"), []byte(post1), 0o644); err != nil {
		t.Fatal(err)
	}

	// Blog post 2 (published).
	post2 := `---
title: "Second Post"
date: 2024-02-20
tags:
  - go
---
This is my **second** post.
`
	if err := os.WriteFile(filepath.Join(root, "content", "blog", "second-post.md"), []byte(post2), 0o644); err != nil {
		t.Fatal(err)
	}

	// Draft post.
	draftPost := `---
title: "Draft Post"
date: 2024-03-01
draft: true
---
This is a draft.
`
	if err := os.WriteFile(filepath.Join(root, "content", "blog", "draft-post.md"), []byte(draftPost), 0o644); err != nil {
		t.Fatal(err)
	}

	// Blog section page.
	blogSection := `---
title: "Blog"
---
All blog posts.
`
	if err := os.WriteFile(filepath.Join(root, "content", "blog", "_index.md"), []byte(blogSection), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create theme templates.
	// baseof is not needed if we use simple templates.
	singleTemplate := `<!DOCTYPE html>
<html>
<head><title>{{ .Title }}</title></head>
<body>{{ .Content }}</body>
</html>`
	if err := os.WriteFile(
		filepath.Join(root, "themes", "default", "layouts", "_default", "single.html"),
		[]byte(singleTemplate), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	listTemplate := `<!DOCTYPE html>
<html>
<head><title>{{ .Title }}</title></head>
<body><h1>{{ .Title }}</h1>{{ .Content }}</body>
</html>`
	if err := os.WriteFile(
		filepath.Join(root, "themes", "default", "layouts", "_default", "list.html"),
		[]byte(listTemplate), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	indexTemplate := `<!DOCTYPE html>
<html>
<head><title>{{ .Title }}</title></head>
<body><h1>{{ .Title }}</h1>{{ .Content }}</body>
</html>`
	if err := os.WriteFile(
		filepath.Join(root, "themes", "default", "layouts", "index.html"),
		[]byte(indexTemplate), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Create static files.
	if err := os.WriteFile(
		filepath.Join(root, "themes", "default", "static", "css", "style.css"),
		[]byte("body { margin: 0; }"), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "static", "images", "logo.png"),
		[]byte("fake-png-data"), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestBuild_FullPipeline(t *testing.T) {
	root := setupTestSite(t)
	outputDir := filepath.Join(root, "public")

	cfg := config.Default()
	cfg.Title = "Test Site"
	cfg.BaseURL = "https://example.com"
	cfg.Theme = "default"

	builder := NewBuilder(cfg, BuildOptions{
		ProjectRoot: root,
		OutputDir:   outputDir,
	})

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Verify result metrics.
	if result.PagesRendered < 3 {
		t.Errorf("PagesRendered = %d, want >= 3 (home + blog section + 2 posts - 1 draft)", result.PagesRendered)
	}
	if result.FilesWritten < 3 {
		t.Errorf("FilesWritten = %d, want >= 3", result.FilesWritten)
	}
	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}
	if result.OutputSize <= 0 {
		t.Error("OutputSize should be positive")
	}

	// Verify output files exist.
	wantFiles := []string{
		"index.html",               // Home page
		"blog/index.html",          // Blog section
		"blog/first-post/index.html",  // First post
		"blog/second-post/index.html", // Second post
	}
	for _, f := range wantFiles {
		path := filepath.Join(outputDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected output file %s to exist", f)
		}
	}

	// Verify draft was NOT included.
	draftPath := filepath.Join(outputDir, "blog", "draft-post", "index.html")
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Error("draft post should not be in output when IncludeDrafts is false")
	}

	// Verify static files were copied.
	staticFiles := []string{
		"css/style.css",       // From theme
		"images/logo.png",     // From site
	}
	for _, f := range staticFiles {
		path := filepath.Join(outputDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected static file %s to exist", f)
		}
	}

	// Verify HTML content of a rendered page.
	postHTML, err := os.ReadFile(filepath.Join(outputDir, "blog", "first-post", "index.html"))
	if err != nil {
		t.Fatalf("reading first post: %v", err)
	}
	postContent := string(postHTML)
	if !strings.Contains(postContent, "First Post") {
		t.Error("first post HTML should contain the title")
	}
	if !strings.Contains(postContent, "<strong>first</strong>") {
		t.Error("first post HTML should contain rendered markdown bold")
	}
}

func TestBuild_IncludeDrafts(t *testing.T) {
	root := setupTestSite(t)
	outputDir := filepath.Join(root, "public")

	cfg := config.Default()
	cfg.Title = "Test Site"
	cfg.BaseURL = "https://example.com"
	cfg.Theme = "default"

	builder := NewBuilder(cfg, BuildOptions{
		ProjectRoot:   root,
		OutputDir:     outputDir,
		IncludeDrafts: true,
	})

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() with drafts error: %v", err)
	}

	// Draft should now be present.
	draftPath := filepath.Join(outputDir, "blog", "draft-post", "index.html")
	if _, err := os.Stat(draftPath); os.IsNotExist(err) {
		t.Error("draft post should be in output when IncludeDrafts is true")
	}

	// Should have more pages rendered than without drafts.
	if result.PagesRendered < 4 {
		t.Errorf("PagesRendered = %d with drafts, want >= 4", result.PagesRendered)
	}
}

func TestBuild_FilterFuture(t *testing.T) {
	root := setupTestSite(t)

	// Create a future-dated post.
	futureDate := time.Now().Add(24 * time.Hour * 365).Format("2006-01-02")
	futurePost := `---
title: "Future Post"
date: ` + futureDate + `
---
This is from the future.
`
	if err := os.WriteFile(filepath.Join(root, "content", "blog", "future-post.md"), []byte(futurePost), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(root, "public")

	cfg := config.Default()
	cfg.Title = "Test Site"
	cfg.BaseURL = "https://example.com"
	cfg.Theme = "default"

	// Without IncludeFuture (default) - future post should be excluded.
	builder := NewBuilder(cfg, BuildOptions{
		ProjectRoot: root,
		OutputDir:   outputDir,
	})
	_, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	futurePath := filepath.Join(outputDir, "blog", "future-post", "index.html")
	if _, err := os.Stat(futurePath); !os.IsNotExist(err) {
		t.Error("future post should not be in output when IncludeFuture is false")
	}

	// With IncludeFuture - future post should be included.
	builder2 := NewBuilder(cfg, BuildOptions{
		ProjectRoot:   root,
		OutputDir:     outputDir,
		IncludeFuture: true,
	})
	_, err = builder2.Build()
	if err != nil {
		t.Fatalf("Build() with future error: %v", err)
	}

	if _, err := os.Stat(futurePath); os.IsNotExist(err) {
		t.Error("future post should be in output when IncludeFuture is true")
	}
}

func TestBuild_PageBundleAssets(t *testing.T) {
	root := setupTestSite(t)

	// Create a page bundle.
	bundleDir := filepath.Join(root, "content", "blog", "bundle-post")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}

	bundleContent := `---
title: "Bundle Post"
date: 2024-04-01
---
A post with assets.
`
	if err := os.WriteFile(filepath.Join(bundleDir, "index.md"), []byte(bundleContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "hero.jpg"), []byte("fake-jpg-data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "diagram.svg"), []byte("<svg></svg>"), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(root, "public")
	cfg := config.Default()
	cfg.Title = "Test Site"
	cfg.BaseURL = "https://example.com"
	cfg.Theme = "default"

	builder := NewBuilder(cfg, BuildOptions{
		ProjectRoot: root,
		OutputDir:   outputDir,
	})

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Verify the bundle page was rendered.
	postPath := filepath.Join(outputDir, "blog", "bundle-post", "index.html")
	if _, err := os.Stat(postPath); os.IsNotExist(err) {
		t.Error("bundle post page should exist")
	}

	// Verify bundle assets were copied.
	heroPath := filepath.Join(outputDir, "blog", "bundle-post", "hero.jpg")
	if _, err := os.Stat(heroPath); os.IsNotExist(err) {
		t.Error("bundle asset hero.jpg should be copied to output")
	}
	diagramPath := filepath.Join(outputDir, "blog", "bundle-post", "diagram.svg")
	if _, err := os.Stat(diagramPath); os.IsNotExist(err) {
		t.Error("bundle asset diagram.svg should be copied to output")
	}

	// Verify FilesCopied count includes bundle assets.
	if result.FilesCopied < 2 {
		t.Errorf("FilesCopied = %d, want >= 2 (at least bundle assets)", result.FilesCopied)
	}
}

func TestBuild_OutputStructure(t *testing.T) {
	root := setupTestSite(t)
	outputDir := filepath.Join(root, "public")

	cfg := config.Default()
	cfg.Title = "Test Site"
	cfg.BaseURL = "https://example.com"
	cfg.Theme = "default"

	builder := NewBuilder(cfg, BuildOptions{
		ProjectRoot: root,
		OutputDir:   outputDir,
	})

	_, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Verify the output directory structure is correct.
	// Each page URL becomes a directory with index.html.
	expectedDirs := []string{
		"blog",
		"blog/first-post",
		"blog/second-post",
	}
	for _, d := range expectedDirs {
		path := filepath.Join(outputDir, d)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory", d)
		}
	}
}

func TestNewBuilder(t *testing.T) {
	cfg := config.Default()
	cfg.Title = "My Site"

	opts := BuildOptions{
		IncludeDrafts: true,
		OutputDir:     "/tmp/output",
		Verbose:       true,
	}

	b := NewBuilder(cfg, opts)
	if b == nil {
		t.Fatal("NewBuilder returned nil")
	}
	if b.config.Title != "My Site" {
		t.Errorf("config.Title = %q, want %q", b.config.Title, "My Site")
	}
	if !b.options.IncludeDrafts {
		t.Error("options.IncludeDrafts should be true")
	}
	if b.options.OutputDir != "/tmp/output" {
		t.Errorf("options.OutputDir = %q, want %q", b.options.OutputDir, "/tmp/output")
	}
}

func TestPageToContext(t *testing.T) {
	page := &content.Page{
		Title:       "Test Page",
		Description: "A test",
		Content:     "<p>hello</p>",
		Summary:     "hello",
		Slug:        "test-page",
		URL:         "/test-page/",
		Permalink:   "https://example.com/test-page/",
		Section:     "blog",
		Type:        content.PageTypeSingle,
		Tags:        []string{"go"},
		Categories:  []string{"tech"},
		WordCount:   100,
		ReadingTime: 1,
		Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Cover: &content.CoverImage{
			Image:   "/images/cover.jpg",
			Alt:     "Cover image",
			Caption: "A caption",
		},
	}

	ctx := pageToContext(page, nil)

	if ctx.Title != "Test Page" {
		t.Errorf("Title = %q, want %q", ctx.Title, "Test Page")
	}
	if ctx.Slug != "test-page" {
		t.Errorf("Slug = %q, want %q", ctx.Slug, "test-page")
	}
	if ctx.URL != "/test-page/" {
		t.Errorf("URL = %q, want %q", ctx.URL, "/test-page/")
	}
	if ctx.Type != "single" {
		t.Errorf("Type = %q, want %q", ctx.Type, "single")
	}
	if ctx.Cover == nil {
		t.Fatal("Cover should not be nil")
	}
	if ctx.Cover.Image != "/images/cover.jpg" {
		t.Errorf("Cover.Image = %q, want %q", ctx.Cover.Image, "/images/cover.jpg")
	}
	if ctx.Site != nil {
		t.Error("Site should be nil when nil is passed")
	}
}

func TestBuild_CleanOutput(t *testing.T) {
	root := setupTestSite(t)
	outputDir := filepath.Join(root, "public")

	// Pre-populate the output directory with a stale file.
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	staleFile := filepath.Join(outputDir, "stale.html")
	if err := os.WriteFile(staleFile, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Title = "Test Site"
	cfg.BaseURL = "https://example.com"
	cfg.Theme = "default"

	builder := NewBuilder(cfg, BuildOptions{
		ProjectRoot: root,
		OutputDir:   outputDir,
	})

	_, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// The stale file should have been removed by CleanDir.
	if _, err := os.Stat(staleFile); !os.IsNotExist(err) {
		t.Error("stale file should have been removed during build")
	}
}
