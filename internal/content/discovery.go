package content

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aellingwood/forge/internal/config"
)

// datePrefixRe matches a leading YYYY-MM-DD- date prefix in a filename.
var datePrefixRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)

// slugifyRe removes characters that are not alphanumeric, hyphens, or periods.
var slugifyRe = regexp.MustCompile(`[^a-z0-9\-.]`)

// multiHyphenRe collapses multiple consecutive hyphens into one.
var multiHyphenRe = regexp.MustCompile(`-{2,}`)

// Discover walks the content directory and builds a slice of Page objects.
// It reads each .md file, parses front matter, determines page type, section,
// slug, URL, and collects bundle files. It does NOT render markdown or filter
// drafts/future/expired pages.
func Discover(contentDir string, cfg *config.SiteConfig) ([]*Page, error) {
	var pages []*Page

	// First pass: collect all index.md directories to identify page bundles.
	bundleDirs := make(map[string]bool)
	err := filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "index.md" {
			bundleDirs[filepath.Dir(path)] = true
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning for page bundles: %w", err)
	}

	// Second pass: discover pages.
	err = filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Only process .md files.
		if filepath.Ext(path) != ".md" {
			return nil
		}

		// Skip .md files in bundle directories that are not the index.md itself.
		dir := filepath.Dir(path)
		if bundleDirs[dir] && filepath.Base(path) != "index.md" {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		metadata, body, err := ParseFrontmatter(raw)
		if err != nil {
			return fmt.Errorf("parsing frontmatter in %s: %w", path, err)
		}

		page := &Page{}
		if metadata != nil {
			if err := PopulatePage(page, metadata); err != nil {
				return fmt.Errorf("populating page from %s: %w", path, err)
			}
		}

		page.RawContent = string(body)

		// Set source path relative to contentDir.
		relPath, err := filepath.Rel(contentDir, path)
		if err != nil {
			return fmt.Errorf("computing relative path for %s: %w", path, err)
		}
		page.SourcePath = filepath.ToSlash(relPath)
		page.SourceDir = filepath.ToSlash(filepath.Dir(relPath))
		if page.SourceDir == "." {
			page.SourceDir = ""
		}

		// Determine section (first path component under contentDir).
		page.Section = firstPathComponent(page.SourcePath)

		// Determine page type.
		filename := filepath.Base(path)
		isBundle := bundleDirs[dir]

		switch {
		case filename == "_index.md" && page.SourceDir == "":
			// Root _index.md -> Home page.
			page.Type = PageTypeHome
		case filename == "_index.md":
			// Section _index.md -> List page.
			page.Type = PageTypeList
		default:
			// Regular page or bundle.
			page.Type = PageTypeSingle
		}

		if isBundle {
			page.IsBundle = true
			page.BundleDir = filepath.ToSlash(dir)
			page.BundleFiles = collectBundleFiles(dir)
		}

		// Generate slug if not set in frontmatter.
		if page.Slug == "" && page.Type == PageTypeSingle {
			name := strings.TrimSuffix(filename, ".md")
			// For bundles, use the directory name.
			if isBundle {
				name = filepath.Base(dir)
			}
			// Strip date prefix.
			name = datePrefixRe.ReplaceAllString(name, "")
			page.Slug = slugify(name)
		}

		// Generate URL.
		page.URL = buildURL(page)

		// Calculate word count and reading time.
		page.WordCount = countWords(page.RawContent)
		if page.WordCount > 0 {
			page.ReadingTime = page.WordCount / 200
			if page.ReadingTime < 1 {
				page.ReadingTime = 1
			}
		}

		pages = append(pages, page)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking content directory: %w", err)
	}

	return pages, nil
}

// slugify converts a name into a URL-safe slug.
// It lowercases, replaces spaces and underscores with hyphens, removes
// non-alphanumeric characters (except hyphens and periods), collapses
// multiple hyphens, and trims leading/trailing hyphens.
func slugify(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = slugifyRe.ReplaceAllString(s, "")
	s = multiHyphenRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// firstPathComponent returns the first directory in a slash-separated path,
// or "" if the path has no directory component (i.e., root-level file).
func firstPathComponent(relPath string) string {
	relPath = filepath.ToSlash(relPath)
	parts := strings.SplitN(relPath, "/", 2)
	if len(parts) < 2 {
		// Root-level file, no section.
		return ""
	}
	return parts[0]
}

// buildURL generates the relative URL for a page based on its type, section, and slug.
func buildURL(p *Page) string {
	switch p.Type {
	case PageTypeHome:
		return "/"
	case PageTypeList:
		return "/" + p.Section + "/"
	case PageTypeSingle:
		if p.Section == "" {
			return "/" + p.Slug + "/"
		}
		return "/" + p.Section + "/" + p.Slug + "/"
	default:
		return "/"
	}
}

// collectBundleFiles returns the relative filenames of non-.md files
// co-located in a page bundle directory.
func collectBundleFiles(dir string) []string {
	var files []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".md" {
			continue
		}
		files = append(files, entry.Name())
	}
	return files
}

// countWords counts words by splitting on whitespace.
func countWords(s string) int {
	fields := strings.Fields(s)
	return len(fields)
}
