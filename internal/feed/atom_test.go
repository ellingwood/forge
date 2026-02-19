package feed

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func atomOpts() FeedOptions {
	return FeedOptions{
		Title:       "My Site",
		Description: "A test site",
		Link:        "https://example.com",
		FeedLink:    "https://example.com/atom.xml",
		Language:    "en",
		Author:      "Jane Doe",
	}
}

func TestGenerateAtom_Basic(t *testing.T) {
	opts := atomOpts()
	items := sampleItems()

	data, err := GenerateAtom(items, opts)
	if err != nil {
		t.Fatalf("GenerateAtom returned error: %v", err)
	}

	output := string(data)

	// Verify XML declaration.
	if !strings.HasPrefix(output, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("missing or incorrect XML declaration")
	}

	// Verify Atom namespace.
	if !strings.Contains(output, `xmlns="http://www.w3.org/2005/Atom"`) {
		t.Error("missing Atom namespace declaration")
	}

	// Verify feed-level elements.
	if !strings.Contains(output, "<title>My Site</title>") {
		t.Error("missing feed title")
	}
	if !strings.Contains(output, "<subtitle>A test site</subtitle>") {
		t.Error("missing feed subtitle")
	}

	// Verify feed-level links.
	if !strings.Contains(output, `href="https://example.com" rel="alternate"`) {
		t.Error("missing alternate link")
	}
	if !strings.Contains(output, `href="https://example.com/atom.xml" rel="self"`) {
		t.Error("missing self link")
	}

	// Verify feed ID.
	if !strings.Contains(output, "<id>https://example.com/</id>") {
		t.Error("missing feed id")
	}

	// Verify feed-level updated.
	if !strings.Contains(output, "<updated>") {
		t.Error("missing feed updated element")
	}

	// Verify feed-level author.
	if !strings.Contains(output, "<author>") {
		t.Error("missing feed-level author")
	}
	if !strings.Contains(output, "<name>Jane Doe</name>") {
		t.Error("missing author name")
	}

	// Verify all three entries are present.
	if strings.Count(output, "<entry>") != 3 {
		t.Errorf("expected 3 entries, got %d", strings.Count(output, "<entry>"))
	}

	// Verify entry titles.
	if !strings.Contains(output, "<title>First Post</title>") {
		t.Error("missing First Post title")
	}
	if !strings.Contains(output, "<title>Second Post</title>") {
		t.Error("missing Second Post title")
	}
	if !strings.Contains(output, "<title>Third Post</title>") {
		t.Error("missing Third Post title")
	}

	// Verify entry links have rel="alternate".
	if !strings.Contains(output, `href="https://example.com/blog/first/" rel="alternate"`) {
		t.Error("missing entry link for first post")
	}

	// Verify summary is present.
	if !strings.Contains(output, "Summary of first post") {
		t.Error("missing summary for first post")
	}

	// Verify the XML is valid by unmarshaling.
	var feed atomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		t.Fatalf("generated XML is not valid: %v", err)
	}
}

