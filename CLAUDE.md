# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Rule: always use qmd before reading files

Before reading files or exploring directories, always use qmd to search for information in local projects.

Available tools:

- `qmd search “query”` — fast keyword search (BM25)

- `qmd query “query”` — hybrid search with reranking (best quality)

- `qmd vsearch “query”` — semantic vector search

- `qmd get <file>` — retrieve a specific document

Use qmd search for quick lookups and qmd query for complex questions.

Use Read/Glob only if qmd doesn’t return enough results.

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
├── embedded/           # go:embed assets (default theme)
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
- Update testdata fixtures manually by editing files in `testdata/`
- Integration tests build the `testdata/` test site and assert output structure
- Run a single package: `go test ./internal/content/...`
