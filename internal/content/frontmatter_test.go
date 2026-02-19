package content

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// testdataPath returns the absolute path to a file in the testdata directory.
func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

// mustReadTestdata reads a testdata file and fatals on error.
func mustReadTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(testdataPath(name))
	if err != nil {
		t.Fatalf("failed to read testdata file %q: %v", name, err)
	}
	return data
}

// ---------------------------------------------------------------------------
// Tests: ParseFrontmatter
// ---------------------------------------------------------------------------

func TestParseFrontmatterYAML(t *testing.T) {
	raw := mustReadTestdata(t, "valid_yaml.md")

	metadata, body, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}

	if metadata == nil {
		t.Fatal("ParseFrontmatter() metadata is nil, expected non-nil")
	}

	// Check a few metadata fields.
	if got, ok := metadata["title"].(string); !ok || got != "My First Post" {
		t.Errorf("metadata[\"title\"] = %v, want %q", metadata["title"], "My First Post")
	}
	if got, ok := metadata["draft"].(bool); !ok || got != false {
		t.Errorf("metadata[\"draft\"] = %v, want false", metadata["draft"])
	}
	if got, ok := metadata["weight"].(int); !ok || got != 10 {
		t.Errorf("metadata[\"weight\"] = %v, want 10", metadata["weight"])
	}

	// Verify tags is a slice.
	tags, ok := metadata["tags"].([]any)
	if !ok {
		t.Fatalf("metadata[\"tags\"] is %T, want []any", metadata["tags"])
	}
	if len(tags) != 3 {
		t.Errorf("len(tags) = %d, want 3", len(tags))
	}

	// Verify body separation.
	if body == nil {
		t.Fatal("body is nil")
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "# Hello World") {
		t.Errorf("body does not contain expected content, got: %q", bodyStr)
	}
	if strings.Contains(bodyStr, "---") {
		t.Errorf("body should not contain frontmatter delimiters, got: %q", bodyStr)
	}
}

func TestParseFrontmatterTOML(t *testing.T) {
	raw := mustReadTestdata(t, "valid_toml.md")

	metadata, body, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}

	if metadata == nil {
		t.Fatal("ParseFrontmatter() metadata is nil, expected non-nil")
	}

	// Check a few metadata fields.
	if got, ok := metadata["title"].(string); !ok || got != "TOML Post" {
		t.Errorf("metadata[\"title\"] = %v, want %q", metadata["title"], "TOML Post")
	}
	if got, ok := metadata["draft"].(bool); !ok || got != true {
		t.Errorf("metadata[\"draft\"] = %v, want true", metadata["draft"])
	}

	// TOML tags should be a slice.
	tags, ok := metadata["tags"].([]any)
	if !ok {
		t.Fatalf("metadata[\"tags\"] is %T, want []any", metadata["tags"])
	}
	if len(tags) != 2 {
		t.Errorf("len(tags) = %d, want 2", len(tags))
	}

	// Cover should be a map.
	cover, ok := metadata["cover"].(map[string]any)
	if !ok {
		t.Fatalf("metadata[\"cover\"] is %T, want map[string]any", metadata["cover"])
	}
	if img, ok := cover["image"].(string); !ok || img != "/images/toml-cover.jpg" {
		t.Errorf("cover[\"image\"] = %v, want %q", cover["image"], "/images/toml-cover.jpg")
	}

	// Verify body separation.
	if body == nil {
		t.Fatal("body is nil")
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "# TOML Content") {
		t.Errorf("body does not contain expected content, got: %q", bodyStr)
	}
	if strings.Contains(bodyStr, "+++") {
		t.Errorf("body should not contain frontmatter delimiters, got: %q", bodyStr)
	}
}

func TestParseFrontmatterNone(t *testing.T) {
	raw := mustReadTestdata(t, "no_frontmatter.md")

	metadata, body, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}

	if metadata != nil {
		t.Errorf("metadata = %v, want nil", metadata)
	}

	// Body should be the full content.
	if !strings.Contains(string(body), "# Just Markdown") {
		t.Errorf("body does not contain expected content")
	}
	if len(body) != len(raw) {
		t.Errorf("body length = %d, want %d (full content)", len(body), len(raw))
	}
}

func TestParseFrontmatterEmpty(t *testing.T) {
	raw := mustReadTestdata(t, "empty_frontmatter.md")

	metadata, body, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}

	if metadata == nil {
		t.Fatal("metadata is nil, expected empty map")
	}
	if len(metadata) != 0 {
		t.Errorf("len(metadata) = %d, want 0", len(metadata))
	}

	// Body should contain the content after frontmatter.
	if body == nil {
		t.Fatal("body is nil")
	}
	if !strings.Contains(string(body), "# Empty Frontmatter") {
		t.Errorf("body does not contain expected content, got: %q", string(body))
	}
}

