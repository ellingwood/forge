package build

import (
	"fmt"
	"strings"
)

// AliasPage represents a redirect from an alias URL to the canonical URL.
type AliasPage struct {
	AliasURL     string // e.g. "/old-post/"
	CanonicalURL string // e.g. "/blog/new-post/"
}

// aliasTemplate is the HTML template used for redirect pages.
const aliasTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta http-equiv="refresh" content="0; url=%s">
  <link rel="canonical" href="%s">
  <title>Redirect</title>
</head>
<body>
  <p>This page has moved to <a href="%s">%s</a>.</p>
</body>
</html>
`

// GenerateAliasPages generates HTML redirect pages for the given aliases.
// Each redirect page uses a <meta http-equiv="refresh"> tag to redirect
// to the canonical URL. Returns a map from output file path to HTML content.
// The output path is derived from AliasURL: "/old-post/" -> "old-post/index.html"
func GenerateAliasPages(aliases []AliasPage) map[string][]byte {
	result := make(map[string][]byte, len(aliases))

	for _, alias := range aliases {
		filePath := aliasURLToFilePath(alias.AliasURL)
		html := fmt.Sprintf(aliasTemplate,
			alias.CanonicalURL,
			alias.CanonicalURL,
			alias.CanonicalURL,
			alias.CanonicalURL,
		)
		result[filePath] = []byte(html)
	}

	return result
}

// aliasURLToFilePath converts an alias URL to an output file path.
// The leading slash is stripped and the path is normalized to end with /index.html.
//
// Examples:
//
//	"/old-post/"  -> "old-post/index.html"
//	"/old-post"   -> "old-post/index.html"
//	"/"           -> "index.html"
func aliasURLToFilePath(url string) string {
	// Strip leading slash.
	path := strings.TrimPrefix(url, "/")

	// Strip trailing slash.
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return "index.html"
	}

	return path + "/index.html"
}
