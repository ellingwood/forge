package content

import "fmt"

// Pager represents a single page of paginated results.
type Pager struct {
	Pages      []*Page
	PageNumber int
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevURL    string
	NextURL    string
	First      string // URL of first page
	Last       string // URL of last page
}

// Paginate splits pages into groups of pageSize and returns a slice of Pagers.
// URL pattern: page 1 = baseURL, page 2 = baseURL + "page/2/", etc.
// Edge cases:
//   - Empty pages returns an empty slice.
//   - pageSize <= 0 is treated as 10.
//   - Fewer pages than pageSize produces a single Pager.
func Paginate(pages []*Page, pageSize int, baseURL string) []*Pager {
	if len(pages) == 0 {
		return nil
	}

	if pageSize <= 0 {
		pageSize = 10
	}

	// Calculate total number of result pages.
	totalPages := (len(pages) + pageSize - 1) / pageSize

	// Determine the URL of the last page.
	lastURL := baseURL
	if totalPages > 1 {
		lastURL = fmt.Sprintf("%spage/%d/", baseURL, totalPages)
	}

	pagers := make([]*Pager, 0, totalPages)

	for i := 0; i < totalPages; i++ {
		start := i * pageSize
		end := start + pageSize
		if end > len(pages) {
			end = len(pages)
		}

		pageNum := i + 1

		pager := &Pager{
			Pages:      pages[start:end],
			PageNumber: pageNum,
			TotalPages: totalPages,
			HasPrev:    pageNum > 1,
			HasNext:    pageNum < totalPages,
			First:      baseURL,
			Last:       lastURL,
		}

		// Set PrevURL.
		if pager.HasPrev {
			if pageNum == 2 {
				pager.PrevURL = baseURL
			} else {
				pager.PrevURL = fmt.Sprintf("%spage/%d/", baseURL, pageNum-1)
			}
		}

		// Set NextURL.
		if pager.HasNext {
			pager.NextURL = fmt.Sprintf("%spage/%d/", baseURL, pageNum+1)
		}

		pagers = append(pagers, pager)
	}

	return pagers
}
