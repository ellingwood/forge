package template

import (
	"html/template"
	"time"
)

// PageContext is the data passed to every template as ".".
type PageContext struct {
	Title           string
	Description     string
	Content         template.HTML
	Summary         template.HTML
	Date            time.Time
	Lastmod         time.Time
	Draft           bool
	Slug            string
	URL             string
	Permalink       string
	ReadingTime     int
	WordCount       int
	Tags            []string
	Categories      []string
	Series          string
	Params          map[string]any
	Cover           *CoverImage
	TableOfContents template.HTML
	PrevPage        *PageContext
	NextPage        *PageContext
	Section         string
	Type            string // "single", "list", "taxonomy", "home", etc.

	Site *SiteContext
}

// CoverImage mirrors content.CoverImage for templates.
type CoverImage struct {
	Image   string
	Alt     string
	Caption string
}

// SiteContext holds site-wide data accessible as .Site in templates.
type SiteContext struct {
	Title       string
	Description string
	BaseURL     string
	Language    string
	Author      AuthorContext
	Menu        []MenuItemContext
	Params      map[string]any
	Data        map[string]any
	Pages       []*PageContext
	Sections    map[string][]*PageContext
	Taxonomies  map[string]map[string][]*PageContext
	BuildDate   time.Time
}

// AuthorContext mirrors config.AuthorConfig for templates.
type AuthorContext struct {
	Name   string
	Email  string
	Bio    string
	Avatar string
	Social SocialContext
}

// SocialContext mirrors config.SocialConfig for templates.
type SocialContext struct {
	GitHub   string
	LinkedIn string
	Twitter  string
	Mastodon string
	Email    string
}

// MenuItemContext represents a single navigation menu entry in templates.
type MenuItemContext struct {
	Name   string
	URL    string
	Weight int
}
