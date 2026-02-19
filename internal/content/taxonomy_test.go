package content

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test helpers for taxonomy tests
// ---------------------------------------------------------------------------

func withTags(tags ...string) func(*Page) {
	return func(p *Page) { p.Tags = tags }
}

func withCategories(cats ...string) func(*Page) {
	return func(p *Page) { p.Categories = cats }
}

func withParams(params map[string]any) func(*Page) {
	return func(p *Page) { p.Params = params }
}

// defaultTaxonomies returns the standard taxonomy config used by most tests.
func defaultTaxonomies() map[string]string {
	return map[string]string{
		"tags":       "tag",
		"categories": "category",
	}
}

// ---------------------------------------------------------------------------
// Tests: BuildTaxonomies
// ---------------------------------------------------------------------------

func TestBuildTaxonomies_TagsAndCategories(t *testing.T) {
	pages := []*Page{
		newPage("Post A", withTags("Go", "Testing"), withCategories("Tech")),
		newPage("Post B", withTags("Go"), withCategories("Tech", "Life")),
		newPage("Post C", withTags("Rust"), withCategories("Tech")),
	}

	result := BuildTaxonomies(pages, defaultTaxonomies())

	// Check tags taxonomy exists.
	tagsTax, ok := result["tags"]
	if !ok {
		t.Fatal("expected 'tags' taxonomy to exist")
	}
	if tagsTax.Name != "tags" {
		t.Errorf("tags taxonomy Name = %q, want %q", tagsTax.Name, "tags")
	}
	if tagsTax.Singular != "tag" {
		t.Errorf("tags taxonomy Singular = %q, want %q", tagsTax.Singular, "tag")
	}

	// Check that "go" term has 2 pages.
	if got := len(tagsTax.Terms["go"]); got != 2 {
		t.Errorf("tags['go'] has %d pages, want 2", got)
	}
	// Check that "testing" term has 1 page.
	if got := len(tagsTax.Terms["testing"]); got != 1 {
		t.Errorf("tags['testing'] has %d pages, want 1", got)
	}
	// Check that "rust" term has 1 page.
	if got := len(tagsTax.Terms["rust"]); got != 1 {
		t.Errorf("tags['rust'] has %d pages, want 1", got)
	}

	// Check categories taxonomy.
	catsTax, ok := result["categories"]
	if !ok {
		t.Fatal("expected 'categories' taxonomy to exist")
	}
	if got := len(catsTax.Terms["tech"]); got != 3 {
		t.Errorf("categories['tech'] has %d pages, want 3", got)
	}
	if got := len(catsTax.Terms["life"]); got != 1 {
		t.Errorf("categories['life'] has %d pages, want 1", got)
	}
}

func TestBuildTaxonomies_CustomTaxonomy(t *testing.T) {
	pages := []*Page{
		newPage("Post A", withParams(map[string]any{
			"genres": []string{"sci-fi", "action"},
		})),
		newPage("Post B", withParams(map[string]any{
			"genres": []string{"action"},
		})),
		newPage("Post C"), // No params at all.
	}

	customTax := map[string]string{
		"genres": "genre",
	}

	result := BuildTaxonomies(pages, customTax)

	genresTax, ok := result["genres"]
	if !ok {
		t.Fatal("expected 'genres' taxonomy to exist")
	}
	if got := len(genresTax.Terms["sci-fi"]); got != 1 {
		t.Errorf("genres['sci-fi'] has %d pages, want 1", got)
	}
	if got := len(genresTax.Terms["action"]); got != 2 {
		t.Errorf("genres['action'] has %d pages, want 2", got)
	}
}

func TestBuildTaxonomies_EmptyPages(t *testing.T) {
	result := BuildTaxonomies(nil, defaultTaxonomies())

	tagsTax, ok := result["tags"]
	if !ok {
		t.Fatal("expected 'tags' taxonomy to exist even with no pages")
	}
	if got := len(tagsTax.Terms); got != 0 {
		t.Errorf("tags taxonomy has %d terms, want 0", got)
	}

	catsTax, ok := result["categories"]
	if !ok {
		t.Fatal("expected 'categories' taxonomy to exist even with no pages")
	}
	if got := len(catsTax.Terms); got != 0 {
		t.Errorf("categories taxonomy has %d terms, want 0", got)
	}
}

