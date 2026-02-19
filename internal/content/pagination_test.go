package content

import (
	"fmt"
	"testing"
)

// makePages creates n pages with sequential titles for testing pagination.
func makePages(n int) []*Page {
	pages := make([]*Page, n)
	for i := 0; i < n; i++ {
		pages[i] = &Page{Title: fmt.Sprintf("Page %d", i+1)}
	}
	return pages
}

func TestPaginate_Basic(t *testing.T) {
	pages := makePages(10)
	pagers := Paginate(pages, 3, "/blog/")

	if len(pagers) != 4 {
		t.Fatalf("expected 4 pagers, got %d", len(pagers))
	}

	// Pager 1: pages 1-3
	p := pagers[0]
	if p.PageNumber != 1 {
		t.Errorf("pager[0].PageNumber = %d, want 1", p.PageNumber)
	}
	if p.TotalPages != 4 {
		t.Errorf("pager[0].TotalPages = %d, want 4", p.TotalPages)
	}
	if len(p.Pages) != 3 {
		t.Errorf("pager[0] has %d pages, want 3", len(p.Pages))
	}
	if p.HasPrev {
		t.Error("pager[0].HasPrev should be false")
	}
	if !p.HasNext {
		t.Error("pager[0].HasNext should be true")
	}

	// Pager 2: pages 4-6
	p = pagers[1]
	if p.PageNumber != 2 {
		t.Errorf("pager[1].PageNumber = %d, want 2", p.PageNumber)
	}
	if len(p.Pages) != 3 {
		t.Errorf("pager[1] has %d pages, want 3", len(p.Pages))
	}
	if !p.HasPrev {
		t.Error("pager[1].HasPrev should be true")
	}
	if !p.HasNext {
		t.Error("pager[1].HasNext should be true")
	}

	// Pager 4: pages 10 (last, only 1 item)
	p = pagers[3]
	if p.PageNumber != 4 {
		t.Errorf("pager[3].PageNumber = %d, want 4", p.PageNumber)
	}
	if len(p.Pages) != 1 {
		t.Errorf("pager[3] has %d pages, want 1", len(p.Pages))
	}
	if !p.HasPrev {
		t.Error("pager[3].HasPrev should be true")
	}
	if p.HasNext {
		t.Error("pager[3].HasNext should be false")
	}
}

func TestPaginate_SinglePage(t *testing.T) {
	pages := makePages(3)
	pagers := Paginate(pages, 10, "/blog/")

	if len(pagers) != 1 {
		t.Fatalf("expected 1 pager, got %d", len(pagers))
	}

	p := pagers[0]
	if p.PageNumber != 1 {
		t.Errorf("PageNumber = %d, want 1", p.PageNumber)
	}
	if p.TotalPages != 1 {
		t.Errorf("TotalPages = %d, want 1", p.TotalPages)
	}
	if len(p.Pages) != 3 {
		t.Errorf("has %d pages, want 3", len(p.Pages))
	}
	if p.HasPrev {
		t.Error("HasPrev should be false")
	}
	if p.HasNext {
		t.Error("HasNext should be false")
	}
	if p.PrevURL != "" {
		t.Errorf("PrevURL = %q, want empty", p.PrevURL)
	}
	if p.NextURL != "" {
		t.Errorf("NextURL = %q, want empty", p.NextURL)
	}
}

func TestPaginate_EmptyPages(t *testing.T) {
	pagers := Paginate(nil, 5, "/blog/")
	if pagers != nil {
		t.Errorf("expected nil for empty pages, got %v", pagers)
	}

	pagers = Paginate([]*Page{}, 5, "/blog/")
	if pagers != nil {
		t.Errorf("expected nil for empty slice, got %v", pagers)
	}
}

