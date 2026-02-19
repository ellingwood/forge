# Forge MCP Server — Specification

> Model Context Protocol server for the Forge static site generator, enabling AI-assisted site development via Claude Code and other MCP clients.

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture](#2-architecture)
3. [Transport & Lifecycle](#3-transport--lifecycle)
4. [Resources](#4-resources)
5. [Tools](#5-tools)
6. [Prompts](#6-prompts)
7. [Notifications](#7-notifications)
8. [Implementation Details](#8-implementation-details)
9. [Configuration](#9-configuration)
10. [Testing Strategy](#10-testing-strategy)
11. [Claude Code Integration](#11-claude-code-integration)
12. [Dependencies](#12-dependencies)
13. [Implementation Phases](#13-implementation-phases)

---

## 1. Overview

### 1.1 Purpose

The Forge MCP server exposes Forge's content graph, build system, and site configuration as structured, queryable data to MCP clients. This enables AI-powered development workflows where Claude Code (or any MCP-compliant client) can:

- Understand the site's content model without parsing files
- Query content by taxonomy, date, section, draft status — with structured results
- Create and validate content with correct frontmatter schemas
- Trigger builds and surface errors programmatically
- Introspect template data contexts for debugging layout issues
- Deploy sites with structured feedback

Without MCP, an AI agent working on a Forge site must `grep`, `cat`, and infer structure from raw files. With MCP, it gets semantic access to the site as a first-class data graph.

### 1.2 Design Principles

- **Read-heavy surface area.** Most operations are reads (queries, introspection). Writes are limited to content scaffolding and build/deploy triggers. Forge's filesystem is the source of truth — the MCP server never bypasses it.
- **Leverage existing internals.** The MCP server is a thin layer over Forge's existing `config`, `content`, `template`, `build`, `taxonomy`, `feed`, and `deploy` packages. No duplicated logic.
- **Structured over raw.** Every response returns typed JSON. Claude Code should never need to parse HTML or regex through Markdown to get metadata.
- **Idempotent and safe.** Read tools are pure queries. Write tools either scaffold files (which can be overwritten) or trigger existing CLI operations (`build`, `deploy`). Nothing destructive.

### 1.3 MCP Specification Version

This server targets **MCP spec 2025-06-18** (stable) with forward compatibility for 2025-11-25 features. The implementation uses the official Go SDK `github.com/modelcontextprotocol/go-sdk/mcp` (v1.2.0+).

---

## 2. Architecture

### 2.1 Server Position in Forge

```
┌─────────────────────────────────────────────────────────────┐
│ Forge Binary                                                │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  │
│  │ cmd/     │  │ cmd/     │  │ cmd/     │  │ cmd/       │  │
│  │ build    │  │ serve    │  │ deploy   │  │ mcp        │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └─────┬──────┘  │
│       │              │              │              │         │
│       ▼              ▼              ▼              ▼         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Shared Internal Packages                │   │
│  │  config / content / template / build / taxonomy /    │   │
│  │  feed / seo / search / deploy / server               │   │
│  └──────────────────────────────────────────────────────┘   │
│                          ▲                                  │
│                          │                                  │
│                   ┌──────┴──────┐                           │
│                   │ internal/   │                           │
│                   │ mcpserver/  │  ◄── NEW: MCP layer       │
│                   └─────────────┘                           │
└─────────────────────────────────────────────────────────────┘
         ▲
         │ stdio (JSON-RPC 2.0)
         ▼
┌─────────────────┐
│  Claude Code    │
│  (MCP Client)   │
└─────────────────┘
```

### 2.2 Package Layout

```
internal/
└── mcpserver/
    ├── server.go           # Server initialization, capability declaration
    ├── resources.go        # Resource handlers (config, pages, taxonomies, etc.)
    ├── tools.go            # Tool handlers (query, create, build, deploy, etc.)
    ├── prompts.go          # Prompt templates (blog post, project, review, etc.)
    ├── notifications.go    # Change notification logic
    ├── context.go          # Shared site context loader (lazy, cached)
    ├── types.go            # MCP response types (JSON-serializable structs)
    └── server_test.go      # Integration tests
```

### 2.3 Site Context Lifecycle

The MCP server maintains a `SiteContext` — a cached, in-memory representation of the site's content graph. This is the same data structure that the build pipeline produces after steps 1–5 (config → discovery → processing → taxonomy → sort/paginate).

```go
// context.go
type SiteContext struct {
    mu       sync.RWMutex
    config   *config.SiteConfig
    pages    []*content.Page
    sections map[string][]*content.Page
    taxos    map[string]map[string][]*content.Page
    buildAt  time.Time
    dirty    bool
}
```

**Loading strategy:**
- On first resource/tool access, perform a full content load (equivalent to build steps 1–5, without rendering)
- Cache the result in memory
- On `fsnotify` file change events (if running alongside `forge serve`), mark context as dirty
- On next access after dirty, reload. Reloads are fast since they skip rendering.

---

## 3. Transport & Lifecycle

### 3.1 Transport: stdio

The primary transport is **stdio** (stdin/stdout JSON-RPC 2.0), which is the standard for local MCP servers used by Claude Code.

```go
// cmd/forge/mcp.go
func runMCP(cmd *cobra.Command, args []string) error {
    siteDir, _ := cmd.Flags().GetString("source")
    srv := mcpserver.New(siteDir)
    return srv.Run(context.Background(), &mcp.StdioTransport{})
}
```

**CLI invocation:**
```bash
forge mcp                    # Run MCP server over stdio (current directory)
forge mcp --source /path/to  # Specify site root
```

### 3.2 Server Capabilities

Declared during MCP `initialize` handshake:

```go
server := mcp.NewServer(
    &mcp.Implementation{
        Name:    "forge",
        Version: version.String(), // e.g., "0.1.0"
    },
    &mcp.ServerOptions{
        Capabilities: &mcp.ServerCapabilities{
            Resources: &mcp.ResourceCapabilities{
                ListChanged: ptr(true), // Notify on content changes
            },
            Tools: &mcp.ToolCapabilities{
                ListChanged: ptr(false), // Tool set is static
            },
            Prompts: &mcp.PromptCapabilities{
                ListChanged: ptr(false), // Prompt set is static
            },
        },
    },
)
```

### 3.3 Lifecycle

1. Client (Claude Code) spawns `forge mcp` as a subprocess
2. `initialize` handshake — server declares capabilities, client declares roots
3. Server lazy-loads site context on first request
4. Server handles requests until client disconnects (stdin EOF)
5. Server exits cleanly

---

## 4. Resources

Resources expose read-only site data that MCP clients can pull into their context window. Each resource has a stable URI and returns structured JSON.

### 4.1 Resource List

| URI | Name | Description |
|-----|------|-------------|
| `forge://config` | Site Configuration | Resolved site configuration (merged defaults + forge.yaml) |
| `forge://content/pages` | Content Inventory | All pages with metadata (no body content) |
| `forge://content/page/{path}` | Page Detail | Full page detail including raw Markdown and rendered HTML |
| `forge://content/sections` | Sections | All content sections with page counts |
| `forge://taxonomies` | Taxonomies Overview | All taxonomies with their terms and counts |
| `forge://taxonomies/{name}` | Taxonomy Detail | All terms and associated pages for a specific taxonomy |
| `forge://templates` | Template Inventory | All available layouts and partials with file paths |
| `forge://build/status` | Build Status | Last build result — timestamp, duration, errors, warnings |
| `forge://schema/frontmatter` | Frontmatter Schema | Valid frontmatter fields, types, defaults, and constraints |

### 4.2 Resource Specifications

#### `forge://config`

Returns the fully resolved site configuration after merging defaults with `forge.yaml`.

```json
{
  "baseURL": "https://example.com",
  "title": "My Site",
  "description": "Personal portfolio and blog",
  "language": "en",
  "theme": "default",
  "author": {
    "name": "Austin",
    "email": "austin@example.com",
    "social": {
      "github": "username",
      "linkedin": "username"
    }
  },
  "menu": {
    "main": [
      { "name": "Home", "url": "/", "weight": 1 },
      { "name": "Blog", "url": "/blog/", "weight": 2 }
    ]
  },
  "taxonomies": {
    "tag": "tags",
    "category": "categories"
  },
  "pagination": { "pageSize": 10 },
  "search": { "enabled": true },
  "feeds": { "rss": true, "atom": true, "limit": 20 },
  "build": { "minify": true, "cleanUrls": true },
  "deploy": {
    "s3": { "bucket": "my-site-bucket", "region": "us-west-2" },
    "cloudfront": { "distributionId": "E1234567890" }
  }
}
```

#### `forge://content/pages`

Returns a lightweight inventory of all content pages. Includes metadata only — no body content. This is the primary discovery resource for understanding what content exists.

```json
{
  "totalPages": 42,
  "pages": [
    {
      "path": "content/blog/resilient-k8s-clusters.md",
      "url": "/blog/resilient-k8s-clusters/",
      "title": "Building Resilient Kubernetes Clusters",
      "date": "2025-01-15T10:00:00Z",
      "lastmod": "2025-02-01T14:30:00Z",
      "draft": false,
      "section": "blog",
      "tags": ["kubernetes", "devops", "reliability"],
      "categories": ["Infrastructure"],
      "series": "Kubernetes Deep Dive",
      "summary": "A deep dive into building resilient...",
      "readingTime": 12,
      "wordCount": 2847,
      "hasCover": true,
      "isPageBundle": false
    }
  ]
}
```

#### `forge://content/page/{path}`

Returns full detail for a single page, including raw Markdown source and rendered HTML. The `{path}` parameter is the relative path from the site root (e.g., `content/blog/my-post.md`).

**Resource template URI:** `forge://content/page/{path}`

```json
{
  "path": "content/blog/resilient-k8s-clusters.md",
  "url": "/blog/resilient-k8s-clusters/",
  "title": "Building Resilient Kubernetes Clusters",
  "date": "2025-01-15T10:00:00Z",
  "lastmod": "2025-02-01T14:30:00Z",
  "draft": false,
  "section": "blog",
  "slug": "resilient-k8s-clusters",
  "description": "A deep dive into...",
  "summary": "A deep dive into building resilient...",
  "tags": ["kubernetes", "devops", "reliability"],
  "categories": ["Infrastructure"],
  "series": "Kubernetes Deep Dive",
  "weight": 0,
  "cover": {
    "image": "cover.jpg",
    "alt": "Kubernetes cluster diagram",
    "caption": "Architecture overview"
  },
  "params": {
    "toc": true,
    "math": false
  },
  "aliases": [],
  "readingTime": 12,
  "wordCount": 2847,
  "rawMarkdown": "---\ntitle: \"Building Resilient...\"...\n---\n\n## Introduction\n\n...",
  "renderedHTML": "<h2 id=\"introduction\">Introduction</h2>\n<p>...</p>",
  "tableOfContents": "<nav class=\"toc\">...</nav>",
  "bundleAssets": ["cover.jpg", "diagram.png"],
  "prevPage": { "title": "Previous Post", "url": "/blog/prev/" },
  "nextPage": { "title": "Next Post", "url": "/blog/next/" }
}
```

#### `forge://content/sections`

Returns all content sections with page counts and metadata.

```json
{
  "sections": [
    {
      "name": "blog",
      "path": "content/blog/",
      "pageCount": 25,
      "draftCount": 3,
      "hasIndex": true,
      "indexTitle": "Blog",
      "latestDate": "2025-02-15T10:00:00Z",
      "oldestDate": "2024-03-01T09:00:00Z"
    },
    {
      "name": "projects",
      "path": "content/projects/",
      "pageCount": 8,
      "draftCount": 1,
      "hasIndex": true,
      "indexTitle": "Projects",
      "latestDate": "2025-01-20T10:00:00Z",
      "oldestDate": "2024-06-15T09:00:00Z"
    }
  ]
}
```

#### `forge://taxonomies`

Returns an overview of all configured taxonomies with term counts.

```json
{
  "taxonomies": [
    {
      "name": "tags",
      "singular": "tag",
      "urlBase": "/tags/",
      "termCount": 18,
      "totalAssignments": 87,
      "terms": [
        { "name": "kubernetes", "slug": "kubernetes", "count": 12 },
        { "name": "go", "slug": "go", "count": 9 },
        { "name": "devops", "slug": "devops", "count": 8 }
      ]
    },
    {
      "name": "categories",
      "singular": "category",
      "urlBase": "/categories/",
      "termCount": 5,
      "totalAssignments": 42,
      "terms": [
        { "name": "Infrastructure", "slug": "infrastructure", "count": 15 },
        { "name": "Programming", "slug": "programming", "count": 12 }
      ]
    }
  ]
}
```

#### `forge://taxonomies/{name}`

Returns full detail for a specific taxonomy, including all terms with their associated page references.

```json
{
  "name": "tags",
  "singular": "tag",
  "urlBase": "/tags/",
  "terms": [
    {
      "name": "kubernetes",
      "slug": "kubernetes",
      "url": "/tags/kubernetes/",
      "count": 12,
      "pages": [
        {
          "title": "Building Resilient Kubernetes Clusters",
          "url": "/blog/resilient-k8s-clusters/",
          "date": "2025-01-15T10:00:00Z",
          "section": "blog"
        }
      ]
    }
  ]
}
```

#### `forge://templates`

Returns an inventory of all templates (layouts + partials) available in both the theme and user overrides.

```json
{
  "layouts": [
    {
      "path": "layouts/_default/baseof.html",
      "source": "theme",
      "type": "base"
    },
    {
      "path": "layouts/_default/single.html",
      "source": "theme",
      "type": "single",
      "overriddenBy": null
    },
    {
      "path": "layouts/blog/single.html",
      "source": "user",
      "type": "single",
      "section": "blog"
    }
  ],
  "partials": [
    {
      "path": "layouts/partials/header.html",
      "source": "theme",
      "overriddenBy": null
    },
    {
      "path": "layouts/partials/post-card.html",
      "source": "user",
      "overrides": "theme"
    }
  ]
}
```

#### `forge://build/status`

Returns information about the last build. Returns `null` fields if no build has been performed.

```json
{
  "lastBuild": {
    "timestamp": "2025-02-15T14:30:00Z",
    "durationMs": 847,
    "success": true,
    "pagesRendered": 42,
    "outputDir": "public/",
    "outputSizeBytes": 2457600,
    "errors": [],
    "warnings": [
      {
        "file": "content/blog/old-post.md",
        "message": "cover image 'missing.jpg' not found",
        "level": "warning"
      }
    ]
  },
  "devServerRunning": false,
  "tailwindInstalled": true,
  "tailwindVersion": "3.4.1"
}
```

#### `forge://schema/frontmatter`

Returns the frontmatter schema — all recognized fields, their types, defaults, and validation constraints. This enables Claude Code to generate valid frontmatter without guessing.

```json
{
  "required": ["title"],
  "fields": {
    "title": {
      "type": "string",
      "description": "Page title (required)",
      "default": null
    },
    "date": {
      "type": "datetime",
      "description": "Publish date (ISO 8601)",
      "default": "now"
    },
    "draft": {
      "type": "boolean",
      "description": "Exclude from production builds",
      "default": true
    },
    "tags": {
      "type": "[]string",
      "description": "Tag taxonomy terms",
      "default": [],
      "existingValues": ["kubernetes", "go", "devops", "python", "aws"]
    },
    "categories": {
      "type": "[]string",
      "description": "Category taxonomy terms",
      "default": [],
      "existingValues": ["Infrastructure", "Programming", "DevOps"]
    },
    "series": {
      "type": "string",
      "description": "Group related posts into a named series",
      "default": null,
      "existingValues": ["Kubernetes Deep Dive", "Go Patterns"]
    },
    "cover": {
      "type": "object",
      "description": "Cover image configuration",
      "fields": {
        "image": { "type": "string" },
        "alt": { "type": "string" },
        "caption": { "type": "string" }
      }
    },
    "slug": {
      "type": "string",
      "description": "URL slug override (default: derived from filename)"
    },
    "description": {
      "type": "string",
      "description": "Meta description / OpenGraph description"
    },
    "summary": {
      "type": "string",
      "description": "Explicit summary for listing pages"
    },
    "weight": {
      "type": "integer",
      "description": "Sort order for non-date ordering",
      "default": 0
    },
    "layout": {
      "type": "string",
      "description": "Explicit layout override",
      "validValues": ["post", "project", "page"]
    },
    "aliases": {
      "type": "[]string",
      "description": "Redirect old URLs to this page"
    },
    "params": {
      "type": "map[string]any",
      "description": "Arbitrary key-value pairs accessible in templates",
      "knownKeys": {
        "toc": { "type": "boolean", "default": false },
        "math": { "type": "boolean", "default": false }
      }
    }
  }
}
```

---

## 5. Tools

Tools are actions that the MCP client can invoke. Each tool has a defined input schema (JSON Schema) and returns structured results.

### 5.1 Tool List

| Tool | Category | Description |
|------|----------|-------------|
| `query_content` | Read | Filter and sort content pages by arbitrary criteria |
| `get_page` | Read | Get full detail for a single page by path or URL |
| `list_drafts` | Read | List all draft content across all sections |
| `validate_frontmatter` | Read | Validate a frontmatter YAML string against the schema |
| `get_template_context` | Read | Show what data a specific template receives at render time |
| `resolve_layout` | Read | Show which layout file a given content page will use |
| `create_content` | Write | Scaffold a new content file with valid frontmatter |
| `build_site` | Write | Trigger a full site build and return the result |
| `deploy_site` | Write | Deploy the site to S3 + CloudFront |

### 5.2 Tool Specifications

#### `query_content`

**Description:** Query the site's content graph with structured filters. Supports filtering by section, taxonomy terms, date ranges, draft status, and arbitrary frontmatter fields. Results are paginated and sorted.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "section": {
      "type": "string",
      "description": "Filter by content section (e.g., 'blog', 'projects')"
    },
    "tags": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Filter pages that have ALL of these tags"
    },
    "categories": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Filter pages that have ANY of these categories"
    },
    "draft": {
      "type": "boolean",
      "description": "Filter by draft status. Omit to include all."
    },
    "dateAfter": {
      "type": "string",
      "format": "date-time",
      "description": "Only pages published after this date (ISO 8601)"
    },
    "dateBefore": {
      "type": "string",
      "format": "date-time",
      "description": "Only pages published before this date (ISO 8601)"
    },
    "series": {
      "type": "string",
      "description": "Filter by series name"
    },
    "search": {
      "type": "string",
      "description": "Full-text search across title, summary, and content"
    },
    "sortBy": {
      "type": "string",
      "enum": ["date", "title", "weight", "readingTime", "wordCount"],
      "description": "Sort field (default: 'date')"
    },
    "sortOrder": {
      "type": "string",
      "enum": ["asc", "desc"],
      "description": "Sort order (default: 'desc')"
    },
    "limit": {
      "type": "integer",
      "minimum": 1,
      "maximum": 100,
      "description": "Max results to return (default: 20)"
    },
    "offset": {
      "type": "integer",
      "minimum": 0,
      "description": "Pagination offset (default: 0)"
    }
  },
  "additionalProperties": false
}
```

**Output:** Same shape as `forge://content/pages` but filtered/sorted, with pagination metadata:
```json
{
  "totalMatches": 12,
  "offset": 0,
  "limit": 20,
  "pages": [ ... ]
}
```

---

#### `get_page`

**Description:** Get full detail for a single page. Accepts either a content file path or a URL slug.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "Content file path relative to site root (e.g., 'content/blog/my-post.md')"
    },
    "url": {
      "type": "string",
      "description": "Page URL (e.g., '/blog/my-post/'). Alternative to path."
    }
  },
  "oneOf": [
    { "required": ["path"] },
    { "required": ["url"] }
  ]
}
```

**Output:** Same shape as `forge://content/page/{path}` resource.

---

#### `list_drafts`

**Description:** List all draft content across all sections. A convenience wrapper around `query_content` with `draft: true`.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "section": {
      "type": "string",
      "description": "Optionally filter drafts by section"
    }
  },
  "additionalProperties": false
}
```

**Output:**
```json
{
  "totalDrafts": 5,
  "drafts": [
    {
      "path": "content/blog/wip-post.md",
      "title": "Work in Progress",
      "section": "blog",
      "date": "2025-02-10T10:00:00Z",
      "tags": ["go"],
      "wordCount": 450
    }
  ]
}
```

---

#### `validate_frontmatter`

**Description:** Validate a YAML frontmatter string against Forge's schema. Returns errors and warnings. Useful for Claude Code to pre-validate content before writing files.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "frontmatter": {
      "type": "string",
      "description": "Raw YAML frontmatter (without --- delimiters)"
    },
    "section": {
      "type": "string",
      "description": "Target section (affects layout validation)"
    }
  },
  "required": ["frontmatter"]
}
```

**Output:**
```json
{
  "valid": false,
  "errors": [
    {
      "field": "date",
      "message": "Invalid date format: expected ISO 8601",
      "value": "January 15, 2025"
    }
  ],
  "warnings": [
    {
      "field": "tags",
      "message": "Tag 'k8s' is similar to existing tag 'kubernetes'. Did you mean 'kubernetes'?",
      "suggestion": "kubernetes"
    },
    {
      "field": "categories",
      "message": "Category 'Infra' does not exist. Existing categories: Infrastructure, Programming, DevOps",
      "suggestion": "Infrastructure"
    }
  ],
  "normalizedFrontmatter": "title: \"My Post\"\ndate: ...\n"
}
```

**Key behavior:** The similarity detection for tags/categories uses Levenshtein distance to catch near-duplicates. This is the single highest-value feature of the MCP server — it prevents taxonomy fragmentation when Claude Code creates content.

---

#### `get_template_context`

**Description:** Show the full data context that a specific template receives during rendering. Invaluable for debugging layout issues — Claude Code can see exactly what variables are available.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "pagePath": {
      "type": "string",
      "description": "Content file path to simulate rendering for"
    },
    "templatePath": {
      "type": "string",
      "description": "Specific template to inspect (optional; auto-resolved if omitted)"
    }
  },
  "required": ["pagePath"]
}
```

**Output:**
```json
{
  "resolvedTemplate": "layouts/blog/single.html",
  "baseTemplate": "layouts/_default/baseof.html",
  "partials": ["header.html", "footer.html", "head/meta.html", "head/css.html", "toc.html"],
  "context": {
    "Title": "Building Resilient Kubernetes Clusters",
    "Content": "(rendered HTML, truncated to 500 chars)...",
    "Date": "2025-01-15T10:00:00Z",
    "Tags": ["kubernetes", "devops"],
    "ReadingTime": 12,
    "PrevPage": { "Title": "Previous Post", "URL": "/blog/prev/" },
    "NextPage": null,
    "Site": {
      "Title": "My Site",
      "BaseURL": "https://example.com",
      "Params": { "... (top-level site params)" },
      "Data": { "skills": "(...)", "experience": "(...)" },
      "Taxonomies": {
        "tags": { "kubernetes": "(12 pages)", "go": "(9 pages)" }
      }
    }
  },
  "availableFunctions": [
    "markdownify", "plainify", "truncate", "slugify", "highlight",
    "safeHTML", "where", "sort", "first", "last", "shuffle", "group",
    "dateFormat", "now", "readingTime", "relURL", "absURL", "ref"
  ]
}
```

---

#### `resolve_layout`

**Description:** Show which layout file a given content page will use, including the full resolution chain.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "pagePath": {
      "type": "string",
      "description": "Content file path (e.g., 'content/blog/my-post.md')"
    }
  },
  "required": ["pagePath"]
}
```

