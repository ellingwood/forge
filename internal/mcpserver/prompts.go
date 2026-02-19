package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (fs *ForgeServer) registerPrompts() {
	fs.server.AddPrompt(&mcp.Prompt{
		Name:        "new_blog_post",
		Description: "Generate a complete blog post with proper frontmatter and Markdown structure",
		Arguments: []*mcp.PromptArgument{
			{Name: "topic", Description: "The topic or title of the blog post", Required: true},
			{Name: "tags", Description: "Comma-separated tags (leave empty for AI to suggest)"},
			{Name: "audience", Description: "Target audience (e.g. senior engineers, beginners)"},
		},
	}, fs.handleNewBlogPostPrompt)

	fs.server.AddPrompt(&mcp.Prompt{
		Name:        "new_project",
		Description: "Generate a project page with tech stack, links, and description",
		Arguments: []*mcp.PromptArgument{
			{Name: "name", Description: "Project name", Required: true},
			{Name: "techStack", Description: "Comma-separated technologies used"},
			{Name: "repoUrl", Description: "Source code repository URL"},
		},
	}, fs.handleNewProjectPrompt)

	fs.server.AddPrompt(&mcp.Prompt{
		Name:        "content_review",
		Description: "Review an existing content file for issues (SEO, formatting, metadata)",
		Arguments: []*mcp.PromptArgument{
			{Name: "pagePath", Description: "Path to the content file to review", Required: true},
		},
	}, fs.handleContentReviewPrompt)

	fs.server.AddPrompt(&mcp.Prompt{
		Name:        "site_overview",
		Description: "Generate a summary of the site's current state for context",
	}, fs.handleSiteOverviewPrompt)
}

func (fs *ForgeServer) handleNewBlogPostPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := req.Params.Arguments
	topic := args["topic"]
	tags := args["tags"]
	audience := args["audience"]
	if audience == "" {
		audience = "experienced developers"
	}

	sc, _ := fs.ctx.Load()
	var schemaText, tagsText string
	if sc != nil {
		sc.mu.RLock()
		existingTags := sc.AllTags()
		sc.mu.RUnlock()
		tagsText = "Existing tags: " + strings.Join(existingTags, ", ")
	}
	schemaText = "Use proper YAML frontmatter with: title, date, draft: true, tags, categories, description (under 160 chars)"

	if tags == "" {
		tags = "(suggest appropriate tags from existing ones)"
	}

	text := fmt.Sprintf(`Write a blog post about: %s

Target audience: %s

%s

%s

Requirements:
- Use proper YAML frontmatter delimited by ---
- Reuse existing tags where applicable to avoid taxonomy fragmentation
- Include a description field (under 160 chars) for SEO
- Use ## for top-level sections within the post
- Include code examples where relevant with language annotations
- End with a brief conclusion or summary
- Set draft: true

Requested tags: %s`, topic, audience, schemaText, tagsText, tags)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Write a blog post about: %s", topic),
		Messages: []*mcp.PromptMessage{
			{
				Role:    mcp.Role("user"),
				Content: &mcp.TextContent{Text: text},
			},
		},
	}, nil
}

func (fs *ForgeServer) handleNewProjectPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := req.Params.Arguments
	name := args["name"]
	techStack := args["techStack"]
	repoURL := args["repoUrl"]

	text := fmt.Sprintf(`Create a project page for: %s

Tech stack: %s
Repository: %s

Requirements:
- Use proper YAML frontmatter with: title, date, draft: true, description, tags (from tech stack)
- Include sections: Overview, Features, Tech Stack, Getting Started (if applicable)
- Add links to repository and live demo if available
- Keep description concise (under 160 chars) for SEO
- Set layout: project in frontmatter`, name, techStack, repoURL)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Create a project page for: %s", name),
		Messages: []*mcp.PromptMessage{
			{
				Role:    mcp.Role("user"),
				Content: &mcp.TextContent{Text: text},
			},
		},
	}, nil
}

func (fs *ForgeServer) handleContentReviewPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := req.Params.Arguments
	pagePath := args["pagePath"]

	sc, _ := fs.ctx.Load()
	var pageContent string
	if sc != nil {
		sc.mu.RLock()
		for _, p := range sc.pages {
			if p.SourcePath == pagePath || strings.HasSuffix(p.SourcePath, pagePath) {
				pageContent = fmt.Sprintf("Title: %s\nSection: %s\nTags: %s\nCategories: %s\nDescription: %s\n\n%s",
					p.Title, p.Section, strings.Join(p.Tags, ", "), strings.Join(p.Categories, ", "), p.Description, p.RawContent)
				break
			}
		}
		sc.mu.RUnlock()
	}
	if pageContent == "" {
		pageContent = fmt.Sprintf("(Could not load page: %s)", pagePath)
	}

	text := fmt.Sprintf(`Review this content file for issues:

%s

Check for:
1. Missing or suboptimal SEO fields (description, summary - should be under 160 chars)
2. Taxonomy consistency (tags/categories should match existing site terms)
3. Markdown formatting issues (heading hierarchy, code block languages)
4. Readability and structure (clear intro, sections, conclusion)
5. Missing frontmatter fields (date, draft status, cover image)

Provide specific suggestions for improvement.`, pageContent)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Review content: %s", pagePath),
		Messages: []*mcp.PromptMessage{
			{
				Role:    mcp.Role("user"),
				Content: &mcp.TextContent{Text: text},
			},
		},
	}, nil
}

func (fs *ForgeServer) handleSiteOverviewPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	sc, _ := fs.ctx.Load()
	var siteTitle, sections, taxonomy, buildInfo string

	if sc != nil {
		sc.mu.RLock()
		cfg := sc.cfg
		pages := sc.pages

		if cfg != nil {
			siteTitle = cfg.Title
		}

		sectionNames := sc.SectionNames()
		sections = strings.Join(sectionNames, ", ")

		tagCount := len(sc.AllTags())
		catCount := len(sc.AllCategories())
		taxonomy = fmt.Sprintf("%d tags, %d categories", tagCount, catCount)

		draftCount := 0
		for _, p := range pages {
			if p.Draft {
				draftCount++
			}
		}
		buildInfo = fmt.Sprintf("%d total pages (%d drafts)", len(pages), draftCount)
		sc.mu.RUnlock()
	}

	text := fmt.Sprintf(`Here is the current state of the Forge site:

Site: %s
Sections: %s
Taxonomy: %s
Content: %s

You now have full context of this site. You can:
- Query content with the query_content tool
- Create new content with create_content tool
- Validate frontmatter with validate_frontmatter tool
- Build the site with build_site tool
- Check build status at forge://build/status resource

What would you like to do?`, siteTitle, sections, taxonomy, buildInfo)

	return &mcp.GetPromptResult{
		Description: "Site overview and context",
		Messages: []*mcp.PromptMessage{
			{
				Role:    mcp.Role("user"),
				Content: &mcp.TextContent{Text: text},
			},
		},
	}, nil
}
