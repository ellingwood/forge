package seo

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func TestGenerateSitemap(t *testing.T) {
	t.Run("entries with and without lastmod", func(t *testing.T) {
		entries := []SitemapEntry{
			{
				URL:     "https://example.com/",
				Lastmod: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			},
			{
				URL:     "https://example.com/about/",
				Lastmod: time.Time{}, // zero time, no lastmod
			},
			{
				URL:     "https://example.com/blog/post-1/",
				Lastmod: time.Date(2025, 12, 1, 10, 30, 0, 0, time.UTC),
			},
		}

		data, err := GenerateSitemap(entries)
		if err != nil {
			t.Fatalf("GenerateSitemap returned error: %v", err)
		}

		result := string(data)

		// Check XML declaration
		if !strings.HasPrefix(result, `<?xml version="1.0" encoding="UTF-8"?>`) {
			t.Error("sitemap should start with XML declaration")
		}

		// Check xmlns
		if !strings.Contains(result, `xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`) {
			t.Error("sitemap should contain sitemaps.org xmlns")
		}

		// Check URLs are present
		if !strings.Contains(result, "<loc>https://example.com/</loc>") {
			t.Error("sitemap should contain the root URL")
		}
		if !strings.Contains(result, "<loc>https://example.com/about/</loc>") {
			t.Error("sitemap should contain the about URL")
		}
		if !strings.Contains(result, "<loc>https://example.com/blog/post-1/</loc>") {
			t.Error("sitemap should contain the blog post URL")
		}

		// Check lastmod for first entry (has date)
		if !strings.Contains(result, "<lastmod>2025-06-15</lastmod>") {
			t.Error("sitemap should contain lastmod for first entry")
		}

		// Check lastmod for third entry
		if !strings.Contains(result, "<lastmod>2025-12-01</lastmod>") {
			t.Error("sitemap should contain lastmod for third entry")
		}

		// Verify the about page does NOT have a lastmod element
		// Parse it as XML to be sure
		type urlEntry struct {
			Loc     string `xml:"loc"`
			Lastmod string `xml:"lastmod"`
		}
		type urlSet struct {
			URLs []urlEntry `xml:"url"`
		}
		var parsed urlSet
		if err := xml.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("failed to parse sitemap XML: %v", err)
		}
		if len(parsed.URLs) != 3 {
			t.Fatalf("expected 3 URLs, got %d", len(parsed.URLs))
		}
		if parsed.URLs[1].Lastmod != "" {
			t.Errorf("about page should not have lastmod, got %q", parsed.URLs[1].Lastmod)
		}
	})

	t.Run("empty entries", func(t *testing.T) {
		data, err := GenerateSitemap([]SitemapEntry{})
		if err != nil {
			t.Fatalf("GenerateSitemap returned error: %v", err)
		}

		result := string(data)
		if !strings.Contains(result, "<urlset") {
			t.Error("empty sitemap should still contain urlset element")
		}
		if strings.Contains(result, "<url>") {
			t.Error("empty sitemap should not contain any url elements")
		}
	})

	t.Run("nil entries", func(t *testing.T) {
		data, err := GenerateSitemap(nil)
		if err != nil {
			t.Fatalf("GenerateSitemap returned error: %v", err)
		}

		result := string(data)
		if !strings.Contains(result, "<urlset") {
			t.Error("nil sitemap should still contain urlset element")
		}
	})
}

func TestGenerateRobotsTxt(t *testing.T) {
	result := string(GenerateRobotsTxt("https://example.com/sitemap.xml"))

	expected := "User-agent: *\nAllow: /\n\nSitemap: https://example.com/sitemap.xml\n"
	if result != expected {
		t.Errorf("robots.txt mismatch\ngot:\n%s\nwant:\n%s", result, expected)
	}
}

