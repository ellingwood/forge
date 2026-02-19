package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

// testdataPath returns the absolute path to a file inside the testdata
// directory, relative to this test file's location on disk.
func testdataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

// ---------------------------------------------------------------------------
// TestDefault
// ---------------------------------------------------------------------------

func TestDefault(t *testing.T) {
	cfg := Default()

	// Top-level defaults
	if cfg.Language != "en" {
		t.Errorf("Language: got %q, want %q", cfg.Language, "en")
	}
	if cfg.Theme != "default" {
		t.Errorf("Theme: got %q, want %q", cfg.Theme, "default")
	}

	// Server defaults
	if cfg.Server.Port != 1313 {
		t.Errorf("Server.Port: got %d, want %d", cfg.Server.Port, 1313)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("Server.Host: got %q, want %q", cfg.Server.Host, "localhost")
	}
	if !cfg.Server.LiveReload {
		t.Error("Server.LiveReload: got false, want true")
	}

	// Pagination
	if cfg.Pagination.PageSize != 10 {
		t.Errorf("Pagination.PageSize: got %d, want %d", cfg.Pagination.PageSize, 10)
	}

	// Highlight
	if cfg.Highlight.Style != "github" {
		t.Errorf("Highlight.Style: got %q, want %q", cfg.Highlight.Style, "github")
	}
	if cfg.Highlight.DarkStyle != "github-dark" {
		t.Errorf("Highlight.DarkStyle: got %q, want %q", cfg.Highlight.DarkStyle, "github-dark")
	}
	if cfg.Highlight.TabWidth != 4 {
		t.Errorf("Highlight.TabWidth: got %d, want %d", cfg.Highlight.TabWidth, 4)
	}

	// Search
	if !cfg.Search.Enabled {
		t.Error("Search.Enabled: got false, want true")
	}
	if cfg.Search.ContentLength != 5000 {
		t.Errorf("Search.ContentLength: got %d, want %d", cfg.Search.ContentLength, 5000)
	}
	if len(cfg.Search.Keys) != 4 {
		t.Errorf("Search.Keys length: got %d, want %d", len(cfg.Search.Keys), 4)
	}

	// Feeds
	if !cfg.Feeds.RSS {
		t.Error("Feeds.RSS: got false, want true")
	}
	if !cfg.Feeds.Atom {
		t.Error("Feeds.Atom: got false, want true")
	}
	if cfg.Feeds.Limit != 20 {
		t.Errorf("Feeds.Limit: got %d, want %d", cfg.Feeds.Limit, 20)
	}

	// SEO
	if !cfg.SEO.JSONLD {
		t.Error("SEO.JSONLD: got false, want true")
	}

	// Taxonomies
	if cfg.Taxonomies["tag"] != "tags" {
		t.Errorf("Taxonomies[tag]: got %q, want %q", cfg.Taxonomies["tag"], "tags")
	}
	if cfg.Taxonomies["category"] != "categories" {
		t.Errorf("Taxonomies[category]: got %q, want %q", cfg.Taxonomies["category"], "categories")
	}
}

// ---------------------------------------------------------------------------
// TestLoadMinimal
// ---------------------------------------------------------------------------

func TestLoadMinimal(t *testing.T) {
	cfg, err := Load(testdataPath("config/minimal.yaml"))
	if err != nil {
		t.Fatalf("Load minimal config: %v", err)
	}

	// Explicit values from the file
	if cfg.Title != "Test Site" {
		t.Errorf("Title: got %q, want %q", cfg.Title, "Test Site")
	}
	if cfg.BaseURL != "https://test.com" {
		t.Errorf("BaseURL: got %q, want %q", cfg.BaseURL, "https://test.com")
	}

	// Defaults should still be filled in
	if cfg.Language != "en" {
		t.Errorf("Language: got %q, want %q", cfg.Language, "en")
	}
	if cfg.Server.Port != 1313 {
		t.Errorf("Server.Port: got %d, want %d", cfg.Server.Port, 1313)
	}
	if cfg.Pagination.PageSize != 10 {
		t.Errorf("Pagination.PageSize: got %d, want %d", cfg.Pagination.PageSize, 10)
	}
	if cfg.Highlight.Style != "github" {
		t.Errorf("Highlight.Style: got %q, want %q", cfg.Highlight.Style, "github")
	}
	if !cfg.Search.Enabled {
		t.Error("Search.Enabled: got false, want true")
	}
	if !cfg.Feeds.RSS {
		t.Error("Feeds.RSS: got false, want true")
	}
}

