package feed

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func defaultOpts() FeedOptions {
	return FeedOptions{
		Title:       "My Site",
		Description: "A test site",
		Link:        "https://example.com",
		FeedLink:    "https://example.com/index.xml",
		Language:    "en",
		Author:      "Jane Doe",
	}
}

func sampleItems() []FeedItem {
	return []FeedItem{
		{
			Title:       "First Post",
			Link:        "https://example.com/blog/first/",
			Description: "Summary of first post",
			Content:     "<p>Full content of first post</p>",
			Author:      "Jane Doe",
			PubDate:     time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			GUID:        "https://example.com/blog/first/",
			Categories:  []string{"go", "programming"},
		},
		{
			Title:       "Second Post",
			Link:        "https://example.com/blog/second/",
			Description: "Summary of second post",
			Content:     "<p>Full content of second post</p>",
			Author:      "John Smith",
			PubDate:     time.Date(2025, 2, 20, 14, 30, 0, 0, time.UTC),
			GUID:        "https://example.com/blog/second/",
			Categories:  []string{"rust"},
		},
		{
			Title:       "Third Post",
			Link:        "https://example.com/blog/third/",
			Description: "Summary of third post",
			Content:     "<p>Full content of third post</p>",
			Author:      "Jane Doe",
			PubDate:     time.Date(2025, 3, 10, 8, 0, 0, 0, time.UTC),
			GUID:        "https://example.com/blog/third/",
		},
	}
}

func TestGenerateRSS_Basic(t *testing.T) {
	opts := defaultOpts()
	items := sampleItems()

	data, err := GenerateRSS(items, opts)
	if err != nil {
		t.Fatalf("GenerateRSS returned error: %v", err)
	}

	output := string(data)

	// Verify XML declaration.
	if !strings.HasPrefix(output, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("missing or incorrect XML declaration")
	}

	// Verify RSS version attribute.
	if !strings.Contains(output, `<rss version="2.0"`) {
		t.Error("missing rss version attribute")
	}

	// Verify atom namespace.
	if !strings.Contains(output, `xmlns:atom="http://www.w3.org/2005/Atom"`) {
		t.Error("missing atom namespace declaration")
	}

	// Verify channel elements.
	if !strings.Contains(output, "<title>My Site</title>") {
		t.Error("missing channel title")
	}
	if !strings.Contains(output, "<link>https://example.com</link>") {
		t.Error("missing channel link")
	}
	if !strings.Contains(output, "<description>A test site</description>") {
		t.Error("missing channel description")
	}
	if !strings.Contains(output, "<language>en</language>") {
		t.Error("missing channel language")
	}

	// Verify atom:link self-reference.
	if !strings.Contains(output, `href="https://example.com/index.xml"`) {
		t.Error("missing atom:link href")
	}
	if !strings.Contains(output, `rel="self"`) {
		t.Error("missing atom:link rel=self")
	}
	if !strings.Contains(output, `type="application/rss+xml"`) {
		t.Error("missing atom:link type")
	}

	// Verify all three items are present.
	if strings.Count(output, "<item>") != 3 {
		t.Errorf("expected 3 items, got %d", strings.Count(output, "<item>"))
	}

	// Verify item titles.
	if !strings.Contains(output, "<title>First Post</title>") {
		t.Error("missing First Post title")
	}
	if !strings.Contains(output, "<title>Second Post</title>") {
		t.Error("missing Second Post title")
	}
	if !strings.Contains(output, "<title>Third Post</title>") {
		t.Error("missing Third Post title")
	}

	// Verify CDATA in description (default mode uses Description field).
	if !strings.Contains(output, "<![CDATA[Summary of first post]]>") {
		t.Error("missing CDATA-wrapped description for first post")
	}

	// Verify the XML is valid by unmarshaling.
	var rss rssFeed
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("generated XML is not valid: %v", err)
	}
}

