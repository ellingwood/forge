// Package scaffold provides functions for creating new sites, posts, pages,
// and projects in the Forge static site generator.
package scaffold

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

//go:embed seedimages
var seedImages embed.FS

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
// If themeFS is non-nil, theme files are extracted from it into themes/.
func NewSite(name string, themeFS fs.FS) error {
	// Check if directory already exists.
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("directory %q already exists", name)
	}

	// Create the directory structure.
	dirs := []string{
		filepath.Join(name, "content", "blog"),
		filepath.Join(name, "content", "projects"),
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

	aboutPath := filepath.Join(name, "content", "about.md")
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

	// Extract theme files from embedded FS if provided.
	if themeFS != nil {
		themeDst := filepath.Join(name, "themes")
		if err := extractFS(themeFS, "themes", themeDst); err != nil {
			return fmt.Errorf("extracting default theme: %w", err)
		}
	}

	return nil
}

// NewSiteSeeded creates a new site (like NewSite) and then pre-populates it
// with a kitchen-sink set of sample content so the full theme can be exercised
// immediately after running `forge serve`.
func NewSiteSeeded(name string, themeFS fs.FS) error {
	if err := NewSite(name, themeFS); err != nil {
		return err
	}

	now := nowFunc()

	type seedFile struct {
		path    string
		content string
	}

	files := []seedFile{
		// Homepage
		{
			path: filepath.Join(name, "content", "_index.md"),
			content: `---
title: "Welcome"
description: "A sample Forge site demonstrating all theme features."
---

Welcome to your new Forge site! This homepage was generated with **--seed** to give you a
full demo of the theme out of the box.

Browse the blog, check out the projects, or read the about page to get started.
`,
		},
		// Blog section listing
		{
			path: filepath.Join(name, "content", "blog", "_index.md"),
			content: `---
title: "Blog"
description: "Thoughts, tutorials, and notes."
---
`,
		},
		// About page (override the stub written by NewSite)
		{
			path: filepath.Join(name, "content", "about.md"),
			content: fmt.Sprintf(`---
title: "About"
date: %s
layout: "page"
description: "Learn more about this site and its author."
---

## Hello!

This is the about page, pre-populated by `+"`forge new site --seed`"+`.

I build things for the web. This site is generated with
[Forge](https://github.com/aellingwood/forge), a fast static site generator
written in Go.

Feel free to replace this content with your own story.
`, now.Format(time.RFC3339)),
		},
		// Blog posts 1–3 are page bundles (index.md + hero.png) written below.
		// Posts 4–5 remain as flat .md files to show both patterns coexist.
		{
			path: filepath.Join(name, "content", "blog", "2025-04-20-markdown-tips.md"),
			content: `---
title: "Markdown Tips for Technical Writers"
date: 2025-04-20T11:00:00Z
tags: ["web", "tutorial"]
categories: ["writing"]
description: "Make the most of Markdown when writing technical documentation."
---

Markdown is the lingua franca of technical writing. These tips will help you
produce cleaner, more readable documents.

## Use Headings Consistently

Start with ` + "`##`" + ` (H2) for top-level sections — reserve ` + "`#`" + ` (H1) for the document
title, which is usually injected by the template.

## Code Blocks

Always specify the language for syntax highlighting:

` + "```python" + `
def greet(name: str) -> str:
    return f"Hello, {name}!"
` + "```" + `

## Tables

| Feature       | Supported |
|---------------|-----------|
| Tables        | Yes       |
| Footnotes     | Yes       |
| Task lists    | Yes       |

## Linking

Prefer reference-style links for long URLs to keep prose readable:

` + "```" + `
Read the [Go specification][gospec].

[gospec]: https://go.dev/ref/spec
` + "```" + `
`,
		},
		{
			path: filepath.Join(name, "content", "blog", "2025-05-30-year-in-review.md"),
			content: `---
title: "Year in Review"
date: 2025-05-30T09:00:00Z
tags: ["devops"]
categories: ["personal"]
description: "Looking back at a year of building, shipping, and learning."
---

This year was defined by shipping. Five side projects, three blog posts a month,
and more than I can count in pull requests.

## What Worked

- **Writing in public** — Sharing work-in-progress attracted collaborators I would
  never have met otherwise.
- **Short feedback loops** — Daily deploys to production kept the codebase honest
  and bugs shallow.
- **Single-purpose tools** — Reaching for the simplest tool that could do the job
  led to less complexity and fewer late-night incidents.

## What Didn't

- **Premature optimization** — More than once I spent a week squeezing performance
  out of a system that wasn't the bottleneck.
- **Neglecting documentation** — Code that isn't documented is code waiting to
  surprise you six months later.

## Goals for Next Year

1. Publish an open-source library that other people actually use
2. Write a proper design document before starting any project > 1 week
3. Take at least one week completely off the computer
`,
		},
		// Projects
		{
			path: filepath.Join(name, "content", "projects", "forge.md"),
			content: `---
title: "Forge"
date: 2025-01-01T00:00:00Z
description: "A fast, opinionated static site generator written in Go."
params:
  tech: ["Go", "HTML", "Tailwind CSS"]
  github: "https://github.com/aellingwood/forge"
  demo: "https://example.com"
---

Forge is a static site generator that turns Markdown files into a complete website.
It renders pages in parallel, supports live reload during development, and generates
RSS/Atom feeds, a sitemap, and a search index automatically.

The theme system lets you customise every template while keeping the default design
as a starting point.
`,
		},
		{
			path: filepath.Join(name, "content", "projects", "go-cli-toolkit.md"),
			content: `---
title: "Go CLI Toolkit"
date: 2025-03-15T00:00:00Z
description: "A collection of utilities for building polished command-line tools in Go."
params:
  tech: ["Go"]
  github: "https://github.com/example/go-cli-toolkit"
  demo: ""
---

A set of small, composable packages for building CLI tools:

- **progress** — animated spinners and progress bars
- **table** — pretty-print tabular data with column alignment
- **prompt** — interactive prompts with validation
- **color** — ANSI colour helpers with automatic NO_COLOR detection

Each package has zero dependencies outside the standard library.
`,
		},
		// Skills data file
		{
			path: filepath.Join(name, "data", "skills.yaml"),
			content: `- category: "Languages"
  items: ["Go", "TypeScript", "Python", "SQL"]
- category: "Infrastructure"
  items: ["AWS", "Cloudflare", "Docker", "Terraform"]
- category: "Tools"
  items: ["Git", "Vim", "Postgres", "Redis"]
`,
		},
	}

	for _, f := range files {
		if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", f.path, err)
		}
		if err := os.WriteFile(f.path, []byte(f.content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", f.path, err)
		}
	}

	// Write the three page-bundle posts (index.md + hero.png each).
	type bundlePost struct {
		dir       string
		indexMd   string
		embedName string
	}

	bundles := []bundlePost{
		{
			dir: filepath.Join(name, "content", "blog", "2025-01-15-getting-started-with-go"),
			indexMd: `---
title: "Getting Started with Go"
date: 2025-01-15T09:00:00Z
tags: ["go", "tutorial"]
categories: ["programming"]
description: "A beginner-friendly introduction to the Go programming language."
cover:
  image: hero.png
  alt: "Blue banner for Getting Started with Go"
---

Go is a statically typed, compiled language designed at Google. It combines the
performance of C with the readability of Python, making it an excellent choice for
web services, CLI tools, and systems programming.

## Why Go?

Go's simplicity is its greatest strength. The language has a small specification,
a fast compiler, and a rich standard library. The built-in concurrency primitives
(goroutines and channels) make writing concurrent programs straightforward.

## Your First Program

` + "```go" + `
package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}
` + "```" + `

Save this to ` + "`main.go`" + ` and run ` + "`go run main.go`" + `. You should see:

` + "```" + `
Hello, world!
` + "```" + `

## Next Steps

- Read the [official tour](https://go.dev/tour/)
- Explore the [standard library](https://pkg.go.dev/std)
- Build something small: a CLI tool, a web API, or a file processor
`,
			embedName: "seedimages/hero-go.png",
		},
		{
			dir: filepath.Join(name, "content", "blog", "2025-02-10-building-static-sites"),
			indexMd: `---
title: "Building Static Sites with Forge"
date: 2025-02-10T10:00:00Z
tags: ["go", "web"]
categories: ["tools"]
description: "How Forge turns Markdown files into a fast static website."
cover:
  image: hero.png
  alt: "Indigo banner for Building Static Sites with Forge"
---

Static sites have made a comeback. They're fast, cheap to host, and easy to reason
about. Forge is a static site generator written in Go that aims to be simple,
fast, and opinionated.

## How It Works

Forge reads Markdown files from your ` + "`content/`" + ` directory, renders them with
your chosen theme, and writes the output to ` + "`public/`" + `. The result is a
directory of plain HTML files you can serve from any CDN or object storage bucket.

## Key Features

- **Fast builds** — Forge renders pages in parallel using all available CPU cores
- **Live reload** — ` + "`forge serve`" + ` watches for changes and reloads the browser
- **Taxonomies** — Tags and categories generate listing pages automatically
- **RSS & Atom feeds** — Generated automatically from your blog section
- **Search index** — A JSON search index is generated for client-side search

## Getting Started

` + "```sh" + `
forge new site mysite
cd mysite
forge serve
` + "```" + `

Open [http://localhost:1313](http://localhost:1313) and start writing.
`,
			embedName: "seedimages/hero-static-sites.png",
		},
		{
			dir: filepath.Join(name, "content", "blog", "2025-03-05-deploying-to-the-cloud"),
			indexMd: `---
title: "Deploying to the Cloud"
date: 2025-03-05T08:00:00Z
tags: ["devops", "tutorial"]
categories: ["infrastructure"]
description: "Ship your static site to S3, Cloudflare Pages, or any CDN."
cover:
  image: hero.png
  alt: "Teal banner for Deploying to the Cloud"
---

Once you've built your site with ` + "`forge build`" + `, deploying is just copying files.
Here are three common options.

## Amazon S3 + CloudFront

1. Create an S3 bucket with static website hosting enabled
2. Upload ` + "`public/`" + ` with ` + "`aws s3 sync public/ s3://your-bucket`" + `
3. Create a CloudFront distribution pointing at your bucket
4. Set a custom domain via Route 53

## Cloudflare Pages

Cloudflare Pages has a free tier and deploys directly from Git. Point it at your
repository, set the build command to ` + "`forge build`" + ` and the output directory to
` + "`public`" + `, and every push deploys automatically.

## Netlify

Like Cloudflare Pages, Netlify supports Git-based deployments. Add a
` + "`netlify.toml`" + `:

` + "```toml" + `
[build]
  command = "forge build"
  publish = "public"
` + "```" + `

## Tips

- Always set ` + "`baseURL`" + ` in ` + "`forge.yaml`" + ` to your production domain before building
- Use ` + "`forge build --minify`" + ` to shrink HTML output
`,
			embedName: "seedimages/hero-cloud.png",
		},
	}

	for _, b := range bundles {
		if err := os.MkdirAll(b.dir, 0o755); err != nil {
			return fmt.Errorf("creating bundle directory %s: %w", b.dir, err)
		}
		indexPath := filepath.Join(b.dir, "index.md")
		if err := os.WriteFile(indexPath, []byte(b.indexMd), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", indexPath, err)
		}
		imgPath := filepath.Join(b.dir, "hero.png")
		if err := writeSeedImage(imgPath, b.embedName); err != nil {
			return fmt.Errorf("writing hero image for %s: %w", b.dir, err)
		}
	}

	return nil
}

// writeSeedImage copies an embedded seed image to dest on disk.
func writeSeedImage(dest, embedPath string) error {
	data, err := seedImages.ReadFile(embedPath)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}

// extractFS copies all files from srcDir within src into dstDir on disk.
func extractFS(src fs.FS, srcDir, dstDir string) error {
	return fs.WalkDir(src, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		data, err := fs.ReadFile(src, path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
}

// RefreshTheme re-extracts theme files from the embedded FS into the site's
// themes/ directory, overwriting existing files. This brings the on-disk theme
// in sync with the version embedded in the current binary.
func RefreshTheme(siteRoot string, themeFS fs.FS) error {
	themeDst := filepath.Join(siteRoot, "themes")
	return extractFS(themeFS, "themes", themeDst)
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
