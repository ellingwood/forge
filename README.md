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
go install github.com/aellingwood/forge/cmd/forge@latest
```

**Or build and install locally:**

```bash
git clone https://github.com/aellingwood/forge
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

# Build for production
forge build
# Output is written to public/
```

## CLI Reference

| Command | Description |
|---------|-------------|
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
| `forge mcp` | Start the MCP server over stdio |
| `forge version` | Print version, commit hash, and build date |

## Site Structure

```
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
├── themes/
│   └── default/            # Override default theme here
│       ├── layouts/
│       └── static/
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