**Output:**
```json
{
  "resolved": "layouts/blog/single.html",
  "source": "theme",
  "lookupOrder": [
    { "path": "layouts/blog/post.html", "exists": false, "source": "user" },
    { "path": "layouts/blog/post.html", "exists": false, "source": "theme" },
    { "path": "layouts/blog/single.html", "exists": false, "source": "user" },
    { "path": "layouts/blog/single.html", "exists": true, "source": "theme" },
    { "path": "layouts/_default/post.html", "exists": false, "source": "user" },
    { "path": "layouts/_default/post.html", "exists": false, "source": "theme" },
    { "path": "layouts/_default/single.html", "exists": true, "source": "theme" }
  ],
  "baseTemplate": "layouts/_default/baseof.html",
  "blocks": ["head", "main", "scripts"]
}
```

---

#### `create_content`

**Description:** Scaffold a new content file with valid frontmatter. More powerful than `forge new` because it accepts structured parameters and can set all frontmatter fields.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "type": {
      "type": "string",
      "enum": ["post", "page", "project"],
      "description": "Content type (determines section and default frontmatter)"
    },
    "title": {
      "type": "string",
      "description": "Page title"
    },
    "slug": {
      "type": "string",
      "description": "URL slug override (default: generated from title)"
    },
    "tags": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Tags to assign"
    },
    "categories": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Categories to assign"
    },
    "series": {
      "type": "string",
      "description": "Series name"
    },
    "draft": {
      "type": "boolean",
      "description": "Mark as draft (default: true)"
    },
    "description": {
      "type": "string",
      "description": "Meta description"
    },
    "body": {
      "type": "string",
      "description": "Initial Markdown body content (optional)"
    },
    "pageBundle": {
      "type": "boolean",
      "description": "Create as a page bundle directory (default: false)"
    },
    "params": {
      "type": "object",
      "description": "Additional frontmatter params (e.g., { \"toc\": true })"
    }
  },
  "required": ["type", "title"]
}
```

**Output:**
```json
{
  "created": true,
  "filePath": "content/blog/2025-02-18-building-go-cli-tools.md",
  "url": "/blog/building-go-cli-tools/",
  "frontmatter": "title: \"Building Go CLI Tools\"\ndate: 2025-02-18T10:00:00-07:00\ndraft: true\ntags:\n  - go\n  - cli\ncategories:\n  - Programming\nparams:\n  toc: false\n",
  "warnings": [
    "Tag 'cli' is new and will create a new taxonomy term"
  ]
}
```

**Key behaviors:**
- Validates tags/categories against existing terms and warns on new ones
- Generates date from current time
- Creates page bundle directory structure if `pageBundle: true`
- Does NOT overwrite existing files — returns an error if file already exists

---

#### `build_site`

**Description:** Trigger a full site build and return structured results. Equivalent to `forge build` but returns machine-readable output.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "drafts": {
      "type": "boolean",
      "description": "Include draft content (default: false)"
    },
    "future": {
      "type": "boolean",
      "description": "Include future-dated content (default: false)"
    },
    "baseURL": {
      "type": "string",
      "description": "Override base URL"
    },
    "outputDir": {
      "type": "string",
      "description": "Override output directory (default: 'public/')"
    },
    "verbose": {
      "type": "boolean",
      "description": "Include per-page timing in output (default: false)"
    }
  },
  "additionalProperties": false
}
```

