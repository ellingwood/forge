package content

import (
	"slices"
	"sort"
	"strings"
	"time"
)

// PageType represents the kind of page being rendered.
type PageType int

const (
	PageTypeSingle       PageType = iota // A regular content page
	PageTypeList                         // A section listing page
	PageTypeTaxonomy                     // A taxonomy term page (e.g., a specific tag)
	PageTypeTaxonomyList                 // A taxonomy listing page (e.g., all tags)
	PageTypeHome                         // The site home page
)

// String returns the human-readable name for a PageType.
func (pt PageType) String() string {
	switch pt {
	case PageTypeSingle:
		return "single"
	case PageTypeList:
		return "list"
	case PageTypeTaxonomy:
		return "taxonomy"
	case PageTypeTaxonomyList:
		return "taxonomylist"
	case PageTypeHome:
		return "home"
	default:
		return "unknown"
	}
}

// CoverImage holds metadata for a page's cover/hero image.
type CoverImage struct {
	Image   string
	Alt     string
	Caption string
}

// Page is the central content model in Forge. It represents a single piece of
// content (typically a Markdown file) along with all its associated metadata,
// rendered output, and relationships to other pages.
type Page struct {
	// Core metadata
	Title       string
	Slug        string
	URL         string // Relative permalink (e.g., "/blog/my-post/")
	Permalink   string // Absolute permalink (e.g., "https://example.com/blog/my-post/")
	Description string
	Summary     string

	// Dates
	Date       time.Time
	Lastmod    time.Time
	ExpiryDate time.Time

	// Content
	RawContent      string // Raw markdown
	Content         string // Rendered HTML
	TableOfContents string // Rendered TOC HTML
	WordCount       int
	ReadingTime     int // Minutes

	// Classification
	Draft   bool
	Type    PageType
	Section string // e.g., "blog", "projects"
	Layout  string // Explicit layout override
	Weight  int

	// Taxonomies
	Tags       []string
	Categories []string
	Series     string

	// Navigation
	PrevPage *Page
	NextPage *Page
	Aliases  []string

	// Media
	Cover *CoverImage

	// Author override
	Author string

	// Bundle info
	IsBundle    bool
	BundleDir   string   // Directory path for page bundles
	BundleFiles []string // Co-located asset file paths

	// Source info
	SourcePath string // Original file path relative to content dir
	SourceDir  string // Directory containing the source file

	// Arbitrary params
	Params map[string]any
}

// SortByDate sorts pages by their Date field. When ascending is true, older
// pages come first; when false, newer pages come first.
func SortByDate(pages []*Page, ascending bool) {
	sort.SliceStable(pages, func(i, j int) bool {
		if ascending {
			return pages[i].Date.Before(pages[j].Date)
		}
		return pages[i].Date.After(pages[j].Date)
	})
}

// SortByWeight sorts pages by Weight in ascending order. Pages with Weight == 0
// (unset) are placed at the end.
func SortByWeight(pages []*Page) {
	sort.SliceStable(pages, func(i, j int) bool {
		wi, wj := pages[i].Weight, pages[j].Weight
		// Both zero: maintain original order
		if wi == 0 && wj == 0 {
			return false
		}
		// Zero goes last
		if wi == 0 {
			return false
		}
		if wj == 0 {
			return true
		}
		return wi < wj
	})
}

// SortByTitle sorts pages alphabetically by Title using case-insensitive comparison.
func SortByTitle(pages []*Page) {
	sort.SliceStable(pages, func(i, j int) bool {
		return strings.ToLower(pages[i].Title) < strings.ToLower(pages[j].Title)
	})
}

// FilterDrafts returns a new slice with all draft pages removed.
func FilterDrafts(pages []*Page) []*Page {
	return slices.DeleteFunc(slices.Clone(pages), func(p *Page) bool {
		return p.Draft
	})
}

// FilterFuture returns a new slice with pages whose Date is in the future removed.
func FilterFuture(pages []*Page) []*Page {
	now := time.Now()
	return slices.DeleteFunc(slices.Clone(pages), func(p *Page) bool {
		return p.Date.After(now)
	})
}

// FilterExpired returns a new slice with pages whose ExpiryDate is non-zero and
// in the past removed.
func FilterExpired(pages []*Page) []*Page {
	now := time.Now()
	return slices.DeleteFunc(slices.Clone(pages), func(p *Page) bool {
		return !p.ExpiryDate.IsZero() && p.ExpiryDate.Before(now)
	})
}