// ---------------------------------------------------------------------------
// Tests: PopulatePage
// ---------------------------------------------------------------------------

func TestPopulatePage(t *testing.T) {
	raw := mustReadTestdata(t, "valid_yaml.md")
	metadata, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}

	page := &Page{}
	if err := PopulatePage(page, metadata); err != nil {
		t.Fatalf("PopulatePage() error = %v", err)
	}

	// Title.
	if page.Title != "My First Post" {
		t.Errorf("Title = %q, want %q", page.Title, "My First Post")
	}

	// Slug.
	if page.Slug != "my-first-post" {
		t.Errorf("Slug = %q, want %q", page.Slug, "my-first-post")
	}

	// Description.
	if page.Description != "A description of my first post" {
		t.Errorf("Description = %q, want %q", page.Description, "A description of my first post")
	}

	// Summary.
	if page.Summary != "This is the summary" {
		t.Errorf("Summary = %q, want %q", page.Summary, "This is the summary")
	}

	// Draft.
	if page.Draft != false {
		t.Errorf("Draft = %v, want false", page.Draft)
	}

	// Weight.
	if page.Weight != 10 {
		t.Errorf("Weight = %d, want 10", page.Weight)
	}

	// Layout.
	if page.Layout != "post" {
		t.Errorf("Layout = %q, want %q", page.Layout, "post")
	}

	// Author.
	if page.Author != "John Doe" {
		t.Errorf("Author = %q, want %q", page.Author, "John Doe")
	}

	// Series.
	if page.Series != "Go Basics" {
		t.Errorf("Series = %q, want %q", page.Series, "Go Basics")
	}

	// Tags.
	wantTags := []string{"go", "programming", "tutorial"}
	if !equalStrings(page.Tags, wantTags) {
		t.Errorf("Tags = %v, want %v", page.Tags, wantTags)
	}

	// Categories.
	wantCategories := []string{"tech", "guides"}
	if !equalStrings(page.Categories, wantCategories) {
		t.Errorf("Categories = %v, want %v", page.Categories, wantCategories)
	}

	// Aliases.
	wantAliases := []string{"/old/path/", "/another/old/path/"}
	if !equalStrings(page.Aliases, wantAliases) {
		t.Errorf("Aliases = %v, want %v", page.Aliases, wantAliases)
	}

	// Cover.
	if page.Cover == nil {
		t.Fatal("Cover is nil")
	}
	if page.Cover.Image != "/images/cover.jpg" {
		t.Errorf("Cover.Image = %q, want %q", page.Cover.Image, "/images/cover.jpg")
	}
	if page.Cover.Alt != "A beautiful cover image" {
		t.Errorf("Cover.Alt = %q, want %q", page.Cover.Alt, "A beautiful cover image")
	}
	if page.Cover.Caption != "Photo by someone" {
		t.Errorf("Cover.Caption = %q, want %q", page.Cover.Caption, "Photo by someone")
	}

	// Params.
	if page.Params == nil {
		t.Fatal("Params is nil")
	}
	if v, ok := page.Params["custom_field"].(string); !ok || v != "custom_value" {
		t.Errorf("Params[\"custom_field\"] = %v, want %q", page.Params["custom_field"], "custom_value")
	}

	// Date (2024-06-15T10:30:00Z).
	wantDate := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	if !page.Date.Equal(wantDate) {
		t.Errorf("Date = %v, want %v", page.Date, wantDate)
	}

	// Lastmod (2024-07-01T08:00:00-05:00).
	wantLastmod := time.Date(2024, 7, 1, 8, 0, 0, 0, time.FixedZone("", -5*3600))
	if !page.Lastmod.Equal(wantLastmod) {
		t.Errorf("Lastmod = %v, want %v", page.Lastmod, wantLastmod)
	}

	// ExpiryDate (2025-12-31).
	wantExpiry := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	if !page.ExpiryDate.Equal(wantExpiry) {
		t.Errorf("ExpiryDate = %v, want %v", page.ExpiryDate, wantExpiry)
	}
}

func TestPopulatePageMissingTitle(t *testing.T) {
	raw := mustReadTestdata(t, "missing_title.md")
	metadata, _, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}

	page := &Page{}
	err = PopulatePage(page, metadata)
	if err == nil {
		t.Fatal("PopulatePage() expected error for missing title, got nil")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Errorf("error message should mention \"title\", got: %v", err)
	}
}