**Output:**
```json
{
  "success": true,
  "durationMs": 847,
  "pagesRendered": 42,
  "staticFilesCopied": 15,
  "outputDir": "public/",
  "outputSizeBytes": 2457600,
  "errors": [],
  "warnings": [
    {
      "file": "content/blog/old-post.md",
      "message": "cover image 'missing.jpg' not found"
    }
  ],
  "generated": {
    "sitemap": true,
    "rss": true,
    "atom": true,
    "searchIndex": true,
    "robotsTxt": true
  }
}
```

---

#### `deploy_site`

**Description:** Deploy the built site to S3 + CloudFront. Requires a successful build first.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "dryRun": {
      "type": "boolean",
      "description": "Show what would change without deploying (default: false)"
    },
    "skipInvalidation": {
      "type": "boolean",
      "description": "Skip CloudFront cache invalidation (default: false)"
    }
  },
  "additionalProperties": false
}
```

**Output:**
```json
{
  "success": true,
  "dryRun": false,
  "bucket": "my-site-bucket",
  "region": "us-west-2",
  "filesUploaded": 8,
  "filesDeleted": 1,
  "filesUnchanged": 48,
  "bytesTransferred": 145200,
  "invalidation": {
    "distributionId": "E1234567890",
    "invalidationId": "I3EXAMPLE",
    "status": "Completed",
    "paths": ["/*"]
  }
}
```

---

## 6. Prompts

Prompts are reusable templates that help MCP clients generate well-structured requests. They guide Claude Code toward producing content that matches Forge's conventions.

### 6.1 Prompt List

| Prompt | Description |
|--------|-------------|
| `new_blog_post` | Generate a complete blog post with proper frontmatter and Markdown structure |
| `new_project` | Generate a project page with tech stack, links, and description |
| `content_review` | Review an existing content file for issues (SEO, formatting, metadata) |
| `site_overview` | Generate a summary of the site's current state for context |

### 6.2 Prompt Specifications

#### `new_blog_post`

**Arguments:**
```json
{
  "topic": {
    "type": "string",
    "description": "The topic or title of the blog post",
    "required": true
  },
  "tags": {
    "type": "string",
    "description": "Comma-separated tags (leave empty for AI to suggest)"
  },
  "audience": {
    "type": "string",
    "description": "Target audience (e.g., 'senior engineers', 'beginners')",
    "required": false
  }
}
```

**Generated Prompt Messages:**
```json
[
  {
    "role": "user",
    "content": {
      "type": "text",
      "text": "Write a blog post about: {{topic}}\n\nTarget audience: {{audience | default: 'experienced developers'}}\n\nUse the following frontmatter schema:\n{{embed forge://schema/frontmatter}}\n\nExisting tags in use on this site:\n{{embed forge://taxonomies/tags}}\n\nRequirements:\n- Use proper YAML frontmatter delimited by ---\n- Reuse existing tags where applicable to avoid taxonomy fragmentation\n- Include a description field (under 160 chars) for SEO\n- Use ## for top-level sections within the post\n- Include code examples where relevant with language annotations\n- End with a brief conclusion or summary\n- Set draft: true\n\nRequested tags: {{tags | default: '(suggest appropriate tags)'}}"
    }
  }
]
```

#### `new_project`

**Arguments:**
```json
{
  "name": {
    "type": "string",
    "description": "Project name",
    "required": true
  },
  "techStack": {
    "type": "string",
    "description": "Comma-separated technologies used"
  },
  "repoUrl": {
    "type": "string",
    "description": "Source code repository URL"
  }
}
```

#### `content_review`

**Arguments:**
```json
{
  "pagePath": {
    "type": "string",
    "description": "Path to the content file to review",
    "required": true
  }
}
```

**Generated Prompt Messages:**

Embeds the full page content via `forge://content/page/{pagePath}` and the frontmatter schema, then asks Claude to check for:
- Missing or suboptimal SEO fields (description, summary)
- Taxonomy consistency (tags matching existing terms)
- Markdown formatting issues
- Readability and structure
- Broken internal links

