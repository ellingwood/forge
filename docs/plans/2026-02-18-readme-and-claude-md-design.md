# Design: README.md and CLAUDE.md

**Date:** 2026-02-18
**Audience:** Both personal reference and open-source users

---

## Goal

Create a comprehensive `README.md` and expand `CLAUDE.md` with project-relevant guidance for Claude Code.

---

## README.md Structure

```
# Forge
> A fast, opinionated static site generator written in Go.
  Markdown in, beautiful HTML out.

## Features
Bullet list: fast builds, single binary, live reload, Tailwind CSS,
MCP server, S3+CloudFront deploy, syntax highlighting, RSS/sitemap, etc.

## Installation
go install, make install, or binary download.

## Quick Start
forge new site my-site → forge serve → forge build

## CLI Reference
Table: command | description
Commands: build, serve, new site/post/page/project,
deploy, list drafts/future/expired, mcp, version

## Site Structure
The my-site/ directory layout (content/, themes/, public/, forge.yaml)

## MCP Server
What it is, how to wire it into Claude Code via
claude_desktop_config or .claude/mcp.json

## Deployment
Brief note on forge deploy + required AWS env vars

## Contributing
go test ./..., make fmt/lint/vet, PR welcome

## License
```

---

## CLAUDE.md Updates

Append the following sections below the existing `qmd` rule:

**Project Structure** — brief map of `cmd/` and `internal/` packages with one-line purpose each.

**Development Commands** — `make build`, `make test`, `make fmt`, `make vet`, `make lint`, `make install`.

**Key Conventions:**
- Frontmatter: YAML default, TOML supported
- Output always goes to `public/`
- Default theme embedded via `go:embed` in `themes/default/`
- MCP server lives in `internal/mcpserver/`

**Testing Notes** — golden files in `testdata/`, `-update` flag to regenerate.

---

## Out of Scope

- Animated demos or screenshots
- Badge collection
- Detailed feature comparison vs Hugo/Jekyll