func TestBuildTaxonomies_SortsByDateNewestFirst(t *testing.T) {
	oldest := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	middle := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	newest := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	pages := []*Page{
		newPage("Oldest", withDate(oldest), withTags("go")),
		newPage("Newest", withDate(newest), withTags("go")),
		newPage("Middle", withDate(middle), withTags("go")),
	}

	result := BuildTaxonomies(pages, defaultTaxonomies())
	goPages := result["tags"].Terms["go"]

	if len(goPages) != 3 {
		t.Fatalf("tags['go'] has %d pages, want 3", len(goPages))
	}

	got := titles(goPages)
	want := []string{"Newest", "Middle", "Oldest"}
	if !equalStrings(got, want) {
		t.Errorf("tags['go'] page order = %v, want %v (newest first)", got, want)
	}
}

func TestBuildTaxonomies_NormalizesTerms(t *testing.T) {
	pages := []*Page{
		newPage("Post A", withTags("Go")),
		newPage("Post B", withTags("go")),
		newPage("Post C", withTags("GO")),
	}

	result := BuildTaxonomies(pages, defaultTaxonomies())

	// All should be normalized to lowercase "go".
	if got := len(result["tags"].Terms); got != 1 {
		t.Errorf("tags taxonomy has %d terms, want 1 (all normalized to 'go')", got)
	}
	if got := len(result["tags"].Terms["go"]); got != 3 {
		t.Errorf("tags['go'] has %d pages, want 3", got)
	}
}

func TestBuildTaxonomies_SkipsEmptyTerms(t *testing.T) {
	pages := []*Page{
		newPage("Post A", withTags("go", "", "  ")),
	}

	result := BuildTaxonomies(pages, defaultTaxonomies())

	if got := len(result["tags"].Terms); got != 1 {
		t.Errorf("tags taxonomy has %d terms, want 1 (empty terms skipped)", got)
	}
}

// ---------------------------------------------------------------------------
// Tests: GenerateTaxonomyPages
// ---------------------------------------------------------------------------

func TestGenerateTaxonomyPages_CreatesCorrectPageTypes(t *testing.T) {
	taxonomies := map[string]*Taxonomy{
		"tags": {
			Name:     "tags",
			Singular: "tag",
			Terms: map[string][]*Page{
				"go":   {newPage("Post A"), newPage("Post B")},
				"rust": {newPage("Post C")},
			},
		},
	}

	pages := GenerateTaxonomyPages(taxonomies)

	// Expect 1 taxonomy list page + 2 term pages = 3 total.
	if got := len(pages); got != 3 {
		t.Fatalf("GenerateTaxonomyPages() returned %d pages, want 3", got)
	}

	// First page should be the taxonomy list.
	listPage := pages[0]
	if listPage.Type != PageTypeTaxonomyList {
		t.Errorf("list page Type = %v, want PageTypeTaxonomyList", listPage.Type)
	}
	if listPage.Title != "Tags" {
		t.Errorf("list page Title = %q, want %q", listPage.Title, "Tags")
	}
	if listPage.URL != "/tags/" {
		t.Errorf("list page URL = %q, want %q", listPage.URL, "/tags/")
	}
	if listPage.Section != "tags" {
		t.Errorf("list page Section = %q, want %q", listPage.Section, "tags")
	}

	// Remaining pages should be term pages (sorted: "go" before "rust").
	goPage := pages[1]
	if goPage.Type != PageTypeTaxonomy {
		t.Errorf("go term page Type = %v, want PageTypeTaxonomy", goPage.Type)
	}
	if goPage.Title != "go" {
		t.Errorf("go term page Title = %q, want %q", goPage.Title, "go")
	}
	if goPage.URL != "/tags/go/" {
		t.Errorf("go term page URL = %q, want %q", goPage.URL, "/tags/go/")
	}
	if goPage.Section != "tags" {
		t.Errorf("go term page Section = %q, want %q", goPage.Section, "tags")
	}

	rustPage := pages[2]
	if rustPage.Type != PageTypeTaxonomy {
		t.Errorf("rust term page Type = %v, want PageTypeTaxonomy", rustPage.Type)
	}
	if rustPage.Title != "rust" {
		t.Errorf("rust term page Title = %q, want %q", rustPage.Title, "rust")
	}
	if rustPage.URL != "/tags/rust/" {
		t.Errorf("rust term page URL = %q, want %q", rustPage.URL, "/tags/rust/")
	}
}

