// Package scaffold provides functions for creating new sites, posts, pages,
// and projects in the Forge static site generator.
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// nowFunc is the function used to get the current time.
// It is a package-level variable so tests can override it.
var nowFunc = time.Now

// Slugify converts a title string into a URL-friendly slug.
// It lowercases the input, replaces spaces with hyphens, strips characters
// that are not letters, digits, or hyphens, collapses multiple hyphens,
// and trims leading/trailing hyphens. Unicode letters are preserved.
func Slugify(title string) string {
	// Normalize Unicode to NFC form (e.g., combining accents become precomposed).
	s := norm.NFC.String(title)
	s = strings.ToLower(s)

	// Replace spaces and underscores with hyphens.
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Keep only letters, digits, and hyphens.
	var buf strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			buf.WriteRune(r)
		}
	}
	s = buf.String()

	// Collapse multiple consecutive hyphens.
	multiHyphen := regexp.MustCompile(`-{2,}`)
	s = multiHyphen.ReplaceAllString(s, "-")

	// Trim leading/trailing hyphens.
	s = strings.Trim(s, "-")

	return s
}

// NewSite creates a new site directory with the standard Forge structure.
// It returns an error if the directory already exists.
func NewSite(name string) error {
	// Check if directory already exists.
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("directory %q already exists", name)
	}

	// Create the directory structure.
	dirs := []string{
		filepath.Join(name, "content", "blog"),
		filepath.Join(name, "content", "projects"),
		filepath.Join(name, "content", "pages"),
		filepath.Join(name, "layouts"),
		filepath.Join(name, "static"),
		filepath.Join(name, "data"),
		filepath.Join(name, "assets"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory %q: %w", dir, err)
		}
	}

	// Write forge.yaml. Use the base name of the path as the title.
	siteTitle := filepath.Base(name)
	configContent := fmt.Sprintf(`title: "%s"
baseURL: "http://localhost:1313"
language: "en"
theme: "default"

author:
  name: "Your Name"
  email: ""

menu:
  main:
    - name: "Home"
      url: "/"
      weight: 1
    - name: "Blog"
      url: "/blog/"
      weight: 2
    - name: "About"
      url: "/about/"
      weight: 3
`, siteTitle)

	configPath := filepath.Join(name, "forge.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		return fmt.Errorf("writing forge.yaml: %w", err)
	}

	// Write sample about page.
	now := nowFunc()
	aboutContent := fmt.Sprintf(`---
title: "About"
date: %s
layout: "page"
description: ""
---

Write your page content here.
`, now.Format(time.RFC3339))

	aboutPath := filepath.Join(name, "content", "pages", "about.md")
	if err := os.WriteFile(aboutPath, []byte(aboutContent), 0o644); err != nil {
		return fmt.Errorf("writing about.md: %w", err)
	}

	// Write sample blog post.
	datePrefix := now.Format("2006-01-02")
	postContent := fmt.Sprintf(`---
title: "Hello World"
date: %s
draft: true
tags: []
categories: []
description: ""
---

Write your post content here.
`, now.Format(time.RFC3339))

	postPath := filepath.Join(name, "content", "blog", datePrefix+"-hello-world.md")
	if err := os.WriteFile(postPath, []byte(postContent), 0o644); err != nil {
		return fmt.Errorf("writing hello-world.md: %w", err)
	}

	return nil
}

// NewPost creates a new blog post file at content/blog/YYYY-MM-DD-slug.md.
// Parent directories are created if they do not exist.
func NewPost(title string) error {
	now := nowFunc()
	slug := Slugify(title)
	datePrefix := now.Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.md", datePrefix, slug)

	dir := filepath.Join("content", "blog")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %q: %w", dir, err)
	}

	content := fmt.Sprintf(`---
title: "%s"
date: %s
draft: true
tags: []
categories: []
description: ""
---

Write your post content here.
`, title, now.Format(time.RFC3339))

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %q: %w", path, err)
	}

	return nil
}

// NewPage creates a new page file at content/pages/slug.md.
// Parent directories are created if they do not exist.
func NewPage(title string) error {
	now := nowFunc()
	slug := Slugify(title)
	filename := fmt.Sprintf("%s.md", slug)

	dir := filepath.Join("content", "pages")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %q: %w", dir, err)
	}

	content := fmt.Sprintf(`---
title: "%s"
date: %s
layout: "page"
description: ""
---

Write your page content here.
`, title, now.Format(time.RFC3339))

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %q: %w", path, err)
	}

	return nil
}

// NewProject creates a new project file at content/projects/slug.md.
// Parent directories are created if they do not exist.
func NewProject(title string) error {
	now := nowFunc()
	slug := Slugify(title)
	filename := fmt.Sprintf("%s.md", slug)

	dir := filepath.Join("content", "projects")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %q: %w", dir, err)
	}

	content := fmt.Sprintf(`---
title: "%s"
date: %s
draft: true
description: ""
params:
  tech: []
  github: ""
  demo: ""
---

Describe your project here.
`, title, now.Format(time.RFC3339))

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %q: %w", path, err)
	}

	return nil
}

// CreatedPostPath returns the file path that NewPost would create for the
// given title. This is useful for printing a success message after creation.
func CreatedPostPath(title string) string {
	now := nowFunc()
	slug := Slugify(title)
	datePrefix := now.Format("2006-01-02")
	return filepath.Join("content", "blog", fmt.Sprintf("%s-%s.md", datePrefix, slug))
}

// CreatedPagePath returns the file path that NewPage would create for the
// given title.
func CreatedPagePath(title string) string {
	slug := Slugify(title)
	return filepath.Join("content", "pages", fmt.Sprintf("%s.md", slug))
}

// CreatedProjectPath returns the file path that NewProject would create for
// the given title.
func CreatedProjectPath(title string) string {
	slug := Slugify(title)
	return filepath.Join("content", "projects", fmt.Sprintf("%s.md", slug))
}