#### `site_overview`

**Arguments:** None.

**Generated Prompt Messages:**

Embeds `forge://config`, `forge://content/sections`, `forge://taxonomies`, and `forge://build/status` to give Claude Code a complete picture of the site's current state. This is the ideal "first prompt" when starting a session.

---

## 7. Notifications

### 7.1 Resource Change Notifications

When running alongside `forge serve` (or independently with file watching enabled), the MCP server sends `notifications/resources/list_changed` when:

- A content file is created, modified, or deleted
- `forge.yaml` is modified
- A template file is modified
- Files in `data/` are modified

This allows the MCP client to re-fetch stale resources.

**Implementation:** The MCP server's `SiteContext` registers an `fsnotify` watcher on the site root. On file change events, it marks the context as dirty and sends the notification.

### 7.2 Progress Notifications

Long-running tools (`build_site`, `deploy_site`) send progress notifications:

```json
{
  "method": "notifications/progress",
  "params": {
    "progressToken": "build-1234",
    "progress": 25,
    "total": 42,
    "message": "Rendering blog/my-post.md"
  }
}
```

---

## 8. Implementation Details

### 8.1 Server Initialization

```go
// internal/mcpserver/server.go

package mcpserver

import (
    "context"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

type ForgeServer struct {
    server  *mcp.Server
    siteDir string
    ctx     *SiteContext
}

func New(siteDir string) *ForgeServer {
    fs := &ForgeServer{
        siteDir: siteDir,
        ctx:     NewSiteContext(siteDir),
    }

    fs.server = mcp.NewServer(
        &mcp.Implementation{
            Name:    "forge",
            Version: version.String(),
        },
        &mcp.ServerOptions{
            Capabilities: &mcp.ServerCapabilities{
                Resources: &mcp.ResourceCapabilities{ListChanged: ptr(true)},
                Tools:     &mcp.ToolCapabilities{ListChanged: ptr(false)},
                Prompts:   &mcp.PromptCapabilities{ListChanged: ptr(false)},
            },
        },
    )

    fs.registerResources()
    fs.registerTools()
    fs.registerPrompts()

    return fs
}

func (fs *ForgeServer) Run(ctx context.Context, transport mcp.Transport) error {
    return fs.server.Run(ctx, transport)
}
```