func TestGenerateRSS_MaxItems(t *testing.T) {
	opts := defaultOpts()
	opts.MaxItems = 2
	items := sampleItems()

	data, err := GenerateRSS(items, opts)
	if err != nil {
		t.Fatalf("GenerateRSS returned error: %v", err)
	}

	output := string(data)
	count := strings.Count(output, "<item>")
	if count != 2 {
		t.Errorf("expected 2 items with MaxItems=2, got %d", count)
	}

	// The two newest items should be present (Third Post and Second Post).
	if !strings.Contains(output, "<title>Third Post</title>") {
		t.Error("expected Third Post (newest) to be present")
	}
	if !strings.Contains(output, "<title>Second Post</title>") {
		t.Error("expected Second Post (second newest) to be present")
	}
	if strings.Contains(output, "<title>First Post</title>") {
		t.Error("First Post (oldest) should be excluded with MaxItems=2")
	}
}

func TestGenerateRSS_SortOrder(t *testing.T) {
	opts := defaultOpts()
	items := sampleItems()

	data, err := GenerateRSS(items, opts)
	if err != nil {
		t.Fatalf("GenerateRSS returned error: %v", err)
	}

	output := string(data)

	// Third Post (March) should appear before Second Post (February)
	// which should appear before First Post (January).
	thirdIdx := strings.Index(output, "<title>Third Post</title>")
	secondIdx := strings.Index(output, "<title>Second Post</title>")
	firstIdx := strings.Index(output, "<title>First Post</title>")

	if thirdIdx == -1 || secondIdx == -1 || firstIdx == -1 {
		t.Fatal("one or more items missing from output")
	}

	if thirdIdx >= secondIdx {
		t.Error("Third Post should appear before Second Post (descending date order)")
	}
	if secondIdx >= firstIdx {
		t.Error("Second Post should appear before First Post (descending date order)")
	}
}

func TestGenerateRSS_FullContent(t *testing.T) {
	opts := defaultOpts()
	opts.FullContent = true
	items := sampleItems()

	data, err := GenerateRSS(items, opts)
	if err != nil {
		t.Fatalf("GenerateRSS returned error: %v", err)
	}

	output := string(data)

	// With FullContent=true, Content field should be used.
	if !strings.Contains(output, "<![CDATA[<p>Full content of first post</p>]]>") {
		t.Error("expected full Content field in description when FullContent=true")
	}
	if strings.Contains(output, "<![CDATA[Summary of first post]]>") {
		t.Error("should not contain summary Description when FullContent=true")
	}
}

func TestGenerateRSS_EmptyItems(t *testing.T) {
	opts := defaultOpts()
	var items []FeedItem

	data, err := GenerateRSS(items, opts)
	if err != nil {
		t.Fatalf("GenerateRSS returned error: %v", err)
	}

	output := string(data)

	// Should still be valid XML with channel but no items.
	if !strings.Contains(output, "<channel>") {
		t.Error("missing channel element")
	}
	if strings.Contains(output, "<item>") {
		t.Error("should not contain any items")
	}

	// Verify valid XML.
	var rss rssFeed
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("generated XML is not valid: %v", err)
	}
	if len(rss.Channel.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(rss.Channel.Items))
	}
}

func TestGenerateRSS_Categories(t *testing.T) {
	opts := defaultOpts()
	items := []FeedItem{
		{
			Title:       "Tagged Post",
			Link:        "https://example.com/blog/tagged/",
			Description: "A tagged post",
			PubDate:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			GUID:        "https://example.com/blog/tagged/",
			Categories:  []string{"tag1", "tag2", "tag3"},
		},
	}

	data, err := GenerateRSS(items, opts)
	if err != nil {
		t.Fatalf("GenerateRSS returned error: %v", err)
	}

	output := string(data)

	if !strings.Contains(output, "<category>tag1</category>") {
		t.Error("missing category tag1")
	}
	if !strings.Contains(output, "<category>tag2</category>") {
		t.Error("missing category tag2")
	}
	if !strings.Contains(output, "<category>tag3</category>") {
		t.Error("missing category tag3")
	}
}

func TestGenerateRSS_NoAuthor(t *testing.T) {
	opts := defaultOpts()
	items := []FeedItem{
		{
			Title:       "No Author Post",
			Link:        "https://example.com/blog/no-author/",
			Description: "A post without author",
			PubDate:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			GUID:        "https://example.com/blog/no-author/",
		},
	}

	data, err := GenerateRSS(items, opts)
	if err != nil {
		t.Fatalf("GenerateRSS returned error: %v", err)
	}

	output := string(data)

	if strings.Contains(output, "<author>") {
		t.Error("author element should be omitted when author is empty")
	}
}