func TestGenerateTaxonomyPages_TermParams(t *testing.T) {
	taxonomies := map[string]*Taxonomy{
		"tags": {
			Name:     "tags",
			Singular: "tag",
			Terms: map[string][]*Page{
				"go": {newPage("Post A"), newPage("Post B"), newPage("Post C")},
			},
		},
	}

	pages := GenerateTaxonomyPages(taxonomies)

	// Find the "go" term page (second page, after the list page).
	if len(pages) < 2 {
		t.Fatalf("expected at least 2 pages, got %d", len(pages))
	}
	termPage := pages[1]

	if termPage.Params == nil {
		t.Fatal("term page Params is nil")
	}
	if got, ok := termPage.Params["term"].(string); !ok || got != "go" {
		t.Errorf("term page Params[\"term\"] = %v, want %q", termPage.Params["term"], "go")
	}
	if got, ok := termPage.Params["taxonomy"].(string); !ok || got != "tags" {
		t.Errorf("term page Params[\"taxonomy\"] = %v, want %q", termPage.Params["taxonomy"], "tags")
	}
	if got, ok := termPage.Params["count"].(int); !ok || got != 3 {
		t.Errorf("term page Params[\"count\"] = %v, want %d", termPage.Params["count"], 3)
	}
}

func TestGenerateTaxonomyPages_MultipleTaxonomies(t *testing.T) {
	taxonomies := map[string]*Taxonomy{
		"tags": {
			Name:     "tags",
			Singular: "tag",
			Terms: map[string][]*Page{
				"go": {newPage("Post A")},
			},
		},
		"categories": {
			Name:     "categories",
			Singular: "category",
			Terms: map[string][]*Page{
				"tech": {newPage("Post A")},
				"life": {newPage("Post B")},
			},
		},
	}

	pages := GenerateTaxonomyPages(taxonomies)

	// "categories" comes before "tags" alphabetically.
	// categories: 1 list + 2 terms = 3
	// tags: 1 list + 1 term = 2
	// Total: 5
	if got := len(pages); got != 5 {
		t.Fatalf("GenerateTaxonomyPages() returned %d pages, want 5", got)
	}

	// First should be the categories list page.
	if pages[0].Title != "Categories" {
		t.Errorf("first page Title = %q, want %q", pages[0].Title, "Categories")
	}
	if pages[0].Type != PageTypeTaxonomyList {
		t.Errorf("first page Type = %v, want PageTypeTaxonomyList", pages[0].Type)
	}
	if pages[0].URL != "/categories/" {
		t.Errorf("first page URL = %q, want %q", pages[0].URL, "/categories/")
	}

	// Then "life" and "tech" term pages for categories.
	if pages[1].Title != "life" || pages[1].URL != "/categories/life/" {
		t.Errorf("pages[1] = {Title: %q, URL: %q}, want {Title: %q, URL: %q}",
			pages[1].Title, pages[1].URL, "life", "/categories/life/")
	}
	if pages[2].Title != "tech" || pages[2].URL != "/categories/tech/" {
		t.Errorf("pages[2] = {Title: %q, URL: %q}, want {Title: %q, URL: %q}",
			pages[2].Title, pages[2].URL, "tech", "/categories/tech/")
	}

	// Then the tags list page.
	if pages[3].Title != "Tags" || pages[3].URL != "/tags/" {
		t.Errorf("pages[3] = {Title: %q, URL: %q}, want {Title: %q, URL: %q}",
			pages[3].Title, pages[3].URL, "Tags", "/tags/")
	}
	if pages[3].Type != PageTypeTaxonomyList {
		t.Errorf("pages[3] Type = %v, want PageTypeTaxonomyList", pages[3].Type)
	}

	// Then the "go" term page for tags.
	if pages[4].Title != "go" || pages[4].URL != "/tags/go/" {
		t.Errorf("pages[4] = {Title: %q, URL: %q}, want {Title: %q, URL: %q}",
			pages[4].Title, pages[4].URL, "go", "/tags/go/")
	}
}

func TestGenerateTaxonomyPages_EmptyTaxonomies(t *testing.T) {
	pages := GenerateTaxonomyPages(map[string]*Taxonomy{})

	if got := len(pages); got != 0 {
		t.Errorf("GenerateTaxonomyPages(empty) returned %d pages, want 0", got)
	}
}

func TestGenerateTaxonomyPages_TaxonomyWithNoTerms(t *testing.T) {
	taxonomies := map[string]*Taxonomy{
		"tags": {
			Name:     "tags",
			Singular: "tag",
			Terms:    map[string][]*Page{},
		},
	}

	pages := GenerateTaxonomyPages(taxonomies)

	// Should still create the list page even with no terms.
	if got := len(pages); got != 1 {
		t.Fatalf("GenerateTaxonomyPages(no terms) returned %d pages, want 1", got)
	}
	if pages[0].Type != PageTypeTaxonomyList {
		t.Errorf("page Type = %v, want PageTypeTaxonomyList", pages[0].Type)
	}
	if pages[0].Title != "Tags" {
		t.Errorf("page Title = %q, want %q", pages[0].Title, "Tags")
	}
}
