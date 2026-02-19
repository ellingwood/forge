package mcpserver

import (
	"testing"
)

// Integration tests are in server_integration_test.go
// This file contains unit tests for helper functions.

func TestFindSimilarTerms(t *testing.T) {
	existing := []string{"kubernetes", "javascript", "typescript", "terraform", "python", "infrastructure"}

	tests := []struct {
		input string
		want  []string
	}{
		{"k8s", []string{"kubernetes"}},
		{"js", []string{"javascript"}},
		{"kubernetez", []string{"kubernetes"}}, // distance 1
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := findSimilarTerms(tc.input, existing, 2)
			if len(got) == 0 && len(tc.want) > 0 {
				t.Errorf("findSimilarTerms(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestValidateFrontmatter(t *testing.T) {
	tags := []string{"go", "kubernetes", "devops"}
	cats := []string{"Infrastructure", "Programming"}
	projects := []string{"forge", "myapp"}

	t.Run("valid frontmatter", func(t *testing.T) {
		fm := `title: "My Post"
date: 2025-01-15T10:00:00Z
draft: true
tags:
  - go
`
		result := validateFrontmatter(fm, tags, cats, projects)
		if !result.Valid {
			t.Errorf("expected valid, got errors: %v", result.Errors)
		}
	})

	t.Run("missing title", func(t *testing.T) {
		fm := `date: 2025-01-15T10:00:00Z`
		result := validateFrontmatter(fm, tags, cats, projects)
		if result.Valid {
			t.Error("expected invalid due to missing title")
		}
	})

	t.Run("invalid date format", func(t *testing.T) {
		fm := `title: "My Post"
date: "January 15, 2025"`
		result := validateFrontmatter(fm, tags, cats, projects)
		if result.Valid {
			t.Error("expected invalid due to bad date format")
		}
	})

	t.Run("similar tag warning", func(t *testing.T) {
		fm := `title: "My Post"
date: 2025-01-15T10:00:00Z
tags:
  - k8s
`
		result := validateFrontmatter(fm, tags, cats, projects)
		if len(result.Warnings) == 0 {
			t.Error("expected warning for k8s similar to kubernetes")
		}
	})

	t.Run("unknown project slug warning", func(t *testing.T) {
		fm := `title: "My Post"
date: 2025-01-15T10:00:00Z
project: "nonexistent"
`
		result := validateFrontmatter(fm, tags, cats, projects)
		if !result.Valid {
			t.Errorf("expected valid (project mismatch is a warning, not error), got errors: %v", result.Errors)
		}
		if len(result.Warnings) == 0 {
			t.Error("expected warning for unknown project slug")
		}
		foundProjectWarning := false
		for _, w := range result.Warnings {
			if w.Field == "project" {
				foundProjectWarning = true
				break
			}
		}
		if !foundProjectWarning {
			t.Error("expected a warning with field 'project'")
		}
	})

	t.Run("valid project slug no warning", func(t *testing.T) {
		fm := `title: "My Post"
date: 2025-01-15T10:00:00Z
project: "forge"
`
		result := validateFrontmatter(fm, tags, cats, projects)
		if !result.Valid {
			t.Errorf("expected valid, got errors: %v", result.Errors)
		}
		for _, w := range result.Warnings {
			if w.Field == "project" {
				t.Errorf("unexpected project warning: %s", w.Message)
			}
		}
	})
}
