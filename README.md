# Forge

> A fast, opinionated static site generator written in Go. Markdown in, beautiful HTML out.

## Features

- **Fast builds** — parallel content processing targets sub-second builds for sites under 500 pages
- **Single binary** — one compiled Go binary, no runtime dependencies
- **Live reload** — dev server with WebSocket-based live reload on file save (theme directory changes require a server restart)
- **Tailwind CSS** — standalone CLI integration, no Node.js required
- **Syntax highlighting** — 200+ languages via chroma
- **RSS + Atom feeds** — global and per-section feed generation
- **Sitemap + SEO** — `sitemap.xml`, `robots.txt`, OpenGraph and Twitter Card meta tags
- **Client-side search** — Fuse.js with a pre-built JSON index, no server required
- **MCP server** — Model Context Protocol server for AI-assisted site development
- **S3 + CloudFront deploy** — content-hash diffing, correct cache headers, invalidation

## Installation

**From source (requires Go 1.26+):**

```bash
go install github.com/ellingwood/forge/cmd/forge@latest
```

**Or build and install locally:**

```bash
git clone https://github.com/ellingwood/forge
cd forge
make install   # builds and copies binary to ~/.local/bin/forge
```

## Quick Start

```bash
# Create a new site
forge new site my-site
cd my-site

# Start the dev server with live reload
forge serve
# Open http://localhost:1313

# Build for production
forge build
# Output is written to public/
```

## CLI Reference

| Command | Description |
| --------- | ------------- |
| `forge build` | Build the static site into `public/` |
| `forge serve` | Start the dev server with live reload (default: `localhost:1313`) |
| `forge new site <name>` | Scaffold a new site with default theme |
| `forge new post <title>` | Create a new blog post |
| `forge new page <title>` | Create a new standalone page |
| `forge new project <title>` | Create a new project entry |
| `forge list drafts` | List all draft content |
| `forge list future` | List future-dated content |
| `forge list expired` | List expired content |
| `forge deploy` | Deploy `public/` to S3 + CloudFront |
| `forge config` | Print the fully resolved site configuration |
| `forge mcp` | Start the MCP server over stdio |
| `forge version` | Print version, commit hash, and build date |

## Site Structure

```tree
my-site/
├── forge.yaml              # Site configuration
├── content/
│   ├── _index.md           # Homepage content
│   ├── about.md            # About page
│   ├── blog/
│   │   ├── _index.md       # Blog listing config
│   │   └── my-first-post.md
│   └── projects/
│       └── my-project.md
├── layouts/                # Template overrides (override default theme layouts here)
└── public/                 # Generated output (git-ignored)
```

Content files use YAML frontmatter (TOML also supported):

```markdown
---
title: "My First Post"
date: 2026-01-15
draft: false
tags: ["go", "web"]
---

Post content here...
```

## Media & Assets

Forge supports three patterns for images and other static files.

### Global assets: `static/`

Files in `static/` are copied verbatim into `public/` at the same relative path.
Use this for site-wide assets (favicon, logo, Open Graph image) that are not tied
to a specific page.

```tree
my-site/
└── static/
    ├── favicon.ico          → public/favicon.ico
    └── images/
        └── og-image.png     → public/images/og-image.png
```

Reference them in Markdown with a root-relative path:

```markdown
![Site logo](/images/og-image.png)
```

### Page bundles (co-located assets)

A page bundle is a directory whose name matches the post slug and that contains
an `index.md` alongside any assets for that page. Forge copies all non-Markdown
files in the bundle directory next to the rendered `index.html`.

```tree
content/blog/
└── my-post/
    ├── index.md             # page content + frontmatter
    ├── hero.png             → public/blog/my-post/hero.png
    └── diagram.svg          → public/blog/my-post/diagram.svg
```

Reference bundle assets with a relative path in the Markdown:

```markdown
![Architecture diagram](diagram.svg)
```

### Cover images

Any page can declare a cover image in its frontmatter. Forge passes this to the
theme as `.Cover`, which the default theme renders as a thumbnail on post-card
listings and as a full-width banner at the top of single posts.

```yaml
---
title: "My Post"
date: 2025-01-15
cover:
  image: hero.png        # relative to the bundle, or an absolute path from public/
  alt: "Descriptive alt text"
  caption: "Optional caption shown below the image"
---
```

Use a bundle-relative path (`hero.png`) for co-located images, or a root-relative
path (`/images/hero.png`) for images in `static/`.

## MCP Server

Forge ships an [MCP](https://modelcontextprotocol.io) server that gives AI clients (Claude Code, etc.) semantic access to your site's content graph, build system, and configuration — without parsing raw files.

**Add to Claude Code** (`.claude/mcp.json` in your site directory):

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

The server exposes resources (site config, pages, taxonomies), tools (query content, create content, trigger builds), and prompt templates.

## Deployment

Configure your S3 bucket and CloudFront distribution in `forge.yaml`:

```yaml
deploy:
  s3:
    bucket: my-site-bucket
    region: us-east-1
  cloudfront:
    distributionId: EXXXXXXXXXXXXX
```

Forge uses the standard AWS credential chain (environment variables → `~/.aws/credentials` → IAM role). No credentials are stored in `forge.yaml`.

**Note:** Deployment is work in progress. The `forge deploy` command is not yet wired up.

```bash
forge build && forge deploy
forge deploy --dry-run   # preview what would change
```

## Contributing

```bash
# Run all tests
make test          # go test ./...

# Format and lint
make fmt           # gofmt -s -w .
make vet           # go vet ./...
make lint          # golangci-lint run ./... (requires golangci-lint)
```

Pull requests welcome. Please open an issue first for significant changes.

## License

MIT — see [LICENSE](LICENSE).