### 8.2 Resource Registration Pattern

```go
// internal/mcpserver/resources.go

func (fs *ForgeServer) registerResources() {
    // Static resources
    mcp.AddResource(fs.server, &mcp.Resource{
        URI:         "forge://config",
        Name:        "Site Configuration",
        Description: ptr("Resolved site configuration from forge.yaml"),
        MIMEType:    ptr("application/json"),
    }, fs.handleConfigResource)

    mcp.AddResource(fs.server, &mcp.Resource{
        URI:         "forge://content/pages",
        Name:        "Content Inventory",
        Description: ptr("All content pages with metadata"),
        MIMEType:    ptr("application/json"),
    }, fs.handlePagesResource)

    // Resource templates (parameterized URIs)
    mcp.AddResourceTemplate(fs.server, &mcp.ResourceTemplate{
        URITemplate: "forge://content/page/{path}",
        Name:        "Page Detail",
        Description: ptr("Full detail for a single content page"),
        MIMEType:    ptr("application/json"),
    }, fs.handlePageDetailResource)

    mcp.AddResourceTemplate(fs.server, &mcp.ResourceTemplate{
        URITemplate: "forge://taxonomies/{name}",
        Name:        "Taxonomy Detail",
        Description: ptr("Terms and pages for a specific taxonomy"),
        MIMEType:    ptr("application/json"),
    }, fs.handleTaxonomyDetailResource)

    // ... remaining resources
}
```