// ---------------------------------------------------------------------------
// TestLoadFull
// ---------------------------------------------------------------------------

func TestLoadFull(t *testing.T) {
	cfg, err := Load(testdataPath("config/full.yaml"))
	if err != nil {
		t.Fatalf("Load full config: %v", err)
	}

	// Top-level fields
	if cfg.BaseURL != "https://example.com" {
		t.Errorf("BaseURL: got %q, want %q", cfg.BaseURL, "https://example.com")
	}
	if cfg.Title != "My Site" {
		t.Errorf("Title: got %q, want %q", cfg.Title, "My Site")
	}
	if cfg.Description != "Personal portfolio and blog" {
		t.Errorf("Description: got %q, want %q", cfg.Description, "Personal portfolio and blog")
	}
	if cfg.Language != "en" {
		t.Errorf("Language: got %q, want %q", cfg.Language, "en")
	}
	if cfg.Theme != "default" {
		t.Errorf("Theme: got %q, want %q", cfg.Theme, "default")
	}

	// Author
	if cfg.Author.Name != "Austin" {
		t.Errorf("Author.Name: got %q, want %q", cfg.Author.Name, "Austin")
	}
	if cfg.Author.Email != "austin@example.com" {
		t.Errorf("Author.Email: got %q, want %q", cfg.Author.Email, "austin@example.com")
	}
	if cfg.Author.Bio != "Cloud engineer." {
		t.Errorf("Author.Bio: got %q, want %q", cfg.Author.Bio, "Cloud engineer.")
	}
	if cfg.Author.Avatar != "/images/avatar.jpg" {
		t.Errorf("Author.Avatar: got %q, want %q", cfg.Author.Avatar, "/images/avatar.jpg")
	}
	if cfg.Author.Social.GitHub != "username" {
		t.Errorf("Author.Social.GitHub: got %q, want %q", cfg.Author.Social.GitHub, "username")
	}
	if cfg.Author.Social.LinkedIn != "username" {
		t.Errorf("Author.Social.LinkedIn: got %q, want %q", cfg.Author.Social.LinkedIn, "username")
	}
	if cfg.Author.Social.Twitter != "username" {
		t.Errorf("Author.Social.Twitter: got %q, want %q", cfg.Author.Social.Twitter, "username")
	}

	// Menu
	if len(cfg.Menu.Main) != 2 {
		t.Fatalf("Menu.Main length: got %d, want %d", len(cfg.Menu.Main), 2)
	}
	if cfg.Menu.Main[0].Name != "Home" {
		t.Errorf("Menu.Main[0].Name: got %q, want %q", cfg.Menu.Main[0].Name, "Home")
	}
	if cfg.Menu.Main[0].URL != "/" {
		t.Errorf("Menu.Main[0].URL: got %q, want %q", cfg.Menu.Main[0].URL, "/")
	}
	if cfg.Menu.Main[0].Weight != 1 {
		t.Errorf("Menu.Main[0].Weight: got %d, want %d", cfg.Menu.Main[0].Weight, 1)
	}
	if cfg.Menu.Main[1].Name != "Blog" {
		t.Errorf("Menu.Main[1].Name: got %q, want %q", cfg.Menu.Main[1].Name, "Blog")
	}
	if cfg.Menu.Main[1].URL != "/blog/" {
		t.Errorf("Menu.Main[1].URL: got %q, want %q", cfg.Menu.Main[1].URL, "/blog/")
	}
	if cfg.Menu.Main[1].Weight != 2 {
		t.Errorf("Menu.Main[1].Weight: got %d, want %d", cfg.Menu.Main[1].Weight, 2)
	}

	// Pagination
	if cfg.Pagination.PageSize != 10 {
		t.Errorf("Pagination.PageSize: got %d, want %d", cfg.Pagination.PageSize, 10)
	}

	// Taxonomies
	if cfg.Taxonomies["tag"] != "tags" {
		t.Errorf("Taxonomies[tag]: got %q, want %q", cfg.Taxonomies["tag"], "tags")
	}
	if cfg.Taxonomies["category"] != "categories" {
		t.Errorf("Taxonomies[category]: got %q, want %q", cfg.Taxonomies["category"], "categories")
	}

	// Highlight
	if cfg.Highlight.Style != "github" {
		t.Errorf("Highlight.Style: got %q, want %q", cfg.Highlight.Style, "github")
	}
	if cfg.Highlight.DarkStyle != "github-dark" {
		t.Errorf("Highlight.DarkStyle: got %q, want %q", cfg.Highlight.DarkStyle, "github-dark")
	}
	if cfg.Highlight.LineNumbers != false {
		t.Error("Highlight.LineNumbers: got true, want false")
	}
	if cfg.Highlight.TabWidth != 4 {
		t.Errorf("Highlight.TabWidth: got %d, want %d", cfg.Highlight.TabWidth, 4)
	}

	// Search
	if !cfg.Search.Enabled {
		t.Error("Search.Enabled: got false, want true")
	}
	if cfg.Search.ContentLength != 5000 {
		t.Errorf("Search.ContentLength: got %d, want %d", cfg.Search.ContentLength, 5000)
	}
	if len(cfg.Search.Keys) != 4 {
		t.Fatalf("Search.Keys length: got %d, want %d", len(cfg.Search.Keys), 4)
	}
	if cfg.Search.Keys[0].Name != "title" || cfg.Search.Keys[0].Weight != 2.0 {
		t.Errorf("Search.Keys[0]: got {%q, %f}, want {%q, %f}",
			cfg.Search.Keys[0].Name, cfg.Search.Keys[0].Weight, "title", 2.0)
	}
	if cfg.Search.Keys[3].Name != "content" || cfg.Search.Keys[3].Weight != 0.5 {
		t.Errorf("Search.Keys[3]: got {%q, %f}, want {%q, %f}",
			cfg.Search.Keys[3].Name, cfg.Search.Keys[3].Weight, "content", 0.5)
	}

	// Feeds
	if !cfg.Feeds.RSS {
		t.Error("Feeds.RSS: got false, want true")
	}
	if !cfg.Feeds.Atom {
		t.Error("Feeds.Atom: got false, want true")
	}
	if cfg.Feeds.Limit != 20 {
		t.Errorf("Feeds.Limit: got %d, want %d", cfg.Feeds.Limit, 20)
	}
	if !cfg.Feeds.FullContent {
		t.Error("Feeds.FullContent: got false, want true")
	}
	if len(cfg.Feeds.Sections) != 1 || cfg.Feeds.Sections[0] != "blog" {
		t.Errorf("Feeds.Sections: got %v, want [blog]", cfg.Feeds.Sections)
	}

	// SEO
	if cfg.SEO.TitleTemplate != "%s | My Site" {
		t.Errorf("SEO.TitleTemplate: got %q, want %q", cfg.SEO.TitleTemplate, "%s | My Site")
	}
	if cfg.SEO.DefaultImage != "/images/og-default.jpg" {
		t.Errorf("SEO.DefaultImage: got %q, want %q", cfg.SEO.DefaultImage, "/images/og-default.jpg")
	}
	if cfg.SEO.TwitterHandle != "@myhandle" {
		t.Errorf("SEO.TwitterHandle: got %q, want %q", cfg.SEO.TwitterHandle, "@myhandle")
	}
	if !cfg.SEO.JSONLD {
		t.Error("SEO.JSONLD: got false, want true")
	}

	// Server
	if cfg.Server.Port != 1313 {
		t.Errorf("Server.Port: got %d, want %d", cfg.Server.Port, 1313)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("Server.Host: got %q, want %q", cfg.Server.Host, "localhost")
	}
	if !cfg.Server.LiveReload {
		t.Error("Server.LiveReload: got false, want true")
	}

	// Build
	if !cfg.Build.Minify {
		t.Error("Build.Minify: got false, want true")
	}
	if !cfg.Build.CleanURLs {
		t.Error("Build.CleanURLs: got false, want true")
	}

	// Deploy
	if cfg.Deploy.S3.Bucket != "my-site-bucket" {
		t.Errorf("Deploy.S3.Bucket: got %q, want %q", cfg.Deploy.S3.Bucket, "my-site-bucket")
	}
	if cfg.Deploy.S3.Region != "us-west-2" {
		t.Errorf("Deploy.S3.Region: got %q, want %q", cfg.Deploy.S3.Region, "us-west-2")
	}
	if cfg.Deploy.CloudFront.DistributionID != "E1234567890" {
		t.Errorf("Deploy.CloudFront.DistributionID: got %q, want %q",
			cfg.Deploy.CloudFront.DistributionID, "E1234567890")
	}
	if len(cfg.Deploy.CloudFront.InvalidatePaths) != 1 ||
		cfg.Deploy.CloudFront.InvalidatePaths[0] != "/*" {
		t.Errorf("Deploy.CloudFront.InvalidatePaths: got %v, want [/*]",
			cfg.Deploy.CloudFront.InvalidatePaths)
	}

	// Params
	if cfg.Params == nil {
		t.Fatal("Params: got nil, want map")
	}
	if math, ok := cfg.Params["math"]; !ok {
		t.Error("Params[math]: key missing")
	} else if math != false {
		t.Errorf("Params[math]: got %v, want false", math)
	}
}

