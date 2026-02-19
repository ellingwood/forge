package content

import (
	"regexp"
	"strings"
)

// moreMarker is the HTML comment used to delimit the summary portion of content.
const moreMarker = "<!--more-->"

// htmlTagRe matches HTML tags for stripping.
var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// firstParaRe extracts the first <p>...</p> block from HTML.
var firstParaRe = regexp.MustCompile(`(?s)<p[^>]*>(.*?)</p>`)

// GenerateSummary produces a summary for a page. Priority:
//  1. If rawMD contains a <!--more--> marker, the rendered HTML is split on
//     the marker and the content before it is returned.
//  2. Otherwise, the first <p>...</p> from renderedHTML is returned.
//  3. The result is truncated to maxLength characters of text content if needed.
//     If maxLength <= 0, it defaults to 300.
func GenerateSummary(rawMD string, renderedHTML string, maxLength int) string {
	if maxLength <= 0 {
		maxLength = 300
	}

	var summary string

	if strings.Contains(rawMD, moreMarker) {
		// Split the rendered HTML on the <!--more--> marker.
		parts := strings.SplitN(renderedHTML, moreMarker, 2)
		summary = strings.TrimSpace(parts[0])
	} else {
		// Extract first paragraph from rendered HTML.
		match := firstParaRe.FindString(renderedHTML)
		if match != "" {
			summary = match
		}
	}

	// Truncate if the plain text content exceeds maxLength.
	plainText := StripHTMLTags(summary)
	if len(plainText) > maxLength {
		truncated := TruncateAtWord(plainText, maxLength)
		summary = "<p>" + truncated + "</p>"
	}

	return summary
}

// CalculateReadingTime estimates reading time at approximately 200 words per
// minute. It always returns at least 1 for non-empty content.
func CalculateReadingTime(content string) int {
	wc := CalculateWordCount(content)
	if wc == 0 {
		return 0
	}
	minutes := wc / 200
	if minutes < 1 {
		minutes = 1
	}
	return minutes
}

// CalculateWordCount counts the number of words in a string by splitting
// on whitespace.
func CalculateWordCount(content string) int {
	return len(strings.Fields(content))
}

// GenerateMetaDescription creates a plain text description from a summary.
// It strips HTML tags and truncates at a word boundary, with a maximum of
// maxLen characters.
func GenerateMetaDescription(summary string, maxLen int) string {
	plain := StripHTMLTags(summary)
	plain = strings.Join(strings.Fields(plain), " ") // normalize whitespace
	return TruncateAtWord(plain, maxLen)
}

// StripHTMLTags removes HTML tags from a string, returning plain text.
func StripHTMLTags(s string) string {
	return htmlTagRe.ReplaceAllString(s, "")
}

// TruncateAtWord truncates text at a word boundary, appending "..." if
// the text was truncated. If the text fits within maxLen, it is returned
// unchanged. If maxLen <= 0, the original string is returned.
func TruncateAtWord(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
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