func TestPopulatePageDates(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   time.Time
	}{
		{
			name:  "date only",
			input: "2024-01-15",
			want:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "datetime UTC",
			input: "2024-06-15T10:30:00Z",
			want:  time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:  "datetime with offset",
			input: "2024-07-01T08:00:00-05:00",
			want:  time.Date(2024, 7, 1, 8, 0, 0, 0, time.FixedZone("", -5*3600)),
		},
		{
			name:  "datetime with positive offset",
			input: "2024-03-20T14:00:00+09:00",
			want:  time.Date(2024, 3, 20, 14, 0, 0, 0, time.FixedZone("", 9*3600)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := map[string]any{
				"title": "Test",
				"date":  tt.input,
			}
			page := &Page{}
			if err := PopulatePage(page, metadata); err != nil {
				t.Fatalf("PopulatePage() error = %v", err)
			}
			if !page.Date.Equal(tt.want) {
				t.Errorf("Date = %v, want %v", page.Date, tt.want)
			}
		})
	}

	// Also test time.Time values directly (as TOML parser may produce).
	t.Run("time.Time value", func(t *testing.T) {
		want := time.Date(2024, 8, 10, 12, 0, 0, 0, time.UTC)
		metadata := map[string]any{
			"title": "Test",
			"date":  want,
		}
		page := &Page{}
		if err := PopulatePage(page, metadata); err != nil {
			t.Fatalf("PopulatePage() error = %v", err)
		}
		if !page.Date.Equal(want) {
			t.Errorf("Date = %v, want %v", page.Date, want)
		}
	})
}

func TestPopulatePageCover(t *testing.T) {
	metadata := map[string]any{
		"title": "Cover Test",
		"cover": map[string]any{
			"image":   "/images/hero.png",
			"alt":     "Hero image",
			"caption": "A hero shot",
		},
	}

	page := &Page{}
	if err := PopulatePage(page, metadata); err != nil {
		t.Fatalf("PopulatePage() error = %v", err)
	}

	if page.Cover == nil {
		t.Fatal("Cover is nil")
	}
	if page.Cover.Image != "/images/hero.png" {
		t.Errorf("Cover.Image = %q, want %q", page.Cover.Image, "/images/hero.png")
	}
	if page.Cover.Alt != "Hero image" {
		t.Errorf("Cover.Alt = %q, want %q", page.Cover.Alt, "Hero image")
	}
	if page.Cover.Caption != "A hero shot" {
		t.Errorf("Cover.Caption = %q, want %q", page.Cover.Caption, "A hero shot")
	}

	// Test with partial cover (only image).
	t.Run("partial cover", func(t *testing.T) {
		metadata := map[string]any{
			"title": "Partial Cover",
			"cover": map[string]any{
				"image": "/images/partial.jpg",
			},
		}
		page := &Page{}
		if err := PopulatePage(page, metadata); err != nil {
			t.Fatalf("PopulatePage() error = %v", err)
		}
		if page.Cover == nil {
			t.Fatal("Cover is nil")
		}
		if page.Cover.Image != "/images/partial.jpg" {
			t.Errorf("Cover.Image = %q, want %q", page.Cover.Image, "/images/partial.jpg")
		}
		if page.Cover.Alt != "" {
			t.Errorf("Cover.Alt = %q, want empty", page.Cover.Alt)
		}
	})
}

func TestPopulatePageTags(t *testing.T) {
	// Test with []any (as YAML parser produces).
	t.Run("[]any input", func(t *testing.T) {
		metadata := map[string]any{
			"title": "Tags Test",
			"tags":  []any{"go", "rust", "python"},
		}
		page := &Page{}
		if err := PopulatePage(page, metadata); err != nil {
			t.Fatalf("PopulatePage() error = %v", err)
		}
		want := []string{"go", "rust", "python"}
		if !equalStrings(page.Tags, want) {
			t.Errorf("Tags = %v, want %v", page.Tags, want)
		}
	})

	// Test with []string (direct assignment).
	t.Run("[]string input", func(t *testing.T) {
		metadata := map[string]any{
			"title": "Tags Test",
			"tags":  []string{"alpha", "beta"},
		}
		page := &Page{}
		if err := PopulatePage(page, metadata); err != nil {
			t.Fatalf("PopulatePage() error = %v", err)
		}
		want := []string{"alpha", "beta"}
		if !equalStrings(page.Tags, want) {
			t.Errorf("Tags = %v, want %v", page.Tags, want)
		}
	})

	// Test with empty slice.
	t.Run("empty slice", func(t *testing.T) {
		metadata := map[string]any{
			"title": "Tags Test",
			"tags":  []any{},
		}
		page := &Page{}
		if err := PopulatePage(page, metadata); err != nil {
			t.Fatalf("PopulatePage() error = %v", err)
		}
		if len(page.Tags) != 0 {
			t.Errorf("Tags = %v, want empty", page.Tags)
		}
	})
}