### 8.3 Tool Registration Pattern

```go
// internal/mcpserver/tools.go

type QueryContentInput struct {
    Section    string   `json:"section,omitempty"`
    Tags       []string `json:"tags,omitempty"`
    Categories []string `json:"categories,omitempty"`
    Draft      *bool    `json:"draft,omitempty"`
    DateAfter  string   `json:"dateAfter,omitempty"`
    DateBefore string   `json:"dateBefore,omitempty"`
    Series     string   `json:"series,omitempty"`
    Search     string   `json:"search,omitempty"`
    SortBy     string   `json:"sortBy,omitempty"`
    SortOrder  string   `json:"sortOrder,omitempty"`
    Limit      int      `json:"limit,omitempty"`
    Offset     int      `json:"offset,omitempty"`
}

type QueryContentOutput struct {
    TotalMatches int         `json:"totalMatches"`
    Offset       int         `json:"offset"`
    Limit        int         `json:"limit"`
    Pages        []PageBrief `json:"pages"`
}

func (fs *ForgeServer) registerTools() {
    mcp.AddTool(fs.server,
        &mcp.Tool{
            Name:        "query_content",
            Description: "Filter and sort content pages by section, tags, date, draft status, and more",
        },
        fs.handleQueryContent,
    )

    mcp.AddTool(fs.server,
        &mcp.Tool{
            Name:        "create_content",
            Description: "Scaffold a new content file with valid frontmatter",
        },
        fs.handleCreateContent,
    )

    // ... remaining tools
}

func (fs *ForgeServer) handleQueryContent(
    ctx context.Context,
    req *mcp.CallToolRequest,
    input QueryContentInput,
) (*mcp.CallToolResult, QueryContentOutput, error) {

    siteCtx := fs.ctx.Load()

    // Apply filters
    pages := siteCtx.pages
    if input.Section != "" {
        pages = filterBySection(pages, input.Section)
    }
    if len(input.Tags) > 0 {
        pages = filterByTags(pages, input.Tags)
    }
    // ... more filters

    // Sort
    sortBy := cmp.Or(input.SortBy, "date")
    sortOrder := cmp.Or(input.SortOrder, "desc")
    pages = sortPages(pages, sortBy, sortOrder)

    // Paginate
    limit := cmp.Or(input.Limit, 20)
    total := len(pages)
    pages = paginate(pages, input.Offset, limit)

    output := QueryContentOutput{
        TotalMatches: total,
        Offset:       input.Offset,
        Limit:        limit,
        Pages:        toBriefs(pages),
    }

    return nil, output, nil
}
```

