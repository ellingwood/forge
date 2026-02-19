package search

import (
	"encoding/json"
	"testing"
)

func TestGenerateIndex_Basic(t *testing.T) {
	entries := []IndexEntry{
		{
			Title:      "First Post",
			URL:        "/posts/first/",
			Tags:       []string{"go", "testing"},
			Categories: []string{"programming"},
			Summary:    "A first post",
			Content:    "This is the content of the first post.",
		},
		{
			Title:      "Second Post",
			URL:        "/posts/second/",
			Tags:       []string{"rust"},
			Categories: []string{"programming"},
			Summary:    "A second post",
			Content:    "This is the content of the second post.",
		},
	}

	data, err := GenerateIndex(entries, 0)
	if err != nil {
		t.Fatalf("GenerateIndex returned error: %v", err)
	}

	var result []IndexEntry
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}

	if result[0].Title != "First Post" {
		t.Errorf("expected title 'First Post', got %q", result[0].Title)
	}
	if result[0].URL != "/posts/first/" {
		t.Errorf("expected URL '/posts/first/', got %q", result[0].URL)
	}
	if len(result[0].Tags) != 2 || result[0].Tags[0] != "go" || result[0].Tags[1] != "testing" {
		t.Errorf("unexpected tags: %v", result[0].Tags)
	}
	if result[1].Title != "Second Post" {
		t.Errorf("expected title 'Second Post', got %q", result[1].Title)
	}
	if result[1].Content != "This is the content of the second post." {
		t.Errorf("expected full content, got %q", result[1].Content)
	}
}

func TestGenerateIndex_MaxContentLen(t *testing.T) {
	entries := []IndexEntry{
		{
			Title:   "Long Post",
			URL:     "/posts/long/",
			Content: "The quick brown fox jumps over the lazy dog and runs away",
		},
	}

	data, err := GenerateIndex(entries, 30)
	if err != nil {
		t.Fatalf("GenerateIndex returned error: %v", err)
	}

	var result []IndexEntry
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}

	content := result[0].Content
	// Should be truncated at word boundary before position 30 with "..." appended.
	if len(content) == 0 {
		t.Fatal("expected non-empty content after truncation")
	}
	if content[len(content)-3:] != "..." {
		t.Errorf("expected truncated content to end with '...', got %q", content)
	}
	// "The quick brown fox jumps over" is 30 chars, so last space before 30 is at position 26 ("over").
	// Truncated = "The quick brown fox jumps" + "..."
	expected := "The quick brown fox jumps..."
	if content != expected {
		t.Errorf("expected %q, got %q", expected, content)
	}
}

func TestGenerateIndex_EmptyEntries(t *testing.T) {
	data, err := GenerateIndex(nil, 0)
	if err != nil {
		t.Fatalf("GenerateIndex returned error: %v", err)
	}

	var result []IndexEntry
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result))
	}

	// Also verify it's a JSON array, not null.
	trimmed := string(data)
	if trimmed != "[]" {
		t.Errorf("expected '[]', got %q", trimmed)
	}
}

func TestGenerateIndex_OmitEmpty(t *testing.T) {
	entries := []IndexEntry{
		{
			Title: "Minimal Post",
			URL:   "/posts/minimal/",
		},
	}

	data, err := GenerateIndex(entries, 0)
	if err != nil {
		t.Fatalf("GenerateIndex returned error: %v", err)
	}

	// The JSON should not contain tags, categories, summary, or content keys.
	var raw []map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(raw) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(raw))
	}

	entry := raw[0]
	for _, key := range []string{"tags", "categories", "summary", "content"} {
		if _, ok := entry[key]; ok {
			t.Errorf("expected key %q to be omitted, but it was present", key)
		}
	}

	// Title and URL should always be present.
	if entry["title"] != "Minimal Post" {
		t.Errorf("expected title 'Minimal Post', got %v", entry["title"])
	}
	if entry["url"] != "/posts/minimal/" {
		t.Errorf("expected url '/posts/minimal/', got %v", entry["url"])
	}
}

func TestStripHTML_Basic(t *testing.T) {
	input := "<p>Hello <strong>world</strong></p>"
	expected := "Hello world"
	result := StripHTML(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestStripHTML_Entities(t *testing.T) {
	input := "Tom &amp; Jerry &lt;friends&gt; said &quot;hello&#39;s&quot;"
	expected := "Tom & Jerry <friends> said \"hello's\""
	result := StripHTML(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestStripHTML_Nested(t *testing.T) {
	input := "<div><p>Nested <em><strong>tags</strong></em> here</p></div>"
	expected := "Nested tags here"
	result := StripHTML(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestStripHTML_Empty(t *testing.T) {
	result := StripHTML("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestTruncateAtWord_Short(t *testing.T) {
	input := "short text"
	result := TruncateAtWord(input, 100)
	if result != input {
		t.Errorf("expected %q, got %q", input, result)
	}
}

func TestTruncateAtWord_Long(t *testing.T) {
	input := "The quick brown fox jumps over the lazy dog"
	result := TruncateAtWord(input, 20)
	// First 20 chars: "The quick brown fox "
	// Last space at or before 20 is position 19 (the space after "fox").
	// Actually: T(0)h(1)e(2) (3)q(4)u(5)i(6)c(7)k(8) (9)b(10)r(11)o(12)w(13)n(14) (15)f(16)o(17)x(18) (19)j(20)
	// s[:20] = "The quick brown fox " -> lastSpace = 19 -> "The quick brown fox" + "..."
	expected := "The quick brown fox..."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
