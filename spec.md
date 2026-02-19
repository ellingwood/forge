# Forge — Static Site Generator

> A fast, opinionated static site generator written in Go. Markdown in, beautiful HTML out.

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Architecture](#2-architecture)
3. [Directory Structure](#3-directory-structure)
4. [Content Model](#4-content-model)
5. [Templating Engine](#5-templating-engine)
6. [Styling — Tailwind CSS + shadcn Design Tokens](#6-styling)
7. [Feature Specifications](#7-feature-specifications)
8. [CLI Interface](#8-cli-interface)
9. [Build Pipeline](#9-build-pipeline)
10. [Deployment — S3 + CloudFront](#10-deployment)
11. [Default Theme — Portfolio + Blog](#11-default-theme)
12. [Testing Strategy](#12-testing-strategy)
13. [Dependencies](#13-dependencies)
14. [Non-Goals](#14-non-goals)
15. [Implementation Phases](#15-implementation-phases)

---

## 1. Project Overview

### 1.1 What Is Forge?

Forge is a Hugo-inspired static site generator (SSG) built from scratch in Go. It transforms a directory of Markdown content, Go HTML templates, and static assets into a deployable static website. It ships with a default theme designed for a personal portfolio + about + blog site.

### 1.2 Design Principles

- **Fast builds.** Leverage Go's concurrency. Parallel content processing, concurrent template rendering, and efficient file I/O. Target sub-second builds for sites under 500 pages.
- **Convention over configuration.** Sensible defaults that work out of the box. Zero-config for the common case, YAML/TOML overrides for everything else.
- **Single binary.** Ship as one compiled Go binary with no runtime dependencies. Tailwind CSS is compiled at build time and embedded.
- **Readable output.** Generate clean, semantic HTML. No JavaScript required for core functionality (search is the exception). Every page loads fast on a 3G connection.
- **Reproducible builds.** Same input always produces the same output. No timestamps or random values in generated files unless explicitly configured.

### 1.3 Name Rationale

"Forge" — raw materials (Markdown, templates, assets) go in, finished artifacts (a website) come out. The CLI reads naturally: `forge build`, `forge serve`, `forge new post "My Article"`.

---

## 2. Architecture

### 2.1 High-Level Pipeline

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Content      │────▶│  Processing  │────▶│  Rendering   │────▶│  Output      │
│  Discovery    │     │  Pipeline    │     │  Engine      │     │  Writer      │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
  - Walk fs            - Parse front       - Execute Go         - Write HTML
  - Glob patterns        matter              templates          - Copy assets
  - Collect            - Markdown → HTML   - Inject content     - Generate
    content files      - Build taxonomy   - Apply layout          sitemap/RSS
                       - Sort/filter        hierarchy           - Build search
                       - Syntax highlight                         index
```

### 2.2 Core Packages

```
forge/
├── cmd/                    # CLI entry points (cobra)
│   └── forge/
│       └── main.go
├── internal/
│   ├── config/             # Site configuration (YAML/TOML parsing)
│   ├── content/            # Content discovery, frontmatter parsing, Markdown rendering
│   │   ├── frontmatter.go  # YAML/TOML frontmatter parser
│   │   ├── markdown.go     # goldmark-based Markdown → HTML
│   │   ├── page.go         # Page model (content + metadata)
│   │   └── taxonomy.go     # Tags, categories, custom taxonomies
│   ├── template/           # Go html/template wrapper with custom functions
│   ├── render/             # Orchestrates template execution with content data
│   ├── build/              # Build pipeline coordinator (parallel processing)
│   ├── server/             # Dev server with live reload (WebSocket-based)
│   ├── search/             # Client-side search index generator (JSON)
│   ├── feed/               # RSS/Atom feed generation
│   ├── seo/                # Sitemap, robots.txt, OpenGraph/meta tag generation
│   ├── deploy/             # S3 sync + CloudFront invalidation
│   └── scaffold/           # `forge new` scaffolding logic
├── themes/
│   └── default/            # Ships with the binary (embedded via go:embed)
│       ├── layouts/
│       ├── static/
│       └── theme.yaml
└── embedded/               # go:embed assets (Tailwind CLI binary, default theme)
```

### 2.3 Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Markdown parser | [goldmark](https://github.com/yuin/goldmark) | Pure Go, fast, extensible, CommonMark compliant, Hugo uses it |
| Frontmatter format | YAML (default), TOML supported | YAML is more widely known; TOML for parity with Hugo |
| Template engine | Go `html/template` | Standard library, secure by default, no dependencies |
| CSS framework | Tailwind CSS (standalone CLI) | No Node.js runtime needed; use the standalone binary |
| Syntax highlighting | [chroma](https://github.com/alecthomas/chroma) via goldmark-highlighting | Pure Go, supports 200+ languages, multiple themes |
| CLI framework | [cobra](https://github.com/spf13/cobra) | Industry standard for Go CLIs |
| Config parser | [viper](https://github.com/spf13/viper) | Multi-format (YAML/TOML), env var support, pairs with cobra |
| Live reload | WebSocket | No dependency on external tools; inject small JS snippet in dev mode only |
| Search | [Fuse.js](https://www.fusejs.io/) + pre-built JSON index | Lightweight client-side fuzzy search, no server needed |

---

## 3. Directory Structure

### 3.1 Project Layout (User's Site)

```
my-site/
├── forge.yaml              # Site configuration
├── content/
│   ├── _index.md           # Homepage content
│   ├── about.md            # About page (portfolio content)
│   ├── blog/
│   │   ├── _index.md       # Blog listing page config
│   │   ├── my-first-post.md
│   │   └── another-post/
│   │       ├── index.md    # Page bundle (post + co-located assets)
│   │       └── diagram.png
│   └── projects/
│       ├── _index.md       # Projects listing page config
│       ├── project-one.md
│       └── project-two.md
├── layouts/                # User overrides (optional, overlays theme)
├── static/                 # Copied verbatim to output (images, favicons, etc.)
├── assets/                 # Processed assets (CSS source, etc.)
│   └── css/
│       └── custom.css      # User's custom Tailwind CSS (optional)
├── data/                   # Structured data files (YAML/JSON/TOML)
│   └── resume.yaml         # Accessible in templates as .Site.Data.resume
└── public/                 # Build output (gitignored)
```

### 3.2 Output Structure

```
public/
├── index.html
├── about/index.html
├── blog/
│   ├── index.html          # Blog listing (paginated)
│   ├── page/2/index.html   # Pagination
│   ├── my-first-post/index.html
│   └── another-post/
│       ├── index.html
│       └── diagram.png     # Co-located asset copied alongside
├── projects/
│   ├── index.html
│   ├── project-one/index.html
│   └── project-two/index.html
├── tags/
│   ├── index.html          # All tags listing
│   └── go/index.html       # Posts tagged "go"
├── categories/
│   ├── index.html
│   └── devops/index.html
├── css/
│   └── style.css           # Compiled + purged Tailwind CSS
├── js/
│   ├── search.js           # Search functionality
│   └── theme.js            # Dark/light mode toggle
├── search-index.json       # Pre-built search index
├── sitemap.xml
├── robots.txt
├── rss.xml                 # RSS 2.0 feed
└── atom.xml                # Atom feed
```

---

## 4. Content Model

### 4.1 Frontmatter Schema

Every content file begins with YAML frontmatter delimited by `---`. All fields are optional except `title`.

```yaml
---
title: "Building Resilient Kubernetes Clusters"  # Required
date: 2025-01-15T10:00:00Z                       # Publish date (ISO 8601)
lastmod: 2025-02-01T14:30:00Z                     # Last modified (auto-set from git if omitted)
draft: false                                       # Excluded from production builds
slug: "resilient-k8s-clusters"                     # URL override (default: filename slug)
description: "A deep dive into..."                 # Meta description / OpenGraph
summary: "Short summary for listing pages"         # Explicit summary (otherwise auto-generated)
tags: ["kubernetes", "devops", "reliability"]       # Taxonomy: tags
categories: ["Infrastructure"]                     # Taxonomy: categories
series: "Kubernetes Deep Dive"                     # Group related posts into a series
weight: 10                                         # Sort order for non-date ordering
cover:
  image: "cover.jpg"                               # Relative to page bundle or /static
  alt: "Kubernetes cluster diagram"
  caption: "Architecture overview"
author: "Austin"                                   # Override site-level default
layout: "post"                                     # Explicit layout override
aliases:                                           # Redirect old URLs to this page
  - /old-path/k8s-post
  - /archive/2024/k8s
params:                                            # Arbitrary key-value pairs for templates
  math: true
  toc: true
---
```

### 4.2 Page Types

| Type | Description | Source | URL Pattern |
|------|-------------|--------|-------------|
| **Single** | Individual content page | `content/blog/my-post.md` | `/blog/my-post/` |
| **List** | Section listing page | `content/blog/_index.md` | `/blog/` |
| **Taxonomy** | Auto-generated for each tag/category | (auto) | `/tags/go/` |
| **Taxonomy List** | Lists all terms in a taxonomy | (auto) | `/tags/` |
| **Home** | Site homepage | `content/_index.md` | `/` |

### 4.3 Content Organization

**Sections** are defined by top-level directories under `content/`. Each section can have its own `_index.md` with frontmatter controlling the listing page.

**Page Bundles** are directories containing an `index.md` plus co-located assets. Assets within page bundles are copied alongside the page's HTML output, enabling relative image references in Markdown that work in both source and rendered output.

### 4.4 Markdown Processing

Forge uses goldmark with the following extensions enabled by default:

- **GFM (GitHub Flavored Markdown):** Tables, strikethrough, autolinks, task lists
- **Footnotes:** `[^1]` syntax
- **Typographer:** Smart quotes, em/en dashes, ellipses
- **Syntax highlighting:** Fenced code blocks with language annotation via chroma
- **Heading IDs:** Auto-generated anchor IDs for headings (e.g., `## My Section` → `id="my-section"`)
- **Table of Contents:** Generated from headings when `params.toc: true` in frontmatter
- **Attributes:** Apply CSS classes/IDs to elements via `{.class #id}` syntax

### 4.5 Summary Generation

If `summary` is not set in frontmatter, Forge auto-generates it using:
1. Content before a `<!--more-->` marker (if present)
2. First paragraph of content (stripped of HTML)
3. Truncated to 160 characters for meta descriptions

### 4.6 Draft / Future / Expired Content

- `draft: true` — excluded from production builds, included with `forge serve` or `forge build --drafts`
- `date` in the future — excluded from production builds, included with `--future`
- `expiryDate` in the past — always excluded from production, included with `--expired`

---

## 5. Templating Engine

### 5.1 Layout Resolution Order

Forge resolves templates using a lookup order (first match wins):

```
For a page at content/blog/my-post.md with layout: "post":

1. layouts/blog/post.html          # Section-specific, layout-specific
2. layouts/blog/single.html        # Section-specific, default single
3. layouts/_default/post.html      # Default, layout-specific
4. layouts/_default/single.html    # Default fallback

For a list page at content/blog/_index.md:

1. layouts/blog/list.html
2. layouts/_default/list.html

For the homepage (content/_index.md):

1. layouts/index.html
2. layouts/_default/list.html
```

User layouts in the site's `layouts/` directory take precedence over theme layouts.

### 5.2 Base Template + Blocks

All layouts extend a base template using Go's `block` system:

```html
<!-- layouts/_default/baseof.html -->
<!DOCTYPE html>
<html lang="{{ .Site.Language }}" class="scroll-smooth">
<head>
  {{ block "head" . }}
    {{ partial "head/meta.html" . }}
    {{ partial "head/css.html" . }}
  {{ end }}
</head>
<body class="bg-background text-foreground font-sans antialiased">
  {{ partial "header.html" . }}
  <main>
    {{ block "main" . }}{{ end }}
  </main>
  {{ partial "footer.html" . }}
  {{ block "scripts" . }}
    {{ partial "scripts.html" . }}
  {{ end }}
</body>
</html>
```

```html
<!-- layouts/_default/single.html -->
{{ define "main" }}
<article class="prose mx-auto max-w-3xl px-4 py-12">
  <h1>{{ .Title }}</h1>
  <time datetime="{{ .Date.Format "2006-01-02" }}">{{ .Date.Format "January 2, 2006" }}</time>
  {{ .Content }}
</article>
{{ end }}
```

### 5.3 Partials

Reusable template fragments stored in `layouts/partials/`:

```
layouts/partials/
├── head/
│   ├── meta.html       # OpenGraph, Twitter Card, canonical URL
│   └── css.html        # Stylesheet links
├── header.html         # Site nav
├── footer.html         # Site footer
├── scripts.html        # JS includes (theme toggle, search, live reload in dev)
├── post-card.html      # Blog post preview card
├── project-card.html   # Project showcase card
├── pagination.html     # Prev/next page navigation
├── toc.html            # Table of contents
├── tags.html           # Tag pills/badges
└── search-modal.html   # Search overlay
```

### 5.4 Template Functions

In addition to Go's built-in template functions, Forge provides:

**String Functions:**
- `markdownify` — render inline Markdown to HTML
- `plainify` — strip HTML tags
- `truncate N` — truncate string to N characters with ellipsis
- `slugify` — convert to URL-safe slug
- `highlight CODE LANG` — syntax highlight a code string
- `safeHTML` — mark string as safe (no escaping)

**Collection Functions:**
- `where COLLECTION KEY VALUE` — filter a collection
- `sort COLLECTION KEY` — sort by field
- `first N COLLECTION` — take first N items
- `last N COLLECTION` — take last N items
- `shuffle COLLECTION` — randomize order
- `group COLLECTION KEY` — group items by field

**Date Functions:**
- `dateFormat FORMAT DATE` — format a date
- `now` — current time (build time)
- `readingTime CONTENT` — estimated reading time in minutes

**URL/Path Functions:**
- `relURL PATH` — site-relative URL
- `absURL PATH` — absolute URL with base URL
- `ref PAGE` — get permalink for a content page by path

**Data Functions:**
- `getJSON URL` — fetch and parse JSON (build time only)
- `readFile PATH` — read a file's contents

### 5.5 Template Data Context

Every template receives a data context (`.`) with these fields:

```go
type PageContext struct {
    // Page-level
    Title       string
    Description string
    Content     template.HTML    // Rendered Markdown
    Summary     template.HTML
    Date        time.Time
    Lastmod     time.Time
    Draft       bool
    Slug        string
    URL         string           // Relative permalink
    Permalink   string           // Absolute permalink
    ReadingTime int              // Minutes
    WordCount   int
    Tags        []string
    Categories  []string
    Series      string
    Params      map[string]any   // Arbitrary frontmatter params
    Cover       *CoverImage
    TableOfContents template.HTML
    PrevPage    *PageContext     // Previous page in section (by date)
    NextPage    *PageContext     // Next page in section (by date)

    // Site-level
    Site        SiteContext
}

type SiteContext struct {
    Title       string
    Description string
    BaseURL     string
    Language    string
    Author      AuthorConfig
    Menu        []MenuItem
    Params      map[string]any
    Data        map[string]any   // Parsed YAML/JSON from /data directory
    Pages       []*PageContext   // All pages
    Sections    map[string][]*PageContext
    Taxonomies  map[string]map[string][]*PageContext // e.g., .Site.Taxonomies.tags.go → []Page
    BuildDate   time.Time
}
```

---

## 6. Styling — Tailwind CSS + shadcn Design Tokens <a id="6-styling"></a>

### 6.1 Approach

Since shadcn/ui is a React component library, Forge adopts its **design system** without React:

- **CSS custom properties** from shadcn's theming system (HSL-based color tokens)
- **Tailwind CSS** configured to use those custom properties
- **Component patterns** translated to pure HTML + Tailwind utility classes

This gives the site shadcn's polished aesthetic while remaining a zero-JS static site.

### 6.2 CSS Custom Properties (Design Tokens)

```css
/* assets/css/globals.css */
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  :root {
    --background: 0 0% 100%;
    --foreground: 240 10% 3.9%;
    --card: 0 0% 100%;
    --card-foreground: 240 10% 3.9%;
    --popover: 0 0% 100%;
    --popover-foreground: 240 10% 3.9%;
    --primary: 240 5.9% 10%;
    --primary-foreground: 0 0% 98%;
    --secondary: 240 4.8% 95.9%;
    --secondary-foreground: 240 5.9% 10%;
    --muted: 240 4.8% 95.9%;
    --muted-foreground: 240 3.8% 46.1%;
    --accent: 240 4.8% 95.9%;
    --accent-foreground: 240 5.9% 10%;
    --destructive: 0 84.2% 60.2%;
    --destructive-foreground: 0 0% 98%;
    --border: 240 5.9% 90%;
    --input: 240 5.9% 90%;
    --ring: 240 5.9% 10%;
    --radius: 0.5rem;
  }

  .dark {
    --background: 240 10% 3.9%;
    --foreground: 0 0% 98%;
    --card: 240 10% 3.9%;
    --card-foreground: 0 0% 98%;
    --popover: 240 10% 3.9%;
    --popover-foreground: 0 0% 98%;
    --primary: 0 0% 98%;
    --primary-foreground: 240 5.9% 10%;
    --secondary: 240 3.7% 15.9%;
    --secondary-foreground: 0 0% 98%;
    --muted: 240 3.7% 15.9%;
    --muted-foreground: 240 5% 64.9%;
    --accent: 240 3.7% 15.9%;
    --accent-foreground: 0 0% 98%;
    --destructive: 0 62.8% 30.6%;
    --destructive-foreground: 0 0% 98%;
    --border: 240 3.7% 15.9%;
    --input: 240 3.7% 15.9%;
    --ring: 240 4.9% 83.9%;
  }
}
```

### 6.3 Tailwind Configuration

```js
// tailwind.config.js (used by Tailwind standalone CLI at build time)
module.exports = {
  content: ["./layouts/**/*.html", "./content/**/*.md"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      fontFamily: {
        sans: ["Inter", "system-ui", "-apple-system", "sans-serif"],
        mono: ["JetBrains Mono", "Fira Code", "monospace"],
      },
      typography: ({ theme }) => ({
        DEFAULT: {
          css: {
            "--tw-prose-body": theme("colors.foreground"),
            "--tw-prose-headings": theme("colors.foreground"),
            "--tw-prose-links": theme("colors.primary.DEFAULT"),
            "--tw-prose-code": theme("colors.foreground"),
            maxWidth: "none",
          },
        },
      }),
    },
  },
  plugins: [require("@tailwindcss/typography")],
};
```

### 6.4 shadcn-Inspired Component Classes

Forge ships reusable CSS component classes that mirror shadcn's component patterns:

```css
@layer components {
  .card {
    @apply rounded-lg border border-border bg-card text-card-foreground shadow-sm;
  }
  .badge {
    @apply inline-flex items-center rounded-full border border-border px-2.5 py-0.5 text-xs
           font-semibold transition-colors hover:bg-accent hover:text-accent-foreground;
  }
  .btn-primary {
    @apply inline-flex items-center justify-center rounded-md bg-primary px-4 py-2
           text-sm font-medium text-primary-foreground shadow-sm transition-colors
           hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2
           focus-visible:ring-ring;
  }
  .btn-outline {
    @apply inline-flex items-center justify-center rounded-md border border-border
           bg-background px-4 py-2 text-sm font-medium shadow-sm transition-colors
           hover:bg-accent hover:text-accent-foreground focus-visible:outline-none
           focus-visible:ring-2 focus-visible:ring-ring;
  }
  .input {
    @apply flex h-10 w-full rounded-md border border-input bg-background px-3 py-2
           text-sm ring-offset-background placeholder:text-muted-foreground
           focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring;
  }
  .separator {
    @apply h-px w-full bg-border;
  }
}
```

### 6.5 Tailwind Build Integration

Forge downloads and caches the [Tailwind CSS standalone CLI](https://tailwindcss.com/blog/standalone-cli) binary appropriate for the host OS/arch. This avoids requiring Node.js.

- **Dev mode:** `tailwindcss --watch` runs alongside the dev server for instant CSS rebuilds
- **Production:** `tailwindcss --minify` produces a purged, minified CSS file

The Tailwind binary path is cached in `~/.forge/bin/tailwindcss`.

---

## 7. Feature Specifications

### 7.1 Dev Server with Live Reload

**Implementation:**
- Embedded HTTP server using Go's `net/http` (no external framework needed)
- Serves the `public/` directory with appropriate MIME types
- WebSocket endpoint at `/__forge/ws` pushes reload events
- File watcher (using `fsnotify`) monitors `content/`, `layouts/`, `static/`, `assets/`
- On file change: rebuild affected pages (incremental if possible), notify WebSocket clients
- In dev mode, inject a small `<script>` before `</body>` that connects to the WebSocket

**Dev-only behaviors:**
- Include draft content
- Include future-dated content
- Skip CSS minification (faster rebuilds)
- Display build errors as an overlay in the browser

**Configuration:**
```yaml
# forge.yaml
server:
  port: 1313
  host: "localhost"
  livereload: true
```

### 7.2 RSS/Atom Feed Generation

**Feeds generated:**
- `/rss.xml` — RSS 2.0 feed (all published blog posts, sorted by date desc, limit 20)
- `/atom.xml` — Atom 1.0 feed (same content)
- Per-section feeds: `/blog/rss.xml`, `/projects/rss.xml`
- Per-taxonomy feeds: `/tags/go/rss.xml`

**Feed content includes:**
- Full article HTML content (not just summary)
- Author information
- Publication date, last modified date
- Categories/tags as RSS categories
- Cover image as enclosure (if present)

**Template overrides:** Users can provide `layouts/_default/rss.xml` and `layouts/_default/atom.xml` to customize feed output.

**Configuration:**
```yaml
# forge.yaml
feeds:
  rss: true
  atom: true
  limit: 20          # Max items per feed
  fullContent: true   # Include full content vs. summary only
  sections:           # Which sections get their own feeds
    - blog
```

### 7.3 SEO Optimization

**Sitemap (`/sitemap.xml`):**
- Auto-generated from all non-draft pages
- Includes `<lastmod>`, `<changefreq>`, `<priority>`
- Priority: homepage=1.0, sections=0.8, posts=0.6 (configurable per-page via frontmatter)
- Generates sitemap index if >50,000 URLs

**Robots.txt (`/robots.txt`):**
- Generated from config or user-provided `static/robots.txt`
- Default allows all crawlers, references sitemap

**OpenGraph + Twitter Cards:**
- Auto-generated `<meta>` tags in `<head>` from frontmatter
- `og:title`, `og:description`, `og:image`, `og:url`, `og:type`
- `twitter:card`, `twitter:title`, `twitter:description`, `twitter:image`
- `<link rel="canonical">` with absolute URL
- JSON-LD structured data for articles (schema.org/Article)

**Configuration:**
```yaml
# forge.yaml
seo:
  titleTemplate: "%s | My Site"   # %s replaced with page title
  defaultImage: "/images/og-default.jpg"
  twitterHandle: "@myhandle"
  jsonLD: true                    # Enable JSON-LD structured data
```

### 7.4 Syntax Highlighting

**Engine:** chroma (via goldmark-highlighting extension)

**Features:**
- 200+ language support
- Line numbers (optional, per-block via `{linenos=true}`)
- Line highlighting: `{hl_lines=[1, "3-5"]}` 
- Multiple themes: configurable light/dark themes that switch with the site theme
- CSS class-based rendering (not inline styles) for theme compatibility

**Markdown usage:**
````markdown
```go {linenos=true, hl_lines=[3]}
func main() {
    fmt.Println("Hello")
    fmt.Println("Highlighted line")
}
```
````

**Configuration:**
```yaml
# forge.yaml
highlight:
  style: "github"           # chroma style name
  darkStyle: "github-dark"  # Used when dark mode is active
  lineNumbers: false         # Default; overridable per-block
  tabWidth: 4
```

**Implementation detail:** Two separate CSS stylesheets are generated at build time (one per theme). The dark mode stylesheet is scoped under `.dark` to match the theme toggle.

### 7.5 Taxonomy Pages

**Built-in taxonomies:** `tags`, `categories`

**Custom taxonomies** can be defined in config:
```yaml
# forge.yaml
taxonomies:
  tag: tags
  category: categories
  series: series         # Custom: groups posts into multi-part series
```

**Generated pages:**
- `/tags/` — lists all tags with post counts, sorted by count desc
- `/tags/go/` — lists all posts tagged "go", paginated
- Same pattern for categories, series, or any custom taxonomy

**Template data for taxonomy pages:**
```go
type TaxonomyContext struct {
    Term     string           // e.g., "go"
    Pages    []*PageContext   // All pages with this term
    Count    int
}

type TaxonomyListContext struct {
    Terms    []TaxonomyContext // All terms, sorted by count
    Taxonomy string            // e.g., "tags"
}
```

### 7.6 Search Functionality

**Architecture:** Build-time index generation + client-side search via Fuse.js.

**Build step:**
1. For each published page, extract: title, URL, tags, categories, summary, and a plaintext excerpt (first 5000 chars, stripped of HTML)
2. Write to `/search-index.json`

**Client-side:**
- Fuse.js loaded via CDN (`<script>` tag, deferred)
- Search UI: a `<dialog>` modal triggered by a search icon in the nav or `Cmd/Ctrl+K` keyboard shortcut
- Searches title, tags, summary, and content excerpt
- Displays results as a list of linked titles with highlighted matches and summary snippets
- This is the **only** JavaScript required for the site (besides the theme toggle)

**Search index JSON schema:**
```json
[
  {
    "title": "Building Resilient Kubernetes Clusters",
    "url": "/blog/resilient-k8s-clusters/",
    "tags": ["kubernetes", "devops"],
    "categories": ["Infrastructure"],
    "summary": "A deep dive into...",
    "content": "Plaintext excerpt of the article..."
  }
]
```

**Configuration:**
```yaml
# forge.yaml
search:
  enabled: true
  contentLength: 5000    # Max chars of content in index per page
  keys:                  # Fuse.js search keys and weights
    - { name: "title", weight: 2.0 }
    - { name: "tags", weight: 1.5 }
    - { name: "summary", weight: 1.0 }
    - { name: "content", weight: 0.5 }
```

### 7.7 Dark/Light Theme Toggle

**Implementation:**
- CSS: shadcn design tokens already define light (`:root`) and dark (`.dark`) color schemes
- JS: Minimal script (~20 lines) that:
  1. Checks `localStorage` for saved preference
  2. Falls back to `prefers-color-scheme` media query
  3. Adds/removes `.dark` class on `<html>`
  4. Persists choice to `localStorage`
- Toggle button in the site header (sun/moon icon, smooth transition)
- Script is inlined in `<head>` (before render) to prevent flash of wrong theme (FOWT)

**No-JS fallback:** Respects `prefers-color-scheme` via CSS media query. Toggle button hidden when JS is disabled via `<noscript>` style.

---

## 8. CLI Interface

### 8.1 Commands

```bash
# Create a new site
forge new site my-site
# Creates my-site/ with full directory structure, default theme, and sample content

# Create new content
forge new post "My First Post"
# Creates content/blog/YYYY-MM-DD-my-first-post.md with frontmatter template

forge new page "About Me"
# Creates content/about-me.md

forge new project "My Project"
# Creates content/projects/my-project.md

# Build the site
forge build
# Builds to public/ with production settings (minified CSS, no drafts)

forge build --drafts         # Include draft content
forge build --future         # Include future-dated content
forge build --baseURL "..."  # Override base URL
forge build -d ./dist        # Custom output directory
forge build --verbose        # Detailed build log with timing

# Development server
forge serve
# Starts dev server at http://localhost:1313 with live reload

forge serve --port 8080      # Custom port
forge serve --bind 0.0.0.0   # Bind to all interfaces
forge serve --no-live-reload  # Disable live reload

# Deploy to S3 + CloudFront
forge deploy
# Syncs public/ to configured S3 bucket + invalidates CloudFront

forge deploy --dry-run       # Show what would change without deploying

# Utilities
forge version                # Print version info
forge config                 # Print resolved configuration
forge list drafts            # List all draft content
forge list future            # List all future-dated content
forge list expired           # List all expired content
```

### 8.2 Scaffolding Templates

`forge new post` generates:

```yaml
---
title: "My First Post"
date: 2025-01-15T10:00:00-07:00
draft: true
description: ""
summary: ""
tags: []
categories: []
cover:
  image: ""
  alt: ""
params:
  toc: false
---

Write your content here...
```

---

## 9. Build Pipeline

### 9.1 Build Steps (Ordered)

```
1. Load Configuration
   └─ Parse forge.yaml, merge with defaults, validate

2. Discover Content
   └─ Walk content/ directory
   └─ Parse frontmatter for every .md file
   └─ Build page collection with metadata

3. Process Content (parallel)
   └─ Render Markdown → HTML (goldmark + chroma)
   └─ Generate summaries where missing
   └─ Calculate reading time, word count

4. Build Taxonomy Maps
   └─ Index pages by tags, categories, custom taxonomies
   └─ Generate taxonomy and taxonomy list page contexts

5. Sort & Paginate
   └─ Sort pages by date (desc) within each section
   └─ Generate pagination contexts (configurable page size)

6. Render Templates (parallel)
   └─ Resolve layout for each page
   └─ Execute Go templates with page context
   └─ Write HTML to public/

7. Generate Ancillary Files
   └─ sitemap.xml
   └─ robots.txt
   └─ rss.xml, atom.xml (global + per-section)
   └─ search-index.json
   └─ aliases (HTML redirect pages)

8. Process Assets
   └─ Run Tailwind CSS CLI (purge + minify in production)
   └─ Copy static/ to public/
   └─ Copy page bundle assets alongside their pages

9. Report
   └─ Print build summary: pages rendered, time elapsed, output size
```

### 9.2 Incremental Builds (Dev Mode)

In dev mode, Forge tracks which files changed and only rebuilds affected pages:

- **Content file changed** → Re-render that page + any list pages in its section + taxonomy pages for its terms
- **Layout/partial changed** → Re-render all pages using that layout (or all pages if baseof changed)
- **Static file changed** → Copy just that file
- **Config changed** → Full rebuild
- **CSS source changed** → Re-run Tailwind CLI

### 9.3 Parallelism

Forge uses a worker pool pattern for CPU-bound steps:

```go
// Pseudocode
numWorkers := runtime.NumCPU()
jobs := make(chan *Page, len(pages))
results := make(chan *RenderedPage, len(pages))

for i := 0; i < numWorkers; i++ {
    go renderWorker(jobs, results, templateEngine)
}
for _, page := range pages {
    jobs <- page
}
close(jobs)
for i := 0; i < len(pages); i++ {
    rendered := <-results
    // write to disk
}
```

### 9.4 Build Configuration

```yaml
# forge.yaml — full example with defaults shown
baseURL: "https://example.com"
title: "My Site"
description: "Personal portfolio and blog"
language: "en"
theme: "default"               # Use embedded default theme

author:
  name: "Austin"
  email: "austin@example.com"
  bio: "Cloud engineer."
  avatar: "/images/avatar.jpg"
  social:
    github: "username"
    linkedin: "username"
    twitter: "username"

menu:
  main:
    - name: "Home"
      url: "/"
      weight: 1
    - name: "Blog"
      url: "/blog/"
      weight: 2
    - name: "Projects"
      url: "/projects/"
      weight: 3
    - name: "About"
      url: "/about/"
      weight: 4

pagination:
  pageSize: 10

taxonomies:
  tag: tags
  category: categories

highlight:
  style: "github"
  darkStyle: "github-dark"
  lineNumbers: false

search:
  enabled: true

feeds:
  rss: true
  atom: true
  limit: 20

seo:
  titleTemplate: "%s | My Site"
  jsonLD: true

server:
  port: 1313
  livereload: true

build:
  minify: true              # Minify CSS in production
  cleanUrls: true           # /blog/post/ instead of /blog/post.html

deploy:
  s3:
    bucket: "my-site-bucket"
    region: "us-west-2"
  cloudfront:
    distributionId: "E1234567890"
    invalidatePaths:
      - "/*"
```

---

## 10. Deployment — S3 + CloudFront <a id="10-deployment"></a>

### 10.1 `forge deploy` Command

**Sync strategy:** Uses content-hash-based diffing to minimize uploads.

```
1. Hash every file in public/ (SHA-256)
2. List objects in S3 bucket (with ETags)
3. Compute diff:
   - New files → upload
   - Changed files (hash mismatch) → upload
   - Deleted files → delete from S3
   - Unchanged → skip
4. Upload with appropriate Content-Type and Cache-Control headers
5. Invalidate CloudFront distribution
```

### 10.2 Upload Headers

| File Type | Content-Type | Cache-Control |
|-----------|-------------|---------------|
| `.html` | `text/html; charset=utf-8` | `public, max-age=0, must-revalidate` |
| `.css` | `text/css` | `public, max-age=31536000, immutable` |
| `.js` | `application/javascript` | `public, max-age=31536000, immutable` |
| `.json` | `application/json` | `public, max-age=0, must-revalidate` |
| `.xml` | `application/xml` | `public, max-age=3600` |
| Images | auto-detected | `public, max-age=31536000, immutable` |
| Other | auto-detected | `public, max-age=86400` |

**Note:** CSS and JS files should use content-hash-based filenames (e.g., `style.abc123.css`) to enable immutable caching. HTML files are never cached so new content deploys instantly.

### 10.3 CloudFront Invalidation

After upload, Forge creates an invalidation for `/*` (or configured paths). It waits for the invalidation to complete before reporting success (with a configurable timeout).

### 10.4 AWS Credential Resolution

Uses the standard AWS SDK credential chain: environment variables → `~/.aws/credentials` → IAM role → EC2 instance metadata. No credentials are stored in `forge.yaml`.

### 10.5 Required IAM Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::my-site-bucket",
        "arn:aws:s3:::my-site-bucket/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "cloudfront:CreateInvalidation",
        "cloudfront:GetInvalidation"
      ],
      "Resource": "arn:aws:cloudfront::ACCOUNT_ID:distribution/DISTRIBUTION_ID"
    }
  ]
}
```

---

## 11. Default Theme — Portfolio + Blog <a id="11-default-theme"></a>

### 11.1 Pages

**Homepage (`/`):**
- Hero section: name, title/tagline, short bio, avatar, social links
- Featured projects section (3-4 cards, pulled from content/projects with `featured: true`)
- Recent blog posts (3-5 latest, pulled from content/blog)
- Call-to-action or contact prompt

**About (`/about/`):**
- Extended bio / professional narrative
- Skills/technologies section (grouped by category, rendered from `data/skills.yaml`)
- Work experience timeline (rendered from `data/experience.yaml`)
- Education (optional, from `data/education.yaml`)

**Blog Listing (`/blog/`):**
- Paginated list of post cards (title, date, summary, tags, reading time, cover image)
- Filterable by tag (client-side via simple JS show/hide, not a SPA)

**Blog Post (`/blog/post-slug/`):**
- Article header: title, date, reading time, tags
- Optional cover image with caption
- Table of contents (optional, sticky sidebar on desktop)
- Rendered Markdown content with Tailwind Typography (`prose`) styling
- Previous/Next post navigation
- Series navigation (if part of a series)

**Projects Listing (`/projects/`):**
- Grid of project cards (title, description, tech stack badges, links)
- Optional filtering by technology

**Project Detail (`/projects/project-slug/`):**
- Project writeup (Markdown)
- Links to live demo, source code
- Tech stack badges

**Taxonomy Pages:**
- `/tags/` — all tags as badge pills with post counts
- `/tags/[tag]/` — paginated post list for that tag
- Same for categories

### 11.2 Design Language

- **Typography:** Inter for body, JetBrains Mono for code. Clean hierarchy with clear visual weight.
- **Spacing:** Generous whitespace. Content max-width of `max-w-3xl` (768px) for readability.
- **Colors:** shadcn neutral palette (zinc-based). Subtle, professional. Accent color configurable.
- **Cards:** Subtle border + shadow, hover effect (slight lift or border color change).
- **Navigation:** Clean top nav with site title left, menu items right. Collapses to hamburger on mobile.
- **Footer:** Minimal — copyright, social links, RSS link, "Built with Forge" (optional).
- **Responsive:** Mobile-first. Breakpoints at `sm`, `md`, `lg`. Blog sidebar (TOC) collapses to accordion on mobile.
- **Animations:** Minimal and tasteful — hover transitions, page load fade-in (CSS-only, no JS animation libs).
- **Accessibility:** Semantic HTML, ARIA labels, focus-visible styles, skip-to-content link, sufficient color contrast (WCAG AA).

### 11.3 Data Files for Portfolio Content

```yaml
# data/skills.yaml
- category: "Cloud & Infrastructure"
  items: ["AWS", "GCP", "Kubernetes", "Terraform", "Docker"]
- category: "Languages"
  items: ["Go", "Python", "TypeScript", "Bash"]
- category: "Tools"
  items: ["Git", "GitHub Actions", "ArgoCD", "Prometheus"]
```

```yaml
# data/experience.yaml
- title: "Senior Cloud Engineer"
  company: "Example Corp"
  period: "2022 – Present"
  description: "Led migration to Kubernetes..."
  highlights:
    - "Reduced deployment time by 80%"
    - "Managed 200+ microservices"
```

---

## 12. Testing Strategy

### 12.1 Unit Tests

- **Frontmatter parsing:** Valid/invalid YAML, TOML, missing required fields, type coercion
- **Markdown rendering:** GFM features, code blocks, footnotes, heading IDs, TOC generation
- **Template functions:** Each custom function with edge cases
- **Taxonomy building:** Correct grouping, counting, sorting
- **Summary generation:** `<!--more-->` marker, first-paragraph extraction, truncation
- **Slug generation:** Unicode handling, special characters, deduplication
- **Config merging:** Defaults + file + CLI flags

### 12.2 Integration Tests

- **Full build pipeline:** Build a test site fixture, assert correct output structure and content
- **Dev server:** Start server, modify a file, assert WebSocket reload event fires
- **Incremental rebuild:** Change one content file, assert only affected pages re-rendered
- **Feed validation:** Validate RSS/Atom output against schemas
- **Sitemap validation:** Validate against sitemap XML schema

### 12.3 Golden File Tests

Maintain a `testdata/` directory with:
- Input Markdown files with various frontmatter configurations
- Expected HTML output
- Tests compare rendered output against golden files, updated with `go test -update`

### 12.4 Benchmarks

- Benchmark build time for 10, 100, 1000, and 5000 page sites
- Benchmark Markdown rendering throughput
- Benchmark template execution throughput

---

## 13. Dependencies

### 13.1 Go Module Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/spf13/viper` | Configuration management |
| `github.com/yuin/goldmark` | Markdown → HTML |
| `github.com/yuin/goldmark-highlighting/v2` | Syntax highlighting (chroma) |
| `github.com/alecthomas/chroma/v2` | Syntax highlighting engine |
| `github.com/yuin/goldmark-meta` | Frontmatter parsing |
| `go.abhg.dev/goldmark/toc` | Table of contents generation |
| `github.com/fsnotify/fsnotify` | File system watching |
| `github.com/aws/aws-sdk-go-v2` | S3 + CloudFront deployment |
| `github.com/gorilla/websocket` | WebSocket for live reload |
| `github.com/tdewolff/minify/v2` | HTML/CSS/JS minification (optional) |
| `gopkg.in/yaml.v3` | YAML parsing |
| `github.com/BurntSushi/toml` | TOML parsing |

### 13.2 External Tools (Managed by Forge)

| Tool | Purpose | Delivery |
|------|---------|----------|
| Tailwind CSS standalone CLI | CSS compilation | Auto-downloaded to `~/.forge/bin/` |

### 13.3 Client-Side Dependencies (CDN)

| Library | Purpose | Size |
|---------|---------|------|
| Fuse.js | Client-side search | ~7KB gzip |

---

## 14. Non-Goals

These are explicitly out of scope for the initial version:

- **Image optimization pipeline** — users can pre-optimize images. May add later.
- **Asset fingerprinting / bundling** — CSS is handled by Tailwind CLI. No JS bundler.
- **i18n / multilingual** — single-language sites only (for now).
- **CMS integration** — content is Markdown files in git. No headless CMS adapters.
- **Server-side rendering / dynamic content** — this is a static site generator.
- **Plugin system** — extend by forking or contributing upstream. Plugin API is a future consideration.
- **Shortcodes** — Hugo-style shortcodes. May add later if Go templates prove insufficient.
- **SCSS/LESS support** — Tailwind CSS is the only supported CSS framework.
- **React / hydration / islands architecture** — pure HTML + Tailwind only.

---

## 15. Implementation Phases

### Phase 1: Core Engine (MVP)

**Goal:** `forge build` produces a working static site from Markdown content.

- [ ] Project scaffolding (`forge new site`)
- [ ] Configuration loading (`forge.yaml` → viper)
- [ ] Content discovery and frontmatter parsing
- [ ] Markdown → HTML rendering (goldmark + chroma)
- [ ] Go `html/template` engine with layout resolution + partials
- [ ] Basic default theme (homepage, blog list, blog post, about page)
- [ ] Static file copying
- [ ] `forge build` command
- [ ] Page bundles (co-located assets)

### Phase 2: Developer Experience

**Goal:** Productive local development workflow.

- [ ] Dev server (`forge serve`) with live reload (WebSocket)
- [ ] File watcher with incremental rebuild
- [ ] Draft / future / expired content filtering
- [ ] `forge new post|page|project` scaffolding
- [ ] Tailwind CSS standalone CLI integration (auto-download + watch mode)
- [ ] Build timing and summary reporting

### Phase 3: Content Features

**Goal:** Full content management capabilities.

- [ ] Taxonomy system (tags, categories, custom taxonomies)
- [ ] Taxonomy pages (term pages + term list pages)
- [ ] Pagination (configurable page size)
- [ ] Series support (ordered multi-post collections)
- [ ] Table of contents generation
- [ ] Summary generation (`<!--more-->`, first paragraph, truncation)
- [ ] URL aliases / redirects
- [ ] Previous / Next page navigation
- [ ] Reading time + word count
- [ ] Data files (`data/` directory → `.Site.Data`)

### Phase 4: SEO + Feeds

**Goal:** Production-ready SEO and syndication.

- [ ] Sitemap generation (`sitemap.xml`)
- [ ] `robots.txt` generation
- [ ] OpenGraph + Twitter Card meta tags
- [ ] Canonical URLs
- [ ] JSON-LD structured data (schema.org/Article)
- [ ] RSS 2.0 feed generation (global + per-section)
- [ ] Atom feed generation
- [ ] SEO title template (`%s | Site Name`)

### Phase 5: Search + Theme Polish

**Goal:** Complete default theme with search.

- [ ] Search index generation (`search-index.json`)
- [ ] Search UI (dialog modal, Cmd+K, Fuse.js)
- [ ] Dark / light theme toggle
- [ ] Projects section (grid, detail pages, tech stack badges)
- [ ] Skills / experience / education data rendering
- [ ] Mobile responsive navigation (hamburger menu)
- [ ] Accessibility audit (ARIA, focus management, contrast)
- [ ] Final theme polish (animations, hover states, loading states)

### Phase 6: Deployment

**Goal:** One-command deploy to S3 + CloudFront.

- [ ] `forge deploy` — S3 sync with content-hash diffing
- [ ] CloudFront invalidation
- [ ] Dry-run mode
- [ ] CSS content-hash filenames for immutable caching
- [ ] Correct `Cache-Control` headers by file type
- [ ] Deploy summary (files uploaded, deleted, unchanged)

---

## Appendix A: Naming Alternatives

If "Forge" conflicts with an existing project:

- **Anvil** — `anvil build`, `anvil serve`
- **Smelt** — `smelt build`, `smelt serve`
- **Ingot** — `ingot build`, `ingot serve`
- **Alloy** — `alloy build`, `alloy serve`

---

## Appendix B: Future Considerations

These may be added in later versions:

- **Shortcodes** — Hugo-style template snippets callable from Markdown
- **Image pipeline** — resize, convert to WebP, generate srcset
- **Asset fingerprinting** — content-hash all static assets
- **Plugin API** — Go plugin system or WASM-based extension points
- **i18n** — multilingual content with language switcher
- **Comments** — integration with Giscus, Utterances, or similar
- **Analytics** — optional, privacy-respecting (Plausible, Umami)
- **Import tool** — migrate content from Hugo, Jekyll, WordPress