### 8.4 Taxonomy Similarity Detection

The `validate_frontmatter` tool uses Levenshtein distance to catch near-duplicate taxonomy terms:

```go
// internal/mcpserver/validate.go

func findSimilarTerms(input string, existing []string, threshold int) []string {
    var similar []string
    inputLower := strings.ToLower(input)
    for _, term := range existing {
        termLower := strings.ToLower(term)
        if inputLower == termLower {
            continue // exact match, not a suggestion
        }
        if levenshtein(inputLower, termLower) <= threshold {
            similar = append(similar, term)
        }
        // Also check common abbreviation patterns
        if isAbbreviation(inputLower, termLower) || isAbbreviation(termLower, inputLower) {
            similar = append(similar, term)
        }
    }
    return similar
}
```

Common catches: `k8s` ↔ `kubernetes`, `js` ↔ `javascript`, `ts` ↔ `typescript`, `tf` ↔ `terraform`, `infra` ↔ `infrastructure`.

### 8.5 Error Handling

All tool handlers follow a consistent error pattern:

- **Validation errors** (bad input): Return `*mcp.CallToolResult` with `IsError: true` and descriptive text content. These are "expected" errors the client can act on.
- **Internal errors** (bugs, filesystem failures): Return a Go `error`, which the SDK translates to a JSON-RPC error response.
- **Partial success** (build with warnings): Return success output with a populated `warnings` array.

```go
// Validation error — client should retry with corrected input
if input.Section != "" && !siteCtx.HasSection(input.Section) {
    return &mcp.CallToolResult{
        IsError: true,
        Content: []mcp.Content{
            &mcp.TextContent{
                Text: fmt.Sprintf(
                    "Unknown section %q. Available sections: %s",
                    input.Section,
                    strings.Join(siteCtx.SectionNames(), ", "),
                ),
            },
        },
    }, QueryContentOutput{}, nil
}
```

---

## 9. Configuration

### 9.1 MCP Server Configuration in forge.yaml

```yaml
# forge.yaml
mcp:
  enabled: true              # Enable forge mcp command
  watchFiles: true           # Watch for file changes and send notifications
  includeRenderedHTML: true   # Include rendered HTML in page detail resources
  maxContentLength: 50000    # Max chars of raw Markdown in page detail (0 = unlimited)
  similarityThreshold: 2     # Levenshtein distance threshold for tag/category suggestions
  abbreviations:             # Custom abbreviation mappings for taxonomy validation
    k8s: kubernetes
    js: javascript
    ts: typescript
    tf: terraform
    py: python
```

### 9.2 Claude Code Configuration

Users configure Claude Code to use the Forge MCP server in their `.mcp.json`:

```json
{
  "mcpServers": {
    "forge": {
      "command": "forge",
      "args": ["mcp", "--source", "."],
      "env": {}
    }
  }
}
```

Or globally in Claude Code settings for all Forge projects:

```json
{
  "mcpServers": {
    "forge": {
      "command": "forge",
      "args": ["mcp"],
      "env": {}
    }
  }
}
```

---

## 10. Testing Strategy

### 10.1 Unit Tests

- **Resource handlers:** Load a test site fixture, call each resource handler, assert JSON output matches expected structure and values
- **Tool handlers:** Test each tool with valid/invalid inputs, assert correct output and error handling
- **Taxonomy similarity:** Test Levenshtein matching with known pairs (`k8s`/`kubernetes`, `js`/`javascript`)
- **Frontmatter validation:** Test with valid, invalid, and edge-case frontmatter YAML
- **Query filtering:** Test all filter combinations (section + tags + date range + draft status)
- **Pagination:** Test boundary conditions (offset > total, limit = 0, etc.)

### 10.2 Integration Tests

- **Full server lifecycle:** Start MCP server over in-process transport, send `initialize`, call tools and resources, verify responses
- **File change notifications:** Modify a content file, assert `resources/list_changed` notification is sent
- **Build + deploy pipeline:** Call `build_site` tool, verify output, call `deploy_site` with `dryRun: true`, verify diff output
- **Site context caching:** Call a resource, modify a file, call again, verify context reload

### 10.3 Test Fixtures

Maintain a `testdata/mcp-test-site/` directory with:
- `forge.yaml` with known configuration
- 10+ content files across `blog/` and `projects/` sections
- Multiple taxonomy terms with known overlaps
- Draft and future-dated content
- Page bundles with co-located assets
- User layout overrides
- Data files (`skills.yaml`, `experience.yaml`)

### 10.4 MCP Inspector Validation

