package content

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newPage creates a *Page with the given title and default zero values for
// everything else. Use the functional option helpers below to set fields.
func newPage(title string, opts ...func(*Page)) *Page {
	p := &Page{Title: title}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func withDate(t time.Time) func(*Page) {
	return func(p *Page) { p.Date = t }
}

func withWeight(w int) func(*Page) {
	return func(p *Page) { p.Weight = w }
}

func withDraft(d bool) func(*Page) {
	return func(p *Page) { p.Draft = d }
}

func withExpiryDate(t time.Time) func(*Page) {
	return func(p *Page) { p.ExpiryDate = t }
}

// titles extracts the Title field from each page for easy comparison.
func titles(pages []*Page) []string {
	out := make([]string, len(pages))
	for i, p := range pages {
		out[i] = p.Title
	}
	return out
}

// equalStrings compares two string slices for equality.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Tests: PageType
// ---------------------------------------------------------------------------

func TestPageTypeString(t *testing.T) {
	tests := []struct {
		pt   PageType
		want string
	}{
		{PageTypeSingle, "single"},
		{PageTypeList, "list"},
		{PageTypeTaxonomy, "taxonomy"},
		{PageTypeTaxonomyList, "taxonomylist"},
		{PageTypeHome, "home"},
		{PageType(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.pt.String()
		if got != tt.want {
			t.Errorf("PageType(%d).String() = %q, want %q", int(tt.pt), got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: Sorting
// ---------------------------------------------------------------------------

func TestSortByDate(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	earlier := now.Add(-48 * time.Hour)
	later := now.Add(48 * time.Hour)

	t.Run("ascending", func(t *testing.T) {
		pages := []*Page{
			newPage("Middle", withDate(now)),
			newPage("Earliest", withDate(earlier)),
			newPage("Latest", withDate(later)),
		}
		SortByDate(pages, true)
		got := titles(pages)
		want := []string{"Earliest", "Middle", "Latest"}
		if !equalStrings(got, want) {
			t.Errorf("SortByDate(ascending) = %v, want %v", got, want)
		}
	})

	t.Run("descending", func(t *testing.T) {
		pages := []*Page{
			newPage("Middle", withDate(now)),
			newPage("Earliest", withDate(earlier)),
			newPage("Latest", withDate(later)),
		}
		SortByDate(pages, false)
		got := titles(pages)
		want := []string{"Latest", "Middle", "Earliest"}
		if !equalStrings(got, want) {
			t.Errorf("SortByDate(descending) = %v, want %v", got, want)
		}
	})

	t.Run("zero dates sort stably", func(t *testing.T) {
		pages := []*Page{
			newPage("HasDate", withDate(now)),
			newPage("ZeroA"),
			newPage("ZeroB"),
		}
		SortByDate(pages, true)
		// Zero time is before any real date in ascending order
		got := titles(pages)
		want := []string{"ZeroA", "ZeroB", "HasDate"}
		if !equalStrings(got, want) {
			t.Errorf("SortByDate(ascending, zero dates) = %v, want %v", got, want)
		}
	})
}

func TestSortByWeight(t *testing.T) {
	pages := []*Page{
		newPage("Unset", withWeight(0)),
		newPage("Heavy", withWeight(10)),
		newPage("Light", withWeight(1)),
		newPage("Medium", withWeight(5)),
		newPage("AlsoUnset", withWeight(0)),
	}
	SortByWeight(pages)
	got := titles(pages)
	want := []string{"Light", "Medium", "Heavy", "Unset", "AlsoUnset"}
	if !equalStrings(got, want) {
		t.Errorf("SortByWeight() = %v, want %v", got, want)
	}
}

func TestSortByTitle(t *testing.T) {
	pages := []*Page{
		newPage("Charlie"),
		newPage("alpha"),
		newPage("Bravo"),
		newPage("delta"),
	}
	SortByTitle(pages)
	got := titles(pages)
	want := []string{"alpha", "Bravo", "Charlie", "delta"}
	if !equalStrings(got, want) {
		t.Errorf("SortByTitle() = %v, want %v", got, want)
	}
}

// ---------------------------------------------------------------------------
// Tests: Filtering
// ---------------------------------------------------------------------------

func TestFilterDrafts(t *testing.T) {
	pages := []*Page{
		newPage("Published1", withDraft(false)),
		newPage("Draft1", withDraft(true)),
		newPage("Published2", withDraft(false)),
		newPage("Draft2", withDraft(true)),
	}

	filtered := FilterDrafts(pages)
	got := titles(filtered)
	want := []string{"Published1", "Published2"}
	if !equalStrings(got, want) {
		t.Errorf("FilterDrafts() = %v, want %v", got, want)
	}

	// Verify original slice is not mutated (still has 4 elements).
	if len(pages) != 4 {
		t.Errorf("FilterDrafts() mutated original slice: len = %d, want 4", len(pages))
	}
}

func TestFilterFuture(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	pages := []*Page{
		newPage("PastPost", withDate(past)),
		newPage("FuturePost", withDate(future)),
		newPage("AnotherPast", withDate(past)),
	}

	filtered := FilterFuture(pages)
	got := titles(filtered)
	want := []string{"PastPost", "AnotherPast"}
	if !equalStrings(got, want) {
		t.Errorf("FilterFuture() = %v, want %v", got, want)
	}

	// Verify original slice is not mutated.
	if len(pages) != 3 {
		t.Errorf("FilterFuture() mutated original slice: len = %d, want 3", len(pages))
	}
}

func TestFilterExpired(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	pages := []*Page{
		newPage("NoExpiry"),                                  // zero ExpiryDate, should be kept
		newPage("ExpiredPost", withExpiryDate(past)),         // expired, should be removed
		newPage("FutureExpiry", withExpiryDate(future)),      // not yet expired, keep
		newPage("AlsoNoExpiry"),                              // zero ExpiryDate, keep
	}

	filtered := FilterExpired(pages)
	got := titles(filtered)
	want := []string{"NoExpiry", "FutureExpiry", "AlsoNoExpiry"}
	if !equalStrings(got, want) {
		t.Errorf("FilterExpired() = %v, want %v", got, want)
	}

	// Verify original slice is not mutated.
	if len(pages) != 4 {
		t.Errorf("FilterExpired() mutated original slice: len = %d, want 4", len(pages))
	}
}
