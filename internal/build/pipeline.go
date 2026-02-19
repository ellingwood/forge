package build

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/aellingwood/forge/internal/content"
)

// renderParallel processes pages concurrently using a worker pool.
// The fn callback is invoked for each page. If any invocation returns an error,
// processing stops and the first error is returned.
func renderParallel(pages []*content.Page, workers int, fn func(*content.Page) error) error {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if len(pages) == 0 {
		return nil
	}
	// Don't create more workers than pages.
	if workers > len(pages) {
		workers = len(pages)
	}

	jobs := make(chan *content.Page, len(pages))
	errCh := make(chan error, 1) // buffered so the first error doesn't block
	var once sync.Once           // ensure we only send one error
	var wg sync.WaitGroup

	// Start workers.
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for page := range jobs {
				if err := fn(page); err != nil {
					once.Do(func() {
						errCh <- fmt.Errorf("processing page %s: %w", page.SourcePath, err)
					})
					return
				}
			}
		}()
	}

	// Send jobs.
	for _, p := range pages {
		jobs <- p
	}
	close(jobs)

	// Wait for workers to finish.
	wg.Wait()
	close(errCh)

	// Return the first error, if any.
	if err, ok := <-errCh; ok {
		return err
	}
	return nil
}

// setSectionNavigation sets PrevPage and NextPage links for pages within
// the same section. Pages should already be sorted (newest first).
func setSectionNavigation(pages []*content.Page) {
	// Group pages by section.
	sections := make(map[string][]*content.Page)
	for _, p := range pages {
		if p.Type == content.PageTypeSingle {
			sections[p.Section] = append(sections[p.Section], p)
		}
	}

	// Set prev/next within each section.
	for _, sectionPages := range sections {
		for i, p := range sectionPages {
			if i > 0 {
				p.NextPage = sectionPages[i-1] // newer page
			}
			if i < len(sectionPages)-1 {
				p.PrevPage = sectionPages[i+1] // older page
			}
		}
	}
}

// buildTaxonomyMaps builds maps from taxonomy term to pages.
// Returns maps for tags and categories.
func buildTaxonomyMaps(pages []*content.Page) (tags map[string][]*content.Page, categories map[string][]*content.Page) {
	tags = make(map[string][]*content.Page)
	categories = make(map[string][]*content.Page)

	for _, p := range pages {
		for _, tag := range p.Tags {
			tags[tag] = append(tags[tag], p)
		}
		for _, cat := range p.Categories {
			categories[cat] = append(categories[cat], p)
		}
	}
	return tags, categories
}