Use the [MCP Inspector](https://github.com/modelcontextprotocol/inspector) tool to manually validate the server during development:

```bash
npx @modelcontextprotocol/inspector forge mcp --source ./testdata/mcp-test-site
```

This provides an interactive UI to browse resources, call tools, and verify prompt generation.

---

## 11. Claude Code Integration

### 11.1 Recommended Workflow

When Claude Code connects to a Forge site with the MCP server:

1. **Session start:** Call `site_overview` prompt to load full context
2. **Content discovery:** Use `forge://content/pages` resource or `query_content` tool to understand what exists
3. **Content creation:** Use `create_content` tool (validates frontmatter, prevents taxonomy fragmentation)
4. **Content editing:** Use `get_page` tool to load full page detail, edit in place, use `validate_frontmatter` to verify
5. **Layout debugging:** Use `resolve_layout` and `get_template_context` when working on templates
6. **Build verification:** Call `build_site` after changes, inspect errors/warnings
7. **Deploy:** Call `deploy_site` (with `dryRun` first) when ready

### 11.2 Example Claude Code Interactions

**"Write me a blog post about Go error handling patterns"**
1. Claude Code calls `query_content` with `tags: ["go"]` to see existing Go content
2. Calls `forge://taxonomies/tags` to see existing tags
3. Calls `create_content` with type "post", title, and validated tags
4. Writes the content to the scaffolded file
5. Calls `validate_frontmatter` to verify
6. Calls `build_site` to check for errors

**"The blog post list page looks wrong — fix it"**
1. Claude Code calls `resolve_layout` for `content/blog/_index.md`
2. Calls `get_template_context` to see what data the list template receives
3. Reads the template file (via filesystem)
4. Identifies and fixes the template issue
5. Calls `build_site` to verify

**"Deploy my site"**
1. Claude Code calls `build_site` first
2. If successful, calls `deploy_site` with `dryRun: true`
3. Reviews the diff, then calls `deploy_site` to execute

---

## 12. Dependencies

### 12.1 New Go Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/modelcontextprotocol/go-sdk/mcp` | Official MCP Go SDK — server, transport, types |
| `github.com/agnivade/levenshtein` | Levenshtein distance for taxonomy similarity detection |

### 12.2 Existing Forge Dependencies (Reused)

| Package | Used For |
|---------|----------|
| `internal/config` | Site configuration loading |
| `internal/content` | Content discovery, frontmatter parsing, Markdown rendering |
| `internal/taxonomy` | Taxonomy building and querying |
| `internal/template` | Template resolution and context building |
| `internal/build` | Build pipeline execution |
| `internal/deploy` | S3 sync and CloudFront invalidation |
| `github.com/fsnotify/fsnotify` | File system watching for change notifications |

---

## 13. Implementation Phases

### Phase 1: Foundation

**Goal:** MCP server starts, handles `initialize`, and serves read-only resources.

- [ ] `forge mcp` CLI command (cobra subcommand, stdio transport)
- [ ] MCP server initialization with capability declaration
- [ ] `SiteContext` — lazy site loading with content graph caching
- [ ] Resource: `forge://config`
- [ ] Resource: `forge://content/pages`
- [ ] Resource: `forge://content/page/{path}` (resource template)
- [ ] Resource: `forge://content/sections`
- [ ] Resource: `forge://schema/frontmatter`
- [ ] Test fixtures and basic integration tests
- [ ] MCP Inspector validation

### Phase 2: Query & Introspection Tools

**Goal:** Claude Code can query and understand the site.

- [ ] Tool: `query_content` (full filter/sort/paginate)
- [ ] Tool: `get_page`
- [ ] Tool: `list_drafts`
- [ ] Resource: `forge://taxonomies`
- [ ] Resource: `forge://taxonomies/{name}`
- [ ] Resource: `forge://templates`
- [ ] Tool: `resolve_layout`
- [ ] Tool: `get_template_context`

### Phase 3: Content Creation & Validation

**Goal:** Claude Code can safely create and validate content.

- [ ] Tool: `create_content`
- [ ] Tool: `validate_frontmatter`
- [ ] Taxonomy similarity detection (Levenshtein + abbreviation mapping)
- [ ] Frontmatter schema validation engine
- [ ] Resource: `forge://build/status`

### Phase 4: Build, Deploy & Notifications

**Goal:** Full build/deploy cycle and live file watching.

- [ ] Tool: `build_site` (with progress notifications)
- [ ] Tool: `deploy_site` (with dry-run support)
- [ ] File watcher integration (`fsnotify`)
- [ ] `notifications/resources/list_changed` on file changes
- [ ] Progress notifications for long-running operations

### Phase 5: Prompts & Polish

**Goal:** Guided workflows and production readiness.

- [ ] Prompt: `new_blog_post`
- [ ] Prompt: `new_project`
- [ ] Prompt: `content_review`
- [ ] Prompt: `site_overview`
- [ ] MCP configuration section in `forge.yaml`
- [ ] Documentation (README section, `forge mcp --help`)
- [ ] End-to-end Claude Code testing

---

## Appendix A: JSON-RPC Examples

### Initialize Handshake

**Client → Server:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-06-18",
    "capabilities": {},
    "clientInfo": {
      "name": "claude-code",
      "version": "1.0.0"
    }
  }
}
```

**Server → Client:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-06-18",
    "capabilities": {
      "resources": { "listChanged": true },
      "tools": { "listChanged": false },
      "prompts": { "listChanged": false }
    },
    "serverInfo": {
      "name": "forge",
      "version": "0.1.0"
    }
  }
}
```

### Tool Call

**Client → Server:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "query_content",
    "arguments": {
      "section": "blog",
      "tags": ["kubernetes"],
      "sortBy": "date",
      "limit": 5
    }
  }
}
```

**Server → Client:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"totalMatches\":3,\"offset\":0,\"limit\":5,\"pages\":[...]}"
      }
    ]
  }
}
```

---

## Appendix B: Future Considerations

- **Streamable HTTP transport** — for remote MCP access (e.g., from a CI/CD pipeline or cloud-hosted Claude)
- **Resource subscriptions** — client subscribes to specific resources and gets pushed updates (MCP spec 2025-11-25)
- **Sampling** — server requests LLM completions from the client for auto-tagging or summary generation
- **OAuth** — for multi-user or remote deployment scenarios
- **Content linting tool** — deeper content analysis (readability scores, broken links, image alt text)
- **Git integration** — expose git history for content files (last editor, commit messages, diff)
