# README and CLAUDE.md Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a comprehensive `README.md` and expand `CLAUDE.md` with project-relevant guidance for Claude Code.

**Architecture:** Two independent file writes — `README.md` is a new file created from scratch, `CLAUDE.md` is edited in-place to append new sections below the existing `qmd` rule. No code changes; no tests needed for documentation.

**Tech Stack:** Markdown, GitHub-flavored markdown

---

### Task 1: Create README.md

**Files:**
- Create: `README.md`

**Step 1: Write README.md**

Create `/Users/aellingwood/dev/personal/forge/README.md` with this exact content:

````markdown
# Forge

> A fast, opinionated static site generator written in Go. Markdown in, beautiful HTML out.

## Features

- **Fast builds** — parallel content processing targets sub-second builds for sites under 500 pages
- **Single binary** — one compiled Go binary, no runtime dependencies
- **Live reload** — dev server with WebSocket-based live reload on file save
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
    distribution_id: EXXXXXXXXXXXXX
```

Forge uses the standard AWS credential chain (environment variables → `~/.aws/credentials` → IAM role). No credentials are stored in `forge.yaml`.

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
````

**Step 2: Verify the file renders correctly**

Open `README.md` and confirm:
- All headings are present (Features, Installation, Quick Start, CLI Reference, Site Structure, MCP Server, Deployment, Contributing, License)
- The CLI table has 11 rows
- Code blocks are fenced correctly (no unclosed blocks)

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add README"
```

Expected output: `1 file changed, N insertions(+)`

---

### Task 2: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Append new sections to CLAUDE.md**

The existing file ends after the `qmd` rule section. Append the following content after the last line:

```markdown

## Project Structure

```
forge/
├── cmd/forge/          # CLI entry points (cobra commands)
│   ├── main.go
│   ├── build.go        # forge build
│   ├── serve.go        # forge serve
│   ├── new.go          # forge new site/post/page/project
│   ├── deploy.go       # forge deploy
│   ├── list.go         # forge list drafts/future/expired
│   ├── mcp.go          # forge mcp
│   └── version.go      # forge version
├── internal/
│   ├── build/          # Build pipeline coordinator (parallel processing)
│   ├── config/         # Site configuration (YAML/TOML parsing via viper)
│   ├── content/        # Content discovery, frontmatter parsing, Markdown rendering
│   ├── deploy/         # S3 sync + CloudFront invalidation
│   ├── feed/           # RSS/Atom feed generation
│   ├── mcpserver/      # MCP server (resources, tools, prompts)
│   ├── render/         # Template execution orchestration
│   ├── scaffold/       # forge new scaffolding logic
│   ├── search/         # Search index (JSON) generation
│   ├── seo/            # Sitemap, robots.txt, meta tag generation
│   ├── server/         # Dev server with WebSocket live reload
│   └── template/       # Go html/template wrapper + custom functions
├── embedded/           # go:embed assets (default theme, Tailwind CLI binary)
├── themes/default/     # Default theme (layouts, static, theme.yaml)
└── testdata/           # Golden file test fixtures
```

## Development Commands

```bash
make build    # compile binary → ./forge
make test     # go test ./...
make fmt      # gofmt -s -w .
make vet      # go vet ./...
make lint     # golangci-lint run ./... (requires golangci-lint)
make install  # build + copy to ~/.local/bin/forge
make clean    # remove ./forge and public/
```

## Key Conventions

- **Frontmatter:** YAML by default; TOML supported (delimiter `+++`)
- **Output directory:** always `public/` (configurable in `forge.yaml`)
- **Default theme:** embedded via `go:embed` from `themes/default/`; users override by placing files in their own `themes/default/`
- **MCP server:** lives in `internal/mcpserver/`; thin layer over existing internal packages
- **Config:** `forge.yaml` at site root; loaded via viper, supports env var overrides

## Testing Notes

- Golden file tests live in `testdata/`; input Markdown → expected HTML output
- Regenerate golden files: `go test ./... -update`
- Integration tests build the `testdata/` test site and assert output structure
- Run a single package: `go test ./internal/content/...`
```

**Step 2: Verify CLAUDE.md looks correct**

Read the file and confirm:
- The original `qmd` rule section is intact at the top
- Four new sections are present: Project Structure, Development Commands, Key Conventions, Testing Notes
- No broken code fences

**Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: expand CLAUDE.md with project structure and dev commands"
```

Expected output: `1 file changed, N insertions(+)`
