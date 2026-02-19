package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixedTime is used by tests to make output deterministic.
var fixedTime = time.Date(2025, 6, 15, 10, 30, 0, 0, time.FixedZone("UTC-5", -5*3600))

func init() {
	nowFunc = func() time.Time { return fixedTime }
}

// ---------------------------------------------------------------------------
// TestSlugify
// ---------------------------------------------------------------------------

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"My First Post", "my-first-post"},
		{"Already-Slugified", "already-slugified"},
		{"  leading and trailing spaces  ", "leading-and-trailing-spaces"},
		{"Special!@#$%^&*()Characters", "specialcharacters"},
		{"Multiple---Hyphens", "multiple-hyphens"},
		{"under_scores_too", "under-scores-too"},
		{"MiXeD CaSe", "mixed-case"},
		{"123 Numbers 456", "123-numbers-456"},
		{"---leading-hyphens---", "leading-hyphens"},
		{"", ""},
		{"cafe\u0301", "caf\u00e9"},       // composed after ToLower
		{"\u00fcber cool", "\u00fcber-cool"}, // German u-umlaut
		{"\u4f60\u597d world", "\u4f60\u597d-world"}, // Chinese characters
		{"one - two - three", "one-two-three"},
		{"a!b@c#d", "abcd"},
	}

	for _, tc := range tests {
		got := Slugify(tc.input)
		if got != tc.want {
			t.Errorf("Slugify(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// TestNewSite
// ---------------------------------------------------------------------------

func TestNewSite(t *testing.T) {
	dir := t.TempDir()
	siteName := filepath.Join(dir, "my-site")

	if err := NewSite(siteName); err != nil {
		t.Fatalf("NewSite(%q): %v", siteName, err)
	}

	// Verify directory structure.
	expectedDirs := []string{
		"content/blog",
		"content/projects",
		"content/pages",
		"layouts",
		"static",
		"data",
		"assets",
	}

	for _, d := range expectedDirs {
		fullPath := filepath.Join(siteName, d)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("expected directory %q to exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %q to be a directory", d)
		}
	}

	// Verify forge.yaml exists and contains the site name.
	configData, err := os.ReadFile(filepath.Join(siteName, "forge.yaml"))
	if err != nil {
		t.Fatalf("reading forge.yaml: %v", err)
	}
	configStr := string(configData)
	if !strings.Contains(configStr, `title: "my-site"`) {
		t.Errorf("forge.yaml should contain site title, got:\n%s", configStr)
	}
	if !strings.Contains(configStr, `baseURL: "http://localhost:1313"`) {
		t.Errorf("forge.yaml should contain baseURL, got:\n%s", configStr)
	}
	if !strings.Contains(configStr, `language: "en"`) {
		t.Errorf("forge.yaml should contain language, got:\n%s", configStr)
	}
	if !strings.Contains(configStr, `theme: "default"`) {
		t.Errorf("forge.yaml should contain theme, got:\n%s", configStr)
	}

	// Verify about page.
	aboutData, err := os.ReadFile(filepath.Join(siteName, "content", "pages", "about.md"))
	if err != nil {
		t.Fatalf("reading about.md: %v", err)
	}
	aboutStr := string(aboutData)
	if !strings.Contains(aboutStr, `title: "About"`) {
		t.Errorf("about.md should contain title, got:\n%s", aboutStr)
	}
	if !strings.Contains(aboutStr, `layout: "page"`) {
		t.Errorf("about.md should contain layout, got:\n%s", aboutStr)
	}

	// Verify blog post with date prefix.
	postPath := filepath.Join(siteName, "content", "blog", "2025-06-15-hello-world.md")
	postData, err := os.ReadFile(postPath)
	if err != nil {
		t.Fatalf("reading hello-world.md: %v", err)
	}
	postStr := string(postData)
	if !strings.Contains(postStr, `title: "Hello World"`) {
		t.Errorf("hello-world.md should contain title, got:\n%s", postStr)
	}
	if !strings.Contains(postStr, "draft: true") {
		t.Errorf("hello-world.md should contain draft: true, got:\n%s", postStr)
	}
}

func TestNewSite_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	siteName := filepath.Join(dir, "existing-site")

	// Create the directory first.
	if err := os.Mkdir(siteName, 0o755); err != nil {
		t.Fatalf("creating test directory: %v", err)
	}

	err := NewSite(siteName)
	if err == nil {
		t.Fatal("expected error when directory already exists, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestNewPost
// ---------------------------------------------------------------------------

func TestNewPost(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("changing to temp directory: %v", err)
	}

	title := "My First Post"
	if err := NewPost(title); err != nil {
		t.Fatalf("NewPost(%q): %v", title, err)
	}

	expectedPath := filepath.Join("content", "blog", "2025-06-15-my-first-post.md")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("reading created post: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `title: "My First Post"`) {
		t.Errorf("post should contain title, got:\n%s", content)
	}
	if !strings.Contains(content, "date: 2025-06-15T10:30:00-05:00") {
		t.Errorf("post should contain date, got:\n%s", content)
	}
	if !strings.Contains(content, "draft: true") {
		t.Errorf("post should contain draft: true, got:\n%s", content)
	}
	if !strings.Contains(content, "tags: []") {
		t.Errorf("post should contain tags: [], got:\n%s", content)
	}
	if !strings.Contains(content, "categories: []") {
		t.Errorf("post should contain categories: [], got:\n%s", content)
	}
	if !strings.Contains(content, "Write your post content here.") {
		t.Errorf("post should contain placeholder content, got:\n%s", content)
	}
}

func TestNewPost_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("changing to temp directory: %v", err)
	}

	// content/blog does not exist yet; NewPost should create it.
	if err := NewPost("Test Post"); err != nil {
		t.Fatalf("NewPost should create parent dirs: %v", err)
	}

	info, err := os.Stat(filepath.Join("content", "blog"))
	if err != nil {
		t.Fatalf("content/blog should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("content/blog should be a directory")
	}
}

// ---------------------------------------------------------------------------
// TestNewPage
// ---------------------------------------------------------------------------

func TestNewPage(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("changing to temp directory: %v", err)
	}

	title := "Contact Us"
	if err := NewPage(title); err != nil {
		t.Fatalf("NewPage(%q): %v", title, err)
	}

	expectedPath := filepath.Join("content", "pages", "contact-us.md")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("reading created page: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `title: "Contact Us"`) {
		t.Errorf("page should contain title, got:\n%s", content)
	}
	if !strings.Contains(content, `layout: "page"`) {
		t.Errorf("page should contain layout, got:\n%s", content)
	}
	if !strings.Contains(content, "Write your page content here.") {
		t.Errorf("page should contain placeholder content, got:\n%s", content)
	}
}

func TestNewPage_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("changing to temp directory: %v", err)
	}

	if err := NewPage("Test Page"); err != nil {
		t.Fatalf("NewPage should create parent dirs: %v", err)
	}

	info, err := os.Stat(filepath.Join("content", "pages"))
	if err != nil {
		t.Fatalf("content/pages should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("content/pages should be a directory")
	}
}

