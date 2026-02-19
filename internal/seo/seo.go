package seo

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"strings"
	"time"
)

// SitemapEntry represents a page in the sitemap.
type SitemapEntry struct {
	URL     string
	Lastmod time.Time
}

// PageMeta holds metadata needed for SEO tag generation.
type PageMeta struct {
	Title         string
	Description   string
	URL           string // full URL like https://example.com/blog/post/
	Permalink     string
	PageType      string // "article", "website", etc.
	SiteName      string
	Author        string
	Date          time.Time
	Tags          []string
	CoverImage    string // URL to cover image
	Language      string
	BaseURL       string
	TitleTemplate string // e.g. "%s | Site Name"
}

// sitemapURLSet is the root element of a sitemap XML document.
type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// sitemapURL represents a single URL entry in the sitemap.
type sitemapURL struct {
	Loc     string `xml:"loc"`
	Lastmod string `xml:"lastmod,omitempty"`
}

// GenerateSitemap produces an XML sitemap per the sitemaps.org protocol.
// It includes the XML declaration, a <urlset> root with the sitemaps.org xmlns,
// and each entry as a <url> with <loc> and optional <lastmod> (date only, YYYY-MM-DD).
// The <lastmod> element is only included when the time is non-zero.
func GenerateSitemap(entries []SitemapEntry) ([]byte, error) {
	urlset := sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  make([]sitemapURL, 0, len(entries)),
	}

	for _, e := range entries {
		u := sitemapURL{Loc: e.URL}
		if !e.Lastmod.IsZero() {
			u.Lastmod = e.Lastmod.Format("2006-01-02")
		}
		urlset.URLs = append(urlset.URLs, u)
	}

	output, err := xml.MarshalIndent(urlset, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("seo: marshaling sitemap: %w", err)
	}

	result := []byte(xml.Header)
	result = append(result, output...)
	result = append(result, '\n')
	return result, nil
}

// GenerateRobotsTxt produces a robots.txt that allows all crawlers and references
// the provided sitemap URL.
func GenerateRobotsTxt(sitemapURL string) []byte {
	return fmt.Appendf(nil, "User-agent: *\nAllow: /\n\nSitemap: %s\n", sitemapURL)
}

// OpenGraphMeta generates HTML meta tags for the Open Graph protocol.
// It returns og:title, og:description, og:url, og:type, og:site_name,
// og:image (if cover exists), and og:locale (from language) as a string
// of <meta> tags separated by newlines.
func OpenGraphMeta(meta PageMeta) string {
	var tags []string

	tags = append(tags, ogTag("og:title", meta.Title))
	tags = append(tags, ogTag("og:description", meta.Description))
	tags = append(tags, ogTag("og:url", meta.URL))
	tags = append(tags, ogTag("og:type", meta.PageType))
	tags = append(tags, ogTag("og:site_name", meta.SiteName))

	if meta.CoverImage != "" {
		tags = append(tags, ogTag("og:image", meta.CoverImage))
	}

	if meta.Language != "" {
		tags = append(tags, ogTag("og:locale", meta.Language))
	}

	return strings.Join(tags, "\n")
}

// ogTag generates a single Open Graph meta tag.
func ogTag(property, content string) string {
	return fmt.Sprintf(`<meta property="%s" content="%s">`, property, html.EscapeString(content))
}

// TwitterCardMeta generates Twitter card meta tags. If a cover image is present,
// the card type is "summary_large_image"; otherwise it is "summary".
func TwitterCardMeta(meta PageMeta) string {
	var tags []string

	cardType := "summary"
	if meta.CoverImage != "" {
		cardType = "summary_large_image"
	}

	tags = append(tags, twitterTag("twitter:card", cardType))
	tags = append(tags, twitterTag("twitter:title", meta.Title))
	tags = append(tags, twitterTag("twitter:description", meta.Description))

	if meta.CoverImage != "" {
		tags = append(tags, twitterTag("twitter:image", meta.CoverImage))
	}

	return strings.Join(tags, "\n")
}

// twitterTag generates a single Twitter card meta tag.
func twitterTag(name, content string) string {
	return fmt.Sprintf(`<meta name="%s" content="%s">`, name, html.EscapeString(content))
}

// jsonLDArticle is the structure for schema.org Article JSON-LD.
type jsonLDArticle struct {
	Context       string        `json:"@context"`
	Type          string        `json:"@type"`
	Headline      string        `json:"headline"`
	DatePublished string        `json:"datePublished"`
	Author        *jsonLDPerson `json:"author,omitempty"`
	Description   string        `json:"description"`
	URL           string        `json:"url"`
	Image         string        `json:"image,omitempty"`
}

// jsonLDPerson represents a schema.org Person.
type jsonLDPerson struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

// JSONLDArticle generates a <script type="application/ld+json"> block with
// schema.org Article markup. It includes @context, @type, headline,
// datePublished (RFC3339), author, description, url, and image (if cover exists).
func JSONLDArticle(meta PageMeta) string {
	article := jsonLDArticle{
		Context:       "https://schema.org",
		Type:          "Article",
		Headline:      meta.Title,
		DatePublished: meta.Date.Format(time.RFC3339),
		Description:   meta.Description,
		URL:           meta.URL,
	}

	if meta.Author != "" {
		article.Author = &jsonLDPerson{
			Type: "Person",
			Name: meta.Author,
		}
	}

	if meta.CoverImage != "" {
		article.Image = meta.CoverImage
	}

	data, err := json.Marshal(article)
	if err != nil {
		// This should not happen with simple string fields, but handle gracefully.
		return ""
	}

	return fmt.Sprintf(`<script type="application/ld+json">%s</script>`, string(data))
}

// CanonicalURL returns a <link rel="canonical"> tag for the given permalink.
func CanonicalURL(permalink string) string {
	return fmt.Sprintf(`<link rel="canonical" href="%s">`, html.EscapeString(permalink))
}

// SEOTitle applies a title template to a page title. If template is empty,
// the page title is returned as-is. If the template contains "%s", fmt.Sprintf
// is used to substitute the page title. Otherwise the page title is returned as-is.
func SEOTitle(pageTitle string, template string) string {
	if template == "" {
		return pageTitle
	}
	if strings.Contains(template, "%s") {
		return fmt.Sprintf(template, pageTitle)
	}
	return pageTitle
}
