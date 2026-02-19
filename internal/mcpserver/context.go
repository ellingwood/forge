package mcpserver

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aellingwood/forge/internal/config"
	"github.com/aellingwood/forge/internal/content"
	"github.com/aellingwood/forge/internal/scaffold"
)

// SiteContext holds a cached, in-memory representation of the site's content graph.
type SiteContext struct {
	mu       sync.RWMutex
	cfg      *config.SiteConfig
	pages    []*content.Page
	siteDir  string
	loadedAt time.Time
	dirty    bool
}

// NewSiteContext creates a new SiteContext for the given site directory.
func NewSiteContext(siteDir string) *SiteContext {
	return &SiteContext{
		siteDir: siteDir,
		dirty:   true,
	}
}

// Load returns the loaded (and cached) site context, reloading if dirty.
func (sc *SiteContext) Load() (*SiteContext, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if !sc.dirty && !sc.loadedAt.IsZero() {
		return sc, nil
	}

	cfg, err := config.Load(filepath.Join(sc.siteDir, "forge.yaml"))
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	contentDir := filepath.Join(sc.siteDir, "content")
	pages, err := content.Discover(contentDir, cfg)
	if err != nil {
		return nil, fmt.Errorf("discovering content: %w", err)
	}

	sc.cfg = cfg
	sc.pages = pages
	sc.loadedAt = time.Now()
	sc.dirty = false

	return sc, nil
}

// MarkDirty marks the context as needing a reload.
func (sc *SiteContext) MarkDirty() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.dirty = true
}

// HasSection returns true if the given section name exists.
func (sc *SiteContext) HasSection(name string) bool {
	for _, p := range sc.pages {
		if p.Section == name {
			return true
		}
	}
	return false
}

// SectionNames returns all unique section names.
func (sc *SiteContext) SectionNames() []string {
	seen := make(map[string]bool)
	for _, p := range sc.pages {
		if p.Section != "" {
			seen[p.Section] = true
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// AllTags returns all unique tag values across all pages.
func (sc *SiteContext) AllTags() []string {
	seen := make(map[string]bool)
	for _, p := range sc.pages {
		for _, t := range p.Tags {
			seen[t] = true
		}
	}
	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// AllCategories returns all unique category values across all pages.
func (sc *SiteContext) AllCategories() []string {
	seen := make(map[string]bool)
	for _, p := range sc.pages {
		for _, c := range p.Categories {
			seen[c] = true
		}
	}
	cats := make([]string, 0, len(seen))
	for c := range seen {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	return cats
}

// AllSeries returns all unique series values across all pages.
func (sc *SiteContext) AllSeries() []string {
	seen := make(map[string]bool)
	for _, p := range sc.pages {
		if p.Series != "" {
			seen[p.Series] = true
		}
	}
	series := make([]string, 0, len(seen))
	for s := range seen {
		series = append(series, s)
	}
	sort.Strings(series)
	return series
}

// AllProjectSlugs returns all unique slugs of project pages (section=="projects", type==single).
func (sc *SiteContext) AllProjectSlugs() []string {
	seen := make(map[string]bool)
	for _, p := range sc.pages {
		if p.Section == "projects" && p.Type == content.PageTypeSingle && p.Slug != "" {
			seen[p.Slug] = true
		}
	}
	slugs := make([]string, 0, len(seen))
	for s := range seen {
		slugs = append(slugs, s)
	}
	sort.Strings(slugs)
	return slugs
}

// SlugifyTitle returns a URL-safe slug from a title.
func SlugifyTitle(title string) string {
	return scaffold.Slugify(title)
}

// normalizeTaxonomyName lowercases a taxonomy name for use in URLs.
func normalizeTaxonomyName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}
