package mcpserver

import (
	"fmt"
	"strings"
	"time"

	"github.com/agnivade/levenshtein"
	"gopkg.in/yaml.v3"
)

// abbreviations maps common abbreviations to their full forms for taxonomy similarity.
var abbreviations = map[string]string{
	"k8s":    "kubernetes",
	"js":     "javascript",
	"ts":     "typescript",
	"tf":     "terraform",
	"py":     "python",
	"infra":  "infrastructure",
	"devops": "devops",
}

// findSimilarTerms finds existing terms similar to input using Levenshtein distance
// and abbreviation detection.
func findSimilarTerms(input string, existing []string, threshold int) []string {
	inputLower := strings.ToLower(input)
	seen := make(map[string]bool)
	var similar []string

	// Check abbreviation expansion
	if expanded, ok := abbreviations[inputLower]; ok {
		for _, term := range existing {
			if strings.ToLower(term) == expanded && !seen[term] {
				seen[term] = true
				similar = append(similar, term)
			}
		}
	}

	// Check if input is an expansion of an abbreviation of a term
	for abbr, expanded := range abbreviations {
		if inputLower == expanded {
			// input is the expanded form; check if any existing term is the abbreviation
			for _, term := range existing {
				if strings.ToLower(term) == abbr && !seen[term] {
					seen[term] = true
					similar = append(similar, term)
				}
			}
		}
	}

	// Levenshtein distance
	for _, term := range existing {
		termLower := strings.ToLower(term)
		if termLower == inputLower {
			continue // exact match
		}
		dist := levenshtein.ComputeDistance(inputLower, termLower)
		if dist <= threshold && !seen[term] {
			seen[term] = true
			similar = append(similar, term)
		}
	}

	return similar
}

// frontmatterData is a partial parse of YAML frontmatter.
type frontmatterData struct {
	Title       string   `yaml:"title"`
	Date        string   `yaml:"date"`
	Draft       *bool    `yaml:"draft"`
	Tags        []string `yaml:"tags"`
	Categories  []string `yaml:"categories"`
	Series      string   `yaml:"series"`
	Project     string   `yaml:"project"`
	Description string   `yaml:"description"`
	Summary     string   `yaml:"summary"`
	Slug        string   `yaml:"slug"`
	Weight      int      `yaml:"weight"`
	Layout      string   `yaml:"layout"`
}

// validateFrontmatter validates YAML frontmatter against the Forge schema.
func validateFrontmatter(raw string, existingTags, existingCats, projectSlugs []string) ValidateFrontmatterOutput {
	var data frontmatterData
	var errs []ValidationError
	var warns []ValidationWarning

	if err := yaml.Unmarshal([]byte(raw), &data); err != nil {
		errs = append(errs, ValidationError{
			Field:   "_yaml",
			Message: fmt.Sprintf("invalid YAML: %s", err.Error()),
		})
		return ValidateFrontmatterOutput{Valid: false, Errors: errs, Warnings: warns}
	}

	// Required: title
	if strings.TrimSpace(data.Title) == "" {
		errs = append(errs, ValidationError{
			Field:   "title",
			Message: "title is required",
		})
	}

	// Date format validation
	if data.Date != "" {
		validFormats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02",
		}
		valid := false
		for _, f := range validFormats {
			if _, err := time.Parse(f, data.Date); err == nil {
				valid = true
				break
			}
		}
		if !valid {
			errs = append(errs, ValidationError{
				Field:   "date",
				Message: "Invalid date format: expected ISO 8601 (e.g. 2025-01-15 or 2025-01-15T10:00:00Z)",
				Value:   data.Date,
			})
		}
	}

	// Layout validation
	if data.Layout != "" {
		validLayouts := []string{"post", "project", "page", "single", "list"}
		found := false
		for _, l := range validLayouts {
			if data.Layout == l {
				found = true
				break
			}
		}
		if !found {
			warns = append(warns, ValidationWarning{
				Field:   "layout",
				Message: fmt.Sprintf("Layout %q may not exist; valid values: post, project, page", data.Layout),
			})
		}
	}

	// Tags similarity check
	for _, tag := range data.Tags {
		similar := findSimilarTerms(tag, existingTags, 2)
		for _, s := range similar {
			if strings.ToLower(s) != strings.ToLower(tag) {
				warns = append(warns, ValidationWarning{
					Field:      "tags",
					Message:    fmt.Sprintf("Tag %q is similar to existing tag %q. Did you mean %q?", tag, s, s),
					Suggestion: s,
				})
			}
		}
	}

	// Categories similarity check
	for _, cat := range data.Categories {
		similar := findSimilarTerms(cat, existingCats, 2)
		for _, s := range similar {
			if strings.ToLower(s) != strings.ToLower(cat) {
				warns = append(warns, ValidationWarning{
					Field:      "categories",
					Message:    fmt.Sprintf("Category %q is similar to existing category %q. Did you mean %q?", cat, s, s),
					Suggestion: s,
				})
			}
		}
	}

	// Project slug validation
	if data.Project != "" && len(projectSlugs) > 0 {
		found := false
		for _, s := range projectSlugs {
			if s == data.Project {
				found = true
				break
			}
		}
		if !found {
			similar := findSimilarTerms(data.Project, projectSlugs, 2)
			w := ValidationWarning{
				Field:   "project",
				Message: fmt.Sprintf("Project %q does not match any existing project slug", data.Project),
			}
			if len(similar) > 0 {
				w.Message = fmt.Sprintf("Project %q does not match any existing project slug. Did you mean %q?", data.Project, similar[0])
				w.Suggestion = similar[0]
			}
			warns = append(warns, w)
		}
	}

	// Normalize and re-marshal
	normalized := raw
	if len(errs) == 0 {
		if b, err := yaml.Marshal(&data); err == nil {
			normalized = string(b)
		}
	}

	return ValidateFrontmatterOutput{
		Valid:                 len(errs) == 0,
		Errors:                errs,
		Warnings:              warns,
		NormalizedFrontmatter: normalized,
	}
}
