// Package render bridges content processing and template execution, converting
// internal content types into template contexts and orchestrating the full page
// rendering pipeline.
package render

import (
	"fmt"
	"html/template"
	"time"

	"github.com/aellingwood/forge/internal/config"
	"github.com/aellingwood/forge/internal/content"
	tmpl "github.com/aellingwood/forge/internal/template"
)

// Renderer orchestrates the rendering pipeline: it takes parsed content pages,
// converts them into template contexts, resolves the appropriate template, and
// executes it to produce final HTML output.
type Renderer struct {
	engine   *tmpl.Engine
	markdown *content.MarkdownRenderer
	config   *config.SiteConfig
}

// NewRenderer creates a Renderer with the given template engine, markdown
// renderer, and site configuration.
func NewRenderer(engine *tmpl.Engine, markdown *content.MarkdownRenderer, cfg *config.SiteConfig) *Renderer {
	return &Renderer{
		engine:   engine,
		markdown: markdown,
		config:   cfg,
	}
}

// BuildPageContext converts an internal content.Page into a template.PageContext
// suitable for template execution. The allPages slice is used to build the
// associated SiteContext. PrevPage and NextPage are converted one level deep
// to avoid infinite recursion.
func (r *Renderer) BuildPageContext(page *content.Page, allPages []*content.Page) *tmpl.PageContext {
	return r.buildPageContext(page, allPages, true)
}

// buildPageContext is the internal implementation that tracks recursion depth
// for PrevPage/NextPage conversion.
func (r *Renderer) buildPageContext(page *content.Page, allPages []*content.Page, includeNav bool) *tmpl.PageContext {
	ctx := &tmpl.PageContext{
		Title:           page.Title,
		Description:     page.Description,
		Content:         template.HTML(page.Content),
		Summary:         template.HTML(page.Summary),
		Date:            page.Date,
		Lastmod:         page.Lastmod,
		Draft:           page.Draft,
		Slug:            page.Slug,
		URL:             page.URL,
		Permalink:       page.Permalink,
		ReadingTime:     page.ReadingTime,
		WordCount:       page.WordCount,
		Tags:            page.Tags,
		Categories:      page.Categories,
		Series:          page.Series,
		Params:          page.Params,
		TableOfContents: template.HTML(page.TableOfContents),
		Section:         page.Section,
		Type:            page.Type.String(),
	}

	// Convert cover image if present.
	if page.Cover != nil {
		ctx.Cover = &tmpl.CoverImage{
			Image:   page.Cover.Image,
			Alt:     page.Cover.Alt,
			Caption: page.Cover.Caption,
		}
	}

	// Convert PrevPage/NextPage one level deep to avoid infinite recursion.
	if includeNav {
		if page.PrevPage != nil {
			ctx.PrevPage = r.buildPageContext(page.PrevPage, nil, false)
		}
		if page.NextPage != nil {
			ctx.NextPage = r.buildPageContext(page.NextPage, nil, false)
		}
	}

	return ctx
}

// BuildSiteContext builds the site-wide context from the configuration and all
// content pages. It groups pages by section and builds taxonomy maps for tags
// and categories.
func (r *Renderer) BuildSiteContext(allPages []*content.Page) *tmpl.SiteContext {
	// Convert all pages to PageContexts.
	pageContexts := make([]*tmpl.PageContext, len(allPages))
	for i, p := range allPages {
		pageContexts[i] = r.buildPageContext(p, nil, false)
	}

	// Build sections map: section name -> page contexts.
	sections := make(map[string][]*tmpl.PageContext)
	for i, p := range allPages {
		if p.Section != "" {
			sections[p.Section] = append(sections[p.Section], pageContexts[i])
		}
	}

	// Build taxonomies map: taxonomy name -> term -> page contexts.
	taxonomies := make(map[string]map[string][]*tmpl.PageContext)

	// Tags taxonomy.
	tagMap := make(map[string][]*tmpl.PageContext)
	for i, p := range allPages {
		for _, tag := range p.Tags {
			tagMap[tag] = append(tagMap[tag], pageContexts[i])
		}
	}
	if len(tagMap) > 0 {
		taxonomies["tags"] = tagMap
	}

	// Categories taxonomy.
	catMap := make(map[string][]*tmpl.PageContext)
	for i, p := range allPages {
		for _, cat := range p.Categories {
			catMap[cat] = append(catMap[cat], pageContexts[i])
		}
	}
	if len(catMap) > 0 {
		taxonomies["categories"] = catMap
	}

	// Build menu items from config.
	menuItems := make([]tmpl.MenuItemContext, len(r.config.Menu.Main))
	for i, item := range r.config.Menu.Main {
		menuItems[i] = tmpl.MenuItemContext{
			Name:   item.Name,
			URL:    item.URL,
			Weight: item.Weight,
		}
	}

	return &tmpl.SiteContext{
		Title:       r.config.Title,
		Description: r.config.Description,
		BaseURL:     r.config.BaseURL,
		Language:    r.config.Language,
		Author: tmpl.AuthorContext{
			Name:   r.config.Author.Name,
			Email:  r.config.Author.Email,
			Bio:    r.config.Author.Bio,
			Avatar: r.config.Author.Avatar,
			Social: tmpl.SocialContext{
				GitHub:   r.config.Author.Social.GitHub,
				LinkedIn: r.config.Author.Social.LinkedIn,
				Twitter:  r.config.Author.Social.Twitter,
				Mastodon: r.config.Author.Social.Mastodon,
				Email:    r.config.Author.Social.Email,
			},
		},
		Menu:       menuItems,
		Params:     r.config.Params,
		Data:       make(map[string]any),
		Pages:      pageContexts,
		Sections:   sections,
		Taxonomies: taxonomies,
		BuildDate:  time.Now(),
	}
}

// RenderPage performs the full rendering pipeline for a single page:
//  1. Renders the page's markdown content to HTML (with TOC).
//  2. Builds PageContext and SiteContext.
//  3. Resolves the appropriate template.
//  4. Executes the template and returns the rendered HTML bytes.
func (r *Renderer) RenderPage(page *content.Page, allPages []*content.Page) ([]byte, error) {
	// Render markdown content to HTML.
	htmlContent, tocHTML, err := r.markdown.RenderWithTOC([]byte(page.RawContent))
	if err != nil {
		return nil, fmt.Errorf("rendering markdown for %q: %w", page.Title, err)
	}

	// Store rendered content back on the page.
	page.Content = string(htmlContent)
	page.TableOfContents = string(tocHTML)

	// Build contexts.
	pageCtx := r.BuildPageContext(page, allPages)
	siteCtx := r.BuildSiteContext(allPages)
	pageCtx.Site = siteCtx

	// Resolve the template.
	pageType := page.Type.String()
	templateName := r.engine.Resolve(pageType, page.Section, page.Layout)
	if templateName == "" {
		return nil, fmt.Errorf("no template found for page %q (type=%s, section=%s, layout=%s)",
			page.Title, pageType, page.Section, page.Layout)
	}

	// Execute the template.
	output, err := r.engine.Execute(templateName, pageCtx)
	if err != nil {
		return nil, fmt.Errorf("executing template %q for page %q: %w", templateName, page.Title, err)
	}

	return output, nil
}
