package content

import (
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/aellingwood/forge/internal/config"
)

// testdataDir returns the absolute path to the testdata/site fixture directory.
func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "site")
}

// findPageByURL finds a page with the given URL in the pages slice.
// Returns nil if not found.
func findPageByURL(pages []*Page, url string) *Page {
	for _, p := range pages {
		if p.URL == url {
			return p
		}
	}
	return nil
}

// findPageByTitle finds a page with the given title in the pages slice.
// Returns nil if not found.
func findPageByTitle(pages []*Page, title string) *Page {
	for _, p := range pages {
		if p.Title == title {
			return p
		}
	}
	return nil
}

func TestDiscover(t *testing.T) {
	contentDir := testdataDir(t)
	cfg := config.Default()

	pages, err := Discover(contentDir, cfg)
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	// Verify total page count: 8 pages
	// _index.md, about.md, blog/_index.md, blog/first-post.md,
	// blog/2025-01-15-second-post.md, blog/bundled-post/index.md,
	// projects/_index.md, projects/my-project.md
	if len(pages) != 8 {
		t.Errorf("Discover() returned %d pages, want 8", len(pages))
		for _, p := range pages {
			t.Logf("  page: %q URL=%q Type=%s", p.Title, p.URL, p.Type)
		}
	}

	// Verify homepage
	home := findPageByURL(pages, "/")
	if home == nil {
		t.Fatal("homepage with URL \"/\" not found")
	}
	if home.Type != PageTypeHome {
		t.Errorf("homepage Type = %v, want PageTypeHome", home.Type)
	}
	if home.Title != "Home" {
		t.Errorf("homepage Title = %q, want %q", home.Title, "Home")
	}

	// Verify blog list page
	blogList := findPageByURL(pages, "/blog/")
	if blogList == nil {
		t.Fatal("blog list page with URL \"/blog/\" not found")
	}
	if blogList.Type != PageTypeList {
		t.Errorf("blog list Type = %v, want PageTypeList", blogList.Type)
	}
	if blogList.Title != "Blog" {
		t.Errorf("blog list Title = %q, want %q", blogList.Title, "Blog")
	}
	if blogList.Section != "blog" {
		t.Errorf("blog list Section = %q, want %q", blogList.Section, "blog")
	}

	// Verify single post URLs
	firstPost := findPageByURL(pages, "/blog/first-post/")
	if firstPost == nil {
		t.Fatal("first post with URL \"/blog/first-post/\" not found")
	}
	if firstPost.Type != PageTypeSingle {
		t.Errorf("first post Type = %v, want PageTypeSingle", firstPost.Type)
	}
	if firstPost.Title != "First Post" {
		t.Errorf("first post Title = %q, want %q", firstPost.Title, "First Post")
	}
	if len(firstPost.Tags) != 2 || firstPost.Tags[0] != "go" || firstPost.Tags[1] != "testing" {
		t.Errorf("first post Tags = %v, want [go testing]", firstPost.Tags)
	}

	// Verify about page (root single page, no section)
	about := findPageByURL(pages, "/about/")
	if about == nil {
		t.Fatal("about page with URL \"/about/\" not found")
	}
	if about.Section != "" {
		t.Errorf("about page Section = %q, want empty string", about.Section)
	}

	// Verify projects section
	projectsList := findPageByURL(pages, "/projects/")
	if projectsList == nil {
		t.Fatal("projects list page with URL \"/projects/\" not found")
	}
	if projectsList.Type != PageTypeList {
		t.Errorf("projects list Type = %v, want PageTypeList", projectsList.Type)
	}

	myProject := findPageByURL(pages, "/projects/my-project/")
	if myProject == nil {
		t.Fatal("my-project page with URL \"/projects/my-project/\" not found")
	}
	if myProject.Section != "projects" {
		t.Errorf("my-project Section = %q, want %q", myProject.Section, "projects")
	}
}

func TestDiscoverPageBundle(t *testing.T) {
	contentDir := testdataDir(t)
	cfg := config.Default()

	pages, err := Discover(contentDir, cfg)
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	bundled := findPageByURL(pages, "/blog/bundled-post/")
	if bundled == nil {
		t.Fatal("bundled post with URL \"/blog/bundled-post/\" not found")
	}

	if !bundled.IsBundle {
		t.Error("bundled post IsBundle = false, want true")
	}

	if bundled.Type != PageTypeSingle {
		t.Errorf("bundled post Type = %v, want PageTypeSingle", bundled.Type)
	}

	// Verify BundleFiles contains diagram.png
	found := slices.Contains(bundled.BundleFiles, "diagram.png")
	if !found {
		t.Errorf("bundled post BundleFiles = %v, want to contain \"diagram.png\"", bundled.BundleFiles)
	}
}

func TestDiscoverDatePrefix(t *testing.T) {
	contentDir := testdataDir(t)
	cfg := config.Default()

	pages, err := Discover(contentDir, cfg)
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	secondPost := findPageByTitle(pages, "Second Post")
	if secondPost == nil {
		t.Fatal("second post not found by title")
	}

	// Slug should have the date prefix stripped.
	if secondPost.Slug != "second-post" {
		t.Errorf("second post Slug = %q, want %q", secondPost.Slug, "second-post")
	}

	// Draft should be true.
	if !secondPost.Draft {
		t.Error("second post Draft = false, want true")
	}

	// URL should use the slug without date prefix.
	if secondPost.URL != "/blog/second-post/" {
		t.Errorf("second post URL = %q, want %q", secondPost.URL, "/blog/second-post/")
	}
}

func TestDiscoverReadingTime(t *testing.T) {
	contentDir := testdataDir(t)
	cfg := config.Default()

	pages, err := Discover(contentDir, cfg)
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	firstPost := findPageByURL(pages, "/blog/first-post/")
	if firstPost == nil {
		t.Fatal("first post not found")
	}

	if firstPost.WordCount == 0 {
		t.Error("first post WordCount = 0, want > 0")
	}

	if firstPost.ReadingTime == 0 {
		t.Error("first post ReadingTime = 0, want >= 1")
	}

	// With fewer than 200 words, reading time should be 1 (minimum).
	if firstPost.ReadingTime != 1 {
		t.Errorf("first post ReadingTime = %d, want 1 (fewer than 200 words)", firstPost.ReadingTime)
	}

	// Verify about page also has word count
	about := findPageByURL(pages, "/about/")
	if about == nil {
		t.Fatal("about page not found")
	}
	if about.WordCount == 0 {
		t.Error("about page WordCount = 0, want > 0")
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"My_Post_Title", "my-post-title"},
		{"UPPERCASE", "uppercase"},
		{"  spaces  ", "spaces"},
		{"special!@#$%chars", "specialchars"},
		{"multiple---hyphens", "multiple-hyphens"},
		{"file.name.ext", "file.name.ext"},
		{"---leading-trailing---", "leading-trailing"},
		{"Hello World!", "hello-world"},
		{"café", "caf"},
		{"a---b___c   d", "a-b-c-d"},
		{"", ""},
	}

	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