func TestOpenGraphMeta(t *testing.T) {
	t.Run("with cover image and language", func(t *testing.T) {
		meta := PageMeta{
			Title:       "My Blog Post",
			Description: "A great blog post",
			URL:         "https://example.com/blog/my-post/",
			PageType:    "article",
			SiteName:    "My Site",
			CoverImage:  "https://example.com/images/cover.jpg",
			Language:    "en_US",
		}

		result := OpenGraphMeta(meta)

		expectations := []string{
			`<meta property="og:title" content="My Blog Post">`,
			`<meta property="og:description" content="A great blog post">`,
			`<meta property="og:url" content="https://example.com/blog/my-post/">`,
			`<meta property="og:type" content="article">`,
			`<meta property="og:site_name" content="My Site">`,
			`<meta property="og:image" content="https://example.com/images/cover.jpg">`,
			`<meta property="og:locale" content="en_US">`,
		}

		for _, exp := range expectations {
			if !strings.Contains(result, exp) {
				t.Errorf("expected og tag not found: %s", exp)
			}
		}
	})

	t.Run("without cover image or language", func(t *testing.T) {
		meta := PageMeta{
			Title:       "Simple Page",
			Description: "A simple page",
			URL:         "https://example.com/simple/",
			PageType:    "website",
			SiteName:    "My Site",
		}

		result := OpenGraphMeta(meta)

		if strings.Contains(result, "og:image") {
			t.Error("should not contain og:image when no cover image")
		}
		if strings.Contains(result, "og:locale") {
			t.Error("should not contain og:locale when no language")
		}

		// Core tags should still be present
		if !strings.Contains(result, `og:title`) {
			t.Error("should contain og:title")
		}
		if !strings.Contains(result, `og:type`) {
			t.Error("should contain og:type")
		}
	})

	t.Run("html escaping in content", func(t *testing.T) {
		meta := PageMeta{
			Title:       `Title with "quotes" & <brackets>`,
			Description: "Normal description",
			URL:         "https://example.com/",
			PageType:    "website",
			SiteName:    "Test",
		}

		result := OpenGraphMeta(meta)

		if strings.Contains(result, `"quotes"`) {
			t.Error("quotes in content should be escaped")
		}
		if !strings.Contains(result, `&amp;`) {
			t.Error("ampersand should be escaped")
		}
	})
}

func TestTwitterCardMeta(t *testing.T) {
	t.Run("with cover image", func(t *testing.T) {
		meta := PageMeta{
			Title:       "My Post",
			Description: "A description",
			CoverImage:  "https://example.com/cover.jpg",
		}

		result := TwitterCardMeta(meta)

		if !strings.Contains(result, `content="summary_large_image"`) {
			t.Error("should use summary_large_image when cover is present")
		}
		if !strings.Contains(result, `<meta name="twitter:image"`) {
			t.Error("should include twitter:image when cover is present")
		}
		if !strings.Contains(result, `<meta name="twitter:title"`) {
			t.Error("should include twitter:title")
		}
		if !strings.Contains(result, `<meta name="twitter:description"`) {
			t.Error("should include twitter:description")
		}
	})

	t.Run("without cover image", func(t *testing.T) {
		meta := PageMeta{
			Title:       "My Post",
			Description: "A description",
		}

		result := TwitterCardMeta(meta)

		if !strings.Contains(result, `content="summary"`) {
			t.Error("should use summary card when no cover image")
		}
		if strings.Contains(result, "twitter:image") {
			t.Error("should not include twitter:image when no cover image")
		}
	})
}

