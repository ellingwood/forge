package search

import (
	"encoding/json"
	"strings"
)

// IndexEntry represents a single page in the search index.
type IndexEntry struct {
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	Tags       []string `json:"tags,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	Content    string   `json:"content,omitempty"`
}

// GenerateIndex serializes entries as a JSON array. If maxContentLen > 0,
// each entry's Content field is truncated to that many characters at a word
// boundary. The output uses indented JSON for readability.
func GenerateIndex(entries []IndexEntry, maxContentLen int) ([]byte, error) {
	if entries == nil {
		entries = []IndexEntry{}
	}

	if maxContentLen > 0 {
		// Work on a copy so we don't mutate the caller's slice.
		truncated := make([]IndexEntry, len(entries))
		copy(truncated, entries)
		for i := range truncated {
			truncated[i].Content = TruncateAtWord(truncated[i].Content, maxContentLen)
		}
		entries = truncated
	}

	return json.MarshalIndent(entries, "", "  ")
}

// StripHTML removes HTML tags from a string, producing plain text. It uses a
// simple state-machine approach (no regexp): scanning character by character,
// tracking whether we are inside a tag. Common HTML entities are decoded, and
// runs of whitespace are collapsed to a single space.
func StripHTML(html string) string {
	var b strings.Builder
	b.Grow(len(html))

	inTag := false
	for i := 0; i < len(html); i++ {
		ch := html[i]
		switch {
		case ch == '<':
			inTag = true
		case ch == '>':
			inTag = false
		case !inTag:
			b.WriteByte(ch)
		}
	}

	result := b.String()

	// Decode common HTML entities.
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&#39;", "'")

	// Collapse whitespace: replace any run of whitespace characters with a
	// single space, then trim leading/trailing whitespace.
	result = collapseWhitespace(result)

	return result
}

// collapseWhitespace replaces runs of whitespace (spaces, tabs, newlines) with
// a single space and trims leading/trailing whitespace.
func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	inSpace := false
	for _, ch := range s {
		switch ch {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
		default:
			b.WriteRune(ch)
			inSpace = false
		}
	}

	return strings.TrimSpace(b.String())
}

// TruncateAtWord truncates s at the last space before maxLen characters. If s
// is shorter than or equal to maxLen it is returned as-is. If truncated, "..."
// is appended to indicate truncation.
func TruncateAtWord(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// Find the last space at or before maxLen.
	truncated := s[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}
