package build

import (
	"strings"
	"testing"
)

func TestGenerateAliasPages_Basic(t *testing.T) {
	aliases := []AliasPage{
		{
			AliasURL:     "/old-post/",
			CanonicalURL: "/blog/new-post/",
		},
	}

	result := GenerateAliasPages(aliases)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	html, ok := result["old-post/index.html"]
	if !ok {
		t.Fatalf("expected key 'old-post/index.html', got keys: %v", keys(result))
	}

	content := string(html)
	if !strings.Contains(content, `content="0; url=/blog/new-post/"`) {
		t.Error("HTML should contain meta refresh with canonical URL")
	}
	if !strings.Contains(content, `href="/blog/new-post/"`) {
		t.Error("HTML should contain canonical link href")
	}
}

func TestGenerateAliasPages_Multiple(t *testing.T) {
	aliases := []AliasPage{
		{AliasURL: "/old-post/", CanonicalURL: "/blog/new-post/"},
		{AliasURL: "/legacy/page/", CanonicalURL: "/updated/page/"},
		{AliasURL: "/archive/2020/", CanonicalURL: "/blog/2020-recap/"},
	}

	result := GenerateAliasPages(aliases)

	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	expectedPaths := []string{
		"old-post/index.html",
		"legacy/page/index.html",
		"archive/2020/index.html",
	}

	for _, path := range expectedPaths {
		if _, ok := result[path]; !ok {
			t.Errorf("expected key %q in result", path)
		}
	}
}

func TestGenerateAliasPages_PathNormalization(t *testing.T) {
	tests := []struct {
		name     string
		aliasURL string
		wantPath string
	}{
		{
			name:     "trailing slash",
			aliasURL: "/old-post/",
			wantPath: "old-post/index.html",
		},
		{
			name:     "no trailing slash",
			aliasURL: "/old-post",
			wantPath: "old-post/index.html",
		},
		{
			name:     "root path",
			aliasURL: "/",
			wantPath: "index.html",
		},
		{
			name:     "nested path with trailing slash",
			aliasURL: "/a/b/c/",
			wantPath: "a/b/c/index.html",
		},
		{
			name:     "nested path without trailing slash",
			aliasURL: "/a/b/c",
			wantPath: "a/b/c/index.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aliases := []AliasPage{
				{AliasURL: tt.aliasURL, CanonicalURL: "/target/"},
			}

			result := GenerateAliasPages(aliases)

			if _, ok := result[tt.wantPath]; !ok {
				t.Errorf("aliasURL %q: expected path %q, got keys: %v",
					tt.aliasURL, tt.wantPath, keys(result))
			}
		})
	}
}

func TestGenerateAliasPages_Empty(t *testing.T) {
	result := GenerateAliasPages(nil)
	if len(result) != 0 {
		t.Errorf("expected empty map for nil input, got %d entries", len(result))
	}

	result = GenerateAliasPages([]AliasPage{})
	if len(result) != 0 {
		t.Errorf("expected empty map for empty input, got %d entries", len(result))
	}
}

func TestGenerateAliasPages_HTMLContent(t *testing.T) {
	aliases := []AliasPage{
		{
			AliasURL:     "/old/",
			CanonicalURL: "/blog/new-post/",
		},
	}

	result := GenerateAliasPages(aliases)
	html := string(result["old/index.html"])

	// Verify it is valid HTML5.
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("HTML should start with <!DOCTYPE html>")
	}

	// Verify meta refresh tag.
	if !strings.Contains(html, `<meta http-equiv="refresh" content="0; url=/blog/new-post/">`) {
		t.Error("HTML should contain meta http-equiv refresh tag with correct URL")
	}

	// Verify canonical link.
	if !strings.Contains(html, `<link rel="canonical" href="/blog/new-post/">`) {
		t.Error("HTML should contain canonical link element")
	}

	// Verify body link.
	if !strings.Contains(html, `<a href="/blog/new-post/">/blog/new-post/</a>`) {
		t.Error("HTML should contain an anchor link to the canonical URL in the body")
	}

	// Verify charset.
	if !strings.Contains(html, `<meta charset="utf-8">`) {
		t.Error("HTML should contain UTF-8 charset meta tag")
	}

	// Verify title.
	if !strings.Contains(html, `<title>Redirect</title>`) {
		t.Error("HTML should contain 'Redirect' title")
	}
}

// keys returns the keys of a map for error reporting.
func keys(m map[string][]byte) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}
