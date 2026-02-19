package content

import (
	"fmt"
	"sort"
	"strings"
)

// Taxonomy holds all terms and their associated pages for a taxonomy type.
type Taxonomy struct {
	Name     string             // e.g., "tags"
	Singular string             // e.g., "tag"
	Terms    map[string][]*Page // term -> pages
}

// BuildTaxonomies creates taxonomy maps from all pages based on config.
// The taxonomies parameter maps plural names to singular names,
// e.g., {"tags": "tag", "categories": "category"}.
func BuildTaxonomies(pages []*Page, taxonomies map[string]string) map[string]*Taxonomy {
	result := make(map[string]*Taxonomy, len(taxonomies))

	for plural, singular := range taxonomies {
		tax := &Taxonomy{
			Name:     plural,
			Singular: singular,
			Terms:    make(map[string][]*Page),
		}

		for _, p := range pages {
			var terms []string
			switch plural {
			case "tags":
				terms = p.Tags
			case "categories":
				terms = p.Categories
			default:
				// For custom taxonomies, look in the page's Params map.
				if p.Params != nil {
					if v, ok := p.Params[plural]; ok {
						if s, err := toStringSlice(v); err == nil {
							terms = s
						}
					}
				}
			}

			for _, term := range terms {
				normalized := strings.ToLower(strings.TrimSpace(term))
				if normalized == "" {
					continue
				}
				tax.Terms[normalized] = append(tax.Terms[normalized], p)
			}
		}

		// Sort pages within each term by date, newest first.
		for term := range tax.Terms {
			SortByDate(tax.Terms[term], false)
		}

		result[plural] = tax
	}

	return result
}

// GenerateTaxonomyPages creates virtual pages for taxonomy listings.
// For each taxonomy (e.g., tags), it creates:
//   - A terms page at /tags/ (lists all tags) with PageTypeTaxonomyList
//   - A term page at /tags/go/ (lists pages with tag "go") with PageTypeTaxonomy
func GenerateTaxonomyPages(taxonomies map[string]*Taxonomy) []*Page {
	var pages []*Page

	// Sort taxonomy names for deterministic output.
	taxNames := make([]string, 0, len(taxonomies))
	for name := range taxonomies {
		taxNames = append(taxNames, name)
	}
	sort.Strings(taxNames)

	for _, name := range taxNames {
		tax := taxonomies[name]

		// Create the taxonomy list page (e.g., /tags/).
		listPage := &Page{
			Title:   capitalizeFirst(name),
			URL:     fmt.Sprintf("/%s/", name),
			Type:    PageTypeTaxonomyList,
			Section: name,
			Params:  map[string]any{},
		}
		pages = append(pages, listPage)

		// Sort term names for deterministic output.
		termNames := make([]string, 0, len(tax.Terms))
		for term := range tax.Terms {
			termNames = append(termNames, term)
		}
		sort.Strings(termNames)

		// Create a page for each term (e.g., /tags/go/).
		for _, term := range termNames {
			termPages := tax.Terms[term]
			termPage := &Page{
				Title:   term,
				URL:     fmt.Sprintf("/%s/%s/", name, term),
				Type:    PageTypeTaxonomy,
				Section: name,
				Params: map[string]any{
					"term":     term,
					"taxonomy": name,
					"count":    len(termPages),
				},
			}
			pages = append(pages, termPage)
		}
	}

	return pages
}

// capitalizeFirst returns s with the first letter uppercased.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