func TestJSONLDArticle(t *testing.T) {
	t.Run("with all fields", func(t *testing.T) {
		date := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
		meta := PageMeta{
			Title:       "My Article",
			Description: "Article description",
			URL:         "https://example.com/blog/my-article/",
			Author:      "Jane Doe",
			Date:        date,
			CoverImage:  "https://example.com/cover.jpg",
		}

		result := JSONLDArticle(meta)

		if !strings.HasPrefix(result, `<script type="application/ld+json">`) {
			t.Error("should start with script tag")
		}
		if !strings.HasSuffix(result, `</script>`) {
			t.Error("should end with closing script tag")
		}

		// Extract JSON content
		jsonStr := strings.TrimPrefix(result, `<script type="application/ld+json">`)
		jsonStr = strings.TrimSuffix(jsonStr, `</script>`)

		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			t.Fatalf("failed to parse JSON-LD: %v", err)
		}

		if data["@context"] != "https://schema.org" {
			t.Errorf("@context = %v, want https://schema.org", data["@context"])
		}
		if data["@type"] != "Article" {
			t.Errorf("@type = %v, want Article", data["@type"])
		}
		if data["headline"] != "My Article" {
			t.Errorf("headline = %v, want My Article", data["headline"])
		}
		if data["description"] != "Article description" {
			t.Errorf("description = %v, want Article description", data["description"])
		}
		if data["url"] != "https://example.com/blog/my-article/" {
			t.Errorf("url = %v, want https://example.com/blog/my-article/", data["url"])
		}
		if data["image"] != "https://example.com/cover.jpg" {
			t.Errorf("image = %v, want https://example.com/cover.jpg", data["image"])
		}

		// Check datePublished is valid RFC3339
		dateStr, ok := data["datePublished"].(string)
		if !ok {
			t.Fatal("datePublished should be a string")
		}
		if _, err := time.Parse(time.RFC3339, dateStr); err != nil {
			t.Errorf("datePublished is not valid RFC3339: %v", err)
		}

		// Check author
		author, ok := data["author"].(map[string]any)
		if !ok {
			t.Fatal("author should be an object")
		}
		if author["@type"] != "Person" {
			t.Errorf("author @type = %v, want Person", author["@type"])
		}
		if author["name"] != "Jane Doe" {
			t.Errorf("author name = %v, want Jane Doe", author["name"])
		}
	})

	t.Run("without cover image", func(t *testing.T) {
		meta := PageMeta{
			Title:       "No Cover Article",
			Description: "Desc",
			URL:         "https://example.com/",
			Author:      "Author",
			Date:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		result := JSONLDArticle(meta)

		jsonStr := strings.TrimPrefix(result, `<script type="application/ld+json">`)
		jsonStr = strings.TrimSuffix(jsonStr, `</script>`)

		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			t.Fatalf("failed to parse JSON-LD: %v", err)
		}

		if _, exists := data["image"]; exists {
			t.Error("image should not be present when no cover image")
		}
	})

	t.Run("without author", func(t *testing.T) {
		meta := PageMeta{
			Title:       "No Author Article",
			Description: "Desc",
			URL:         "https://example.com/",
			Date:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		result := JSONLDArticle(meta)

		jsonStr := strings.TrimPrefix(result, `<script type="application/ld+json">`)
		jsonStr = strings.TrimSuffix(jsonStr, `</script>`)

		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			t.Fatalf("failed to parse JSON-LD: %v", err)
		}

		if _, exists := data["author"]; exists {
			t.Error("author should not be present when no author provided")
		}
	})

	t.Run("special characters in fields", func(t *testing.T) {
		meta := PageMeta{
			Title:       `Title with "quotes" & <script>alert('xss')</script>`,
			Description: "Safe description",
			URL:         "https://example.com/",
			Author:      "O'Brien",
			Date:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		result := JSONLDArticle(meta)

		jsonStr := strings.TrimPrefix(result, `<script type="application/ld+json">`)
		jsonStr = strings.TrimSuffix(jsonStr, `</script>`)

		// The JSON should be valid and properly escaped
		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			t.Fatalf("JSON-LD with special characters should be valid JSON: %v", err)
		}

		if data["headline"] != `Title with "quotes" & <script>alert('xss')</script>` {
			t.Errorf("headline should preserve special characters after JSON decode, got: %v", data["headline"])
		}
	})
}

func TestCanonicalURL(t *testing.T) {
	result := CanonicalURL("https://example.com/blog/my-post/")
	expected := `<link rel="canonical" href="https://example.com/blog/my-post/">`
	if result != expected {
		t.Errorf("CanonicalURL = %q, want %q", result, expected)
	}
}

func TestSEOTitle(t *testing.T) {
	t.Run("with template containing %s", func(t *testing.T) {
		result := SEOTitle("My Post", "%s | My Site")
		if result != "My Post | My Site" {
			t.Errorf("SEOTitle = %q, want %q", result, "My Post | My Site")
		}
	})

	t.Run("without template", func(t *testing.T) {
		result := SEOTitle("My Post", "")
		if result != "My Post" {
			t.Errorf("SEOTitle = %q, want %q", result, "My Post")
		}
	})

	t.Run("template without %s", func(t *testing.T) {
		result := SEOTitle("My Post", "Static Title")
		if result != "My Post" {
			t.Errorf("SEOTitle = %q, want %q", result, "My Post")
		}
	})

	t.Run("empty page title with template", func(t *testing.T) {
		result := SEOTitle("", "%s | My Site")
		if result != " | My Site" {
			t.Errorf("SEOTitle = %q, want %q", result, " | My Site")
		}
	})

	t.Run("empty page title without template", func(t *testing.T) {
		result := SEOTitle("", "")
		if result != "" {
			t.Errorf("SEOTitle = %q, want %q", result, "")
		}
	})
}
