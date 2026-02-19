package feed

import (
	"encoding/xml"
	"sort"
	"time"
)

// atomFeed is the top-level Atom 1.0 XML structure.
type atomFeed struct {
	XMLName  xml.Name    `xml:"feed"`
	Xmlns    string      `xml:"xmlns,attr"`
	Title    string      `xml:"title"`
	Subtitle string      `xml:"subtitle,omitempty"`
	Links    []atomLink  `xml:"link"`
	ID       string      `xml:"id"`
	Updated  string      `xml:"updated"`
	Author   *atomAuthor `xml:"author,omitempty"`
	Entries  []atomEntry `xml:"entry"`
}

// atomLink represents a <link> element in the Atom feed.
type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

// atomAuthor represents an <author> element in the Atom feed.
type atomAuthor struct {
	Name string `xml:"name"`
}

// atomEntry represents a single <entry> element in the Atom feed.
type atomEntry struct {
	Title      string          `xml:"title"`
	Link       atomLink        `xml:"link"`
	ID         string          `xml:"id"`
	Published  string          `xml:"published"`
	Updated    string          `xml:"updated"`
	Summary    *atomContent    `xml:"summary,omitempty"`
	Content    *atomContent    `xml:"content,omitempty"`
	Author     *atomAuthor     `xml:"author,omitempty"`
	Categories []atomCategory  `xml:"category,omitempty"`
}

// atomContent represents a text element with a type attribute (e.g. summary, content).
type atomContent struct {
	Type string `xml:"type,attr"`
	Body string `xml:",chardata"`
}

// atomCategory represents a <category> element with a term attribute.
type atomCategory struct {
	Term string `xml:"term,attr"`
}

// GenerateAtom generates an Atom 1.0 XML feed from the given items and options.
// Items are sorted by PubDate descending. If opts.MaxItems > 0, only that many
// items are included. If opts.FullContent is true, a <content type="html"> element
// is included with item.Content; otherwise it is omitted. A <summary type="html">
// element is always included with item.Description.
func GenerateAtom(items []FeedItem, opts FeedOptions) ([]byte, error) {
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

	// Determine feed-level updated time: most recent item's PubDate, or now.
	var updatedTime time.Time
	if len(sorted) > 0 {
		updatedTime = sorted[0].PubDate
	} else {
		updatedTime = time.Now().UTC()
	}

	// Build Atom entries.
	entries := make([]atomEntry, 0, len(sorted))
	for _, item := range sorted {
		entry := atomEntry{
			Title: item.Title,
			Link: atomLink{
				Href: item.Link,
				Rel:  "alternate",
			},
			ID:        item.GUID,
			Published: item.PubDate.Format(time.RFC3339),
			Updated:   item.PubDate.Format(time.RFC3339),
			Summary: &atomContent{
				Type: "html",
				Body: item.Description,
			},
		}

		if opts.FullContent && item.Content != "" {
			entry.Content = &atomContent{
				Type: "html",
				Body: item.Content,
			}
		}

		if item.Author != "" {
			entry.Author = &atomAuthor{Name: item.Author}
		}

		if len(item.Categories) > 0 {
			cats := make([]atomCategory, len(item.Categories))
			for i, c := range item.Categories {
				cats[i] = atomCategory{Term: c}
			}
			entry.Categories = cats
		}

		entries = append(entries, entry)
	}

	// Build feed-level links.
	links := []atomLink{
		{Href: opts.Link, Rel: "alternate"},
		{Href: opts.FeedLink, Rel: "self"},
	}

	// Build the feed.
	feed := atomFeed{
		Xmlns:    "http://www.w3.org/2005/Atom",
		Title:    opts.Title,
		Subtitle: opts.Description,
		Links:    links,
		ID:       opts.Link + "/",
		Updated:  updatedTime.Format(time.RFC3339),
		Entries:  entries,
	}

	if opts.Author != "" {
		feed.Author = &atomAuthor{Name: opts.Author}
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
