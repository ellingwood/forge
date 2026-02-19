package feed

import (
	"encoding/xml"
	"sort"
	"time"
)

// FeedOptions configures feed generation.
type FeedOptions struct {
	Title       string
	Description string
	Link        string // site URL e.g. "https://example.com"
	FeedLink    string // feed URL e.g. "https://example.com/index.xml"
	Language    string
	Author      string
	MaxItems    int  // 0 means no limit
	FullContent bool // true = include full content, false = summary only
}

// FeedItem represents a single item in a feed.
type FeedItem struct {
	Title       string
	Link        string // full permalink
	Description string // summary or full HTML content
	Content     string // full HTML content (for Atom content:encoded)
	Author      string
	PubDate     time.Time
	GUID        string // typically same as Link
	Categories  []string
}

// CDATA wraps text in a CDATA section when marshaled to XML.
type CDATA struct {
	Text string `xml:",cdata"`
}

// rssFeed is the top-level RSS 2.0 XML structure.
type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	AtomNS  string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

// rssChannel represents the <channel> element in RSS 2.0.
type rssChannel struct {
	Title       string      `xml:"title"`
	Link        string      `xml:"link"`
	Description string      `xml:"description"`
	Language    string      `xml:"language,omitempty"`
	AtomLink    rssAtomLink `xml:"atom:link"`
	Items       []rssItem   `xml:"item"`
}

// rssAtomLink represents the atom:link self-reference element.
type rssAtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

// rssItem represents a single <item> element in the RSS feed.
type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	PubDate     string   `xml:"pubDate"`
	GUID        string   `xml:"guid"`
	Description CDATA    `xml:"description"`
	Author      string   `xml:"author,omitempty"`
	Categories  []string `xml:"category,omitempty"`
}

// GenerateRSS generates an RSS 2.0 XML feed from the given items and options.
// Items are sorted by PubDate descending. If opts.MaxItems > 0, only that many
// items are included. If opts.FullContent is true, item.Content is used for the
// description; otherwise item.Description is used.
func GenerateRSS(items []FeedItem, opts FeedOptions) ([]byte, error) {
	// Make a copy to avoid mutating the caller's slice.
	sorted := make([]FeedItem, len(items))
	copy(sorted, items)

	// Sort by PubDate descending (newest first).
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].PubDate.After(sorted[j].PubDate)
	})

	// Apply MaxItems limit.
	if opts.MaxItems > 0 && len(sorted) > opts.MaxItems {
		sorted = sorted[:opts.MaxItems]
	}

	// Build RSS items.
	rssItems := make([]rssItem, 0, len(sorted))
	for _, item := range sorted {
		desc := item.Description
		if opts.FullContent && item.Content != "" {
			desc = item.Content
		}

		ri := rssItem{
			Title:       item.Title,
			Link:        item.Link,
			PubDate:     item.PubDate.Format(time.RFC1123Z),
			GUID:        item.GUID,
			Description: CDATA{Text: desc},
			Author:      item.Author,
			Categories:  item.Categories,
		}
		rssItems = append(rssItems, ri)
	}

	feed := rssFeed{
		Version: "2.0",
		AtomNS:  "http://www.w3.org/2005/Atom",
		Channel: rssChannel{
			Title:       opts.Title,
			Link:        opts.Link,
			Description: opts.Description,
			Language:    opts.Language,
			AtomLink: rssAtomLink{
				Href: opts.FeedLink,
				Rel:  "self",
				Type: "application/rss+xml",
			},
			Items: rssItems,
		},
	}

	output, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, err
	}

	// Prepend the XML declaration.
	header := []byte(xml.Header)
	result := make([]byte, 0, len(header)+len(output))
	result = append(result, header...)
	result = append(result, output...)

	return result, nil
}