func TestGenerateAtom_MaxItems(t *testing.T) {
	opts := atomOpts()
	opts.MaxItems = 2
	items := sampleItems()

	data, err := GenerateAtom(items, opts)
	if err != nil {
		t.Fatalf("GenerateAtom returned error: %v", err)
	}

	output := string(data)
	count := strings.Count(output, "<entry>")
	if count != 2 {
		t.Errorf("expected 2 entries with MaxItems=2, got %d", count)
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

func TestGenerateAtom_SortOrder(t *testing.T) {
	opts := atomOpts()
	items := sampleItems()

	data, err := GenerateAtom(items, opts)
	if err != nil {
		t.Fatalf("GenerateAtom returned error: %v", err)
	}

	output := string(data)

	// Third Post (March) should appear before Second Post (February)
	// which should appear before First Post (January).
	thirdIdx := strings.Index(output, "<title>Third Post</title>")
	secondIdx := strings.Index(output, "<title>Second Post</title>")
	firstIdx := strings.Index(output, "<title>First Post</title>")

	if thirdIdx == -1 || secondIdx == -1 || firstIdx == -1 {
		t.Fatal("one or more entries missing from output")
	}

	if thirdIdx >= secondIdx {
		t.Error("Third Post should appear before Second Post (descending date order)")
	}
	if secondIdx >= firstIdx {
		t.Error("Second Post should appear before First Post (descending date order)")
	}
}

func TestGenerateAtom_FullContent(t *testing.T) {
	opts := atomOpts()
	opts.FullContent = true
	items := sampleItems()

	data, err := GenerateAtom(items, opts)
	if err != nil {
		t.Fatalf("GenerateAtom returned error: %v", err)
	}

	output := string(data)

	// With FullContent=true, content element should be present.
	if !strings.Contains(output, `<content type="html">`) {
		t.Error("expected <content type=\"html\"> element when FullContent=true")
	}

	// Verify content contains the full HTML content.
	if !strings.Contains(output, "&lt;p&gt;Full content of first post&lt;/p&gt;") {
		t.Error("expected full Content field in content element when FullContent=true")
	}

	// Summary should still be present.
	if !strings.Contains(output, `<summary type="html">`) {
		t.Error("expected summary element to still be present when FullContent=true")
	}
	if !strings.Contains(output, "Summary of first post") {
		t.Error("expected summary text to still be present when FullContent=true")
	}
}

func TestGenerateAtom_SummaryOnly(t *testing.T) {
	opts := atomOpts()
	opts.FullContent = false
	items := sampleItems()

	data, err := GenerateAtom(items, opts)
	if err != nil {
		t.Fatalf("GenerateAtom returned error: %v", err)
	}

	output := string(data)

	// Without FullContent, content element should NOT be present.
	if strings.Contains(output, `<content`) {
		t.Error("content element should not be present when FullContent=false")
	}

	// Summary should still be present.
	if !strings.Contains(output, `<summary type="html">`) {
		t.Error("expected summary element when FullContent=false")
	}
	if !strings.Contains(output, "Summary of first post") {
		t.Error("expected summary text when FullContent=false")
	}
}

func TestGenerateAtom_EmptyItems(t *testing.T) {
	opts := atomOpts()
	var items []FeedItem

	data, err := GenerateAtom(items, opts)
	if err != nil {
		t.Fatalf("GenerateAtom returned error: %v", err)
	}

	output := string(data)

	// Should still be valid XML with feed element but no entries.
	if !strings.Contains(output, `<feed xmlns="http://www.w3.org/2005/Atom"`) {
		t.Error("missing feed element")
	}
	if strings.Contains(output, "<entry>") {
		t.Error("should not contain any entries")
	}

	// Verify feed title is present.
	if !strings.Contains(output, "<title>My Site</title>") {
		t.Error("missing feed title")
	}

	// Verify updated element is present (should be current time).
	if !strings.Contains(output, "<updated>") {
		t.Error("missing updated element for empty feed")
	}

	// Verify valid XML.
	var feed atomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		t.Fatalf("generated XML is not valid: %v", err)
	}
	if len(feed.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(feed.Entries))
	}
}

func TestGenerateAtom_Categories(t *testing.T) {
	opts := atomOpts()
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

	data, err := GenerateAtom(items, opts)
	if err != nil {
		t.Fatalf("GenerateAtom returned error: %v", err)
	}

	output := string(data)

	if !strings.Contains(output, `<category term="tag1"></category>`) {
		t.Error("missing category tag1")
	}
	if !strings.Contains(output, `<category term="tag2"></category>`) {
		t.Error("missing category tag2")
	}
	if !strings.Contains(output, `<category term="tag3"></category>`) {
		t.Error("missing category tag3")
	}
}

func TestGenerateAtom_NoAuthor(t *testing.T) {
	opts := atomOpts()
	opts.Author = "" // No feed-level author.
	items := []FeedItem{
		{
			Title:       "No Author Post",
			Link:        "https://example.com/blog/no-author/",
			Description: "A post without author",
			PubDate:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			GUID:        "https://example.com/blog/no-author/",
		},
	}

	data, err := GenerateAtom(items, opts)
	if err != nil {
		t.Fatalf("GenerateAtom returned error: %v", err)
	}

	output := string(data)

	if strings.Contains(output, "<author>") {
		t.Error("author element should be omitted when author is empty")
	}
	if strings.Contains(output, "<name>") {
		t.Error("name element should be omitted when author is empty")
	}
}