func TestPaginate_URLs(t *testing.T) {
	pages := makePages(10)
	pagers := Paginate(pages, 3, "/blog/")

	tests := []struct {
		index   int
		prevURL string
		nextURL string
		first   string
		last    string
	}{
		{0, "", "/blog/page/2/", "/blog/", "/blog/page/4/"},
		{1, "/blog/", "/blog/page/3/", "/blog/", "/blog/page/4/"},
		{2, "/blog/page/2/", "/blog/page/4/", "/blog/", "/blog/page/4/"},
		{3, "/blog/page/3/", "", "/blog/", "/blog/page/4/"},
	}

	for _, tt := range tests {
		p := pagers[tt.index]
		if p.PrevURL != tt.prevURL {
			t.Errorf("pager[%d].PrevURL = %q, want %q", tt.index, p.PrevURL, tt.prevURL)
		}
		if p.NextURL != tt.nextURL {
			t.Errorf("pager[%d].NextURL = %q, want %q", tt.index, p.NextURL, tt.nextURL)
		}
		if p.First != tt.first {
			t.Errorf("pager[%d].First = %q, want %q", tt.index, p.First, tt.first)
		}
		if p.Last != tt.last {
			t.Errorf("pager[%d].Last = %q, want %q", tt.index, p.Last, tt.last)
		}
	}
}

func TestPaginate_HasPrevHasNext(t *testing.T) {
	pages := makePages(9)
	pagers := Paginate(pages, 3, "/")

	if len(pagers) != 3 {
		t.Fatalf("expected 3 pagers, got %d", len(pagers))
	}

	// Page 1: no prev, has next
	if pagers[0].HasPrev {
		t.Error("pager[0].HasPrev should be false")
	}
	if !pagers[0].HasNext {
		t.Error("pager[0].HasNext should be true")
	}

	// Page 2: has prev, has next
	if !pagers[1].HasPrev {
		t.Error("pager[1].HasPrev should be true")
	}
	if !pagers[1].HasNext {
		t.Error("pager[1].HasNext should be true")
	}

	// Page 3: has prev, no next
	if !pagers[2].HasPrev {
		t.Error("pager[2].HasPrev should be true")
	}
	if pagers[2].HasNext {
		t.Error("pager[2].HasNext should be false")
	}
}

func TestPaginate_PageSizeZeroDefaultsToTen(t *testing.T) {
	pages := makePages(25)
	pagers := Paginate(pages, 0, "/blog/")

	if len(pagers) != 3 {
		t.Fatalf("expected 3 pagers (25 pages / 10 per page), got %d", len(pagers))
	}

	// First pager should have 10 items.
	if len(pagers[0].Pages) != 10 {
		t.Errorf("pager[0] has %d pages, want 10", len(pagers[0].Pages))
	}
	// Second pager should have 10 items.
	if len(pagers[1].Pages) != 10 {
		t.Errorf("pager[1] has %d pages, want 10", len(pagers[1].Pages))
	}
	// Third pager should have 5 items.
	if len(pagers[2].Pages) != 5 {
		t.Errorf("pager[2] has %d pages, want 5", len(pagers[2].Pages))
	}
}

func TestPaginate_NegativePageSizeDefaultsToTen(t *testing.T) {
	pages := makePages(15)
	pagers := Paginate(pages, -5, "/blog/")

	if len(pagers) != 2 {
		t.Fatalf("expected 2 pagers (15 pages / 10 per page), got %d", len(pagers))
	}

	if len(pagers[0].Pages) != 10 {
		t.Errorf("pager[0] has %d pages, want 10", len(pagers[0].Pages))
	}
	if len(pagers[1].Pages) != 5 {
		t.Errorf("pager[1] has %d pages, want 5", len(pagers[1].Pages))
	}
}

func TestPaginate_SinglePageLastURLEqualsBaseURL(t *testing.T) {
	pages := makePages(3)
	pagers := Paginate(pages, 10, "/blog/")

	if len(pagers) != 1 {
		t.Fatalf("expected 1 pager, got %d", len(pagers))
	}

	if pagers[0].Last != "/blog/" {
		t.Errorf("single page Last = %q, want %q", pagers[0].Last, "/blog/")
	}
	if pagers[0].First != "/blog/" {
		t.Errorf("single page First = %q, want %q", pagers[0].First, "/blog/")
	}
}