// ---------------------------------------------------------------------------
// TestValidate
// ---------------------------------------------------------------------------

func TestValidate(t *testing.T) {
	t.Run("missing title", func(t *testing.T) {
		cfg := Default()
		cfg.Title = ""
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for missing title, got nil")
		}
	})

	t.Run("whitespace-only title", func(t *testing.T) {
		cfg := Default()
		cfg.Title = "   "
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for whitespace-only title, got nil")
		}
	})

	t.Run("trailing slash on baseURL", func(t *testing.T) {
		cfg := Default()
		cfg.Title = "Test"
		cfg.BaseURL = "https://example.com/"
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for trailing slash, got nil")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		cfg := Default()
		cfg.Title = "Test"
		cfg.BaseURL = "https://example.com"
		if err := cfg.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("valid config without baseURL", func(t *testing.T) {
		cfg := Default()
		cfg.Title = "Test"
		if err := cfg.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestWithOverrides
// ---------------------------------------------------------------------------

func TestWithOverrides(t *testing.T) {
	cfg := Default()
	cfg.Title = "Original"
	cfg.BaseURL = "https://original.com"

	result := cfg.WithOverrides(map[string]any{
		"baseURL": "https://override.com",
		"title":   "Overridden",
		"theme":   "custom",
		"port":    8080,
		"host":    "0.0.0.0",
		"minify":  true,
	})

	// WithOverrides returns the same pointer
	if result != cfg {
		t.Error("WithOverrides should return the same config pointer")
	}

	if cfg.BaseURL != "https://override.com" {
		t.Errorf("BaseURL: got %q, want %q", cfg.BaseURL, "https://override.com")
	}
	if cfg.Title != "Overridden" {
		t.Errorf("Title: got %q, want %q", cfg.Title, "Overridden")
	}
	if cfg.Theme != "custom" {
		t.Errorf("Theme: got %q, want %q", cfg.Theme, "custom")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port: got %d, want %d", cfg.Server.Port, 8080)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host: got %q, want %q", cfg.Server.Host, "0.0.0.0")
	}
	if !cfg.Build.Minify {
		t.Error("Build.Minify: got false, want true")
	}

	// Language should remain the default since it was not overridden.
	if cfg.Language != "en" {
		t.Errorf("Language: got %q, want %q (should not have changed)", cfg.Language, "en")
	}
}
