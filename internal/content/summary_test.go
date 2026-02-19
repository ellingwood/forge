package content

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests: GenerateSummary
// ---------------------------------------------------------------------------

func TestGenerateSummary_WithMoreMarker(t *testing.T) {
	rawMD := "This is the intro.\n\n<!--more-->\n\nThis is the rest."
	renderedHTML := "<p>This is the intro.</p>\n<!--more-->\n<p>This is the rest.</p>"

	got := GenerateSummary(rawMD, renderedHTML, 300)

	if !strings.Contains(got, "This is the intro.") {
		t.Errorf("expected summary to contain intro, got %q", got)
	}
	if strings.Contains(got, "This is the rest.") {
		t.Errorf("expected summary not to contain content after <!--more-->, got %q", got)
	}
}

func TestGenerateSummary_WithoutMoreMarker(t *testing.T) {
	rawMD := "First paragraph.\n\nSecond paragraph."
	renderedHTML := "<p>First paragraph.</p>\n<p>Second paragraph.</p>"

	got := GenerateSummary(rawMD, renderedHTML, 300)

	if got != "<p>First paragraph.</p>" {
		t.Errorf("expected first paragraph, got %q", got)
	}
}

func TestGenerateSummary_Truncation(t *testing.T) {
	rawMD := "A very long paragraph with lots of words."
	longText := strings.Repeat("word ", 100) // 500 chars
	renderedHTML := "<p>" + longText + "</p>"

	got := GenerateSummary(rawMD, renderedHTML, 50)

	plain := StripHTMLTags(got)
	if len(plain) > 60 { // some tolerance for "..."
		t.Errorf("expected truncated summary, plain text length = %d", len(plain))
	}
	if !strings.HasSuffix(plain, "...") {
		t.Errorf("expected truncated summary to end with '...', got %q", plain)
	}
}

func TestGenerateSummary_EmptyInput(t *testing.T) {
	got := GenerateSummary("", "", 300)
	if got != "" {
		t.Errorf("expected empty summary for empty input, got %q", got)
	}
}

func TestGenerateSummary_DefaultMaxLength(t *testing.T) {
	rawMD := "Short text."
	renderedHTML := "<p>Short text.</p>"

	// maxLength 0 should default to 300
	got := GenerateSummary(rawMD, renderedHTML, 0)
	if got != "<p>Short text.</p>" {
		t.Errorf("expected full paragraph with default maxLength, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Tests: CalculateReadingTime
// ---------------------------------------------------------------------------

func TestCalculateReadingTime(t *testing.T) {
	tests := []struct {
		name    string
		words   int
		wantMin int
	}{
		{"200 words = 1 min", 200, 1},
		{"400 words = 2 min", 400, 2},
		{"100 words = 1 min (minimum)", 100, 1},
		{"0 words = 0 min", 0, 0},
		{"1 word = 1 min", 1, 1},
		{"600 words = 3 min", 600, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := strings.Repeat("word ", tt.words)
			got := CalculateReadingTime(content)
			if got != tt.wantMin {
				t.Errorf("CalculateReadingTime(%d words) = %d, want %d", tt.words, got, tt.wantMin)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: CalculateWordCount
// ---------------------------------------------------------------------------

func TestCalculateWordCount(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"hello world", 2},
		{"  multiple   spaces  ", 2},
		{"", 0},
		{"one", 1},
		{"tabs\tand\nnewlines\twork", 4},
		{"a b c d e f g h i j", 10},
	}

	for _, tt := range tests {
		got := CalculateWordCount(tt.input)
		if got != tt.want {
			t.Errorf("CalculateWordCount(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: GenerateMetaDescription
// ---------------------------------------------------------------------------

func TestGenerateMetaDescription(t *testing.T) {
	t.Run("strips HTML and truncates", func(t *testing.T) {
		summary := "<p>This is a <strong>bold</strong> summary with HTML.</p>"
		got := GenerateMetaDescription(summary, 30)

		if strings.Contains(got, "<") {
			t.Errorf("expected no HTML tags, got %q", got)
		}
		if len(got) > 40 { // tolerance for "..."
			t.Errorf("expected truncated result, got length %d: %q", len(got), got)
		}
	})

	t.Run("short summary not truncated", func(t *testing.T) {
		summary := "<p>Short.</p>"
		got := GenerateMetaDescription(summary, 100)

		if got != "Short." {
			t.Errorf("expected %q, got %q", "Short.", got)
		}
	})

	t.Run("normalizes whitespace", func(t *testing.T) {
		summary := "<p>Multiple   spaces\nand\nnewlines</p>"
		got := GenerateMetaDescription(summary, 100)

		if got != "Multiple spaces and newlines" {
			t.Errorf("expected normalized whitespace, got %q", got)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: StripHTMLTags
// ---------------------------------------------------------------------------

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p>hello</p>", "hello"},
		{"<strong>bold</strong> text", "bold text"},
		{"no tags here", "no tags here"},
		{"<a href=\"url\">link</a>", "link"},
		{"<br/>line<br/>break", "linebreak"},
		{"", ""},
		{"<div class=\"foo\"><span>nested</span></div>", "nested"},
	}

	for _, tt := range tests {
		got := StripHTMLTags(tt.input)
		if got != tt.want {
			t.Errorf("StripHTMLTags(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: TruncateAtWord
// ---------------------------------------------------------------------------

func TestTruncateAtWord(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello world foo", 15, "hello world foo"},     // fits exactly, no truncation
		{"hello world foo", 16, "hello world foo"},     // fits within maxLen, no truncation
		{"hello world foo bar", 15, "hello world..."},  // s[:15]="hello world foo", last space at 11
		{"short", 100, "short"},                        // fits within maxLen
		{"hello world", 5, "hello..."},                 // s[:5]="hello", no space found => full truncated + "..."
		{"hello world", 0, "hello world"},              // maxLen 0 returns original
		{"", 10, ""},                                   // empty string
		{"oneword", 3, "one..."},                       // single long word, no space to break at
		{"a b c d e", 5, "a b..."},                     // s[:5]="a b c", last space at 3
	}

	for _, tt := range tests {
		got := TruncateAtWord(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateAtWord(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}