// ---------------------------------------------------------------------------
// TestNewProject
// ---------------------------------------------------------------------------

func TestNewProject(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("changing to temp directory: %v", err)
	}

	title := "My Cool Project"
	if err := NewProject(title); err != nil {
		t.Fatalf("NewProject(%q): %v", title, err)
	}

	expectedPath := filepath.Join("content", "projects", "my-cool-project.md")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("reading created project: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `title: "My Cool Project"`) {
		t.Errorf("project should contain title, got:\n%s", content)
	}
	if !strings.Contains(content, "draft: true") {
		t.Errorf("project should contain draft: true, got:\n%s", content)
	}
	if !strings.Contains(content, "tech: []") {
		t.Errorf("project should contain tech: [], got:\n%s", content)
	}
	if !strings.Contains(content, `github: ""`) {
		t.Errorf("project should contain github field, got:\n%s", content)
	}
	if !strings.Contains(content, `demo: ""`) {
		t.Errorf("project should contain demo field, got:\n%s", content)
	}
	if !strings.Contains(content, "Describe your project here.") {
		t.Errorf("project should contain placeholder content, got:\n%s", content)
	}
}

func TestNewProject_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("changing to temp directory: %v", err)
	}

	if err := NewProject("Test Project"); err != nil {
		t.Fatalf("NewProject should create parent dirs: %v", err)
	}

	info, err := os.Stat(filepath.Join("content", "projects"))
	if err != nil {
		t.Fatalf("content/projects should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("content/projects should be a directory")
	}
}

// ---------------------------------------------------------------------------
// TestCreatedPaths
// ---------------------------------------------------------------------------

func TestCreatedPaths(t *testing.T) {
	t.Run("post path", func(t *testing.T) {
		got := CreatedPostPath("My First Post")
		want := filepath.Join("content", "blog", "2025-06-15-my-first-post.md")
		if got != want {
			t.Errorf("CreatedPostPath = %q, want %q", got, want)
		}
	})

	t.Run("page path", func(t *testing.T) {
		got := CreatedPagePath("About Me")
		want := filepath.Join("content", "pages", "about-me.md")
		if got != want {
			t.Errorf("CreatedPagePath = %q, want %q", got, want)
		}
	})

	t.Run("project path", func(t *testing.T) {
		got := CreatedProjectPath("My Project")
		want := filepath.Join("content", "projects", "my-project.md")
		if got != want {
			t.Errorf("CreatedProjectPath = %q, want %q", got, want)
		}
	})
}
