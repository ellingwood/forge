package build

import (
	"testing"
	"time"

	"github.com/aellingwood/forge/internal/content"
)

func TestBuildProjectPostMap(t *testing.T) {
	pages := []*content.Page{
		{Title: "Post A", Project: "forge", Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "Post B", Project: "forge", Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "Post C", Project: "other", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "No Project", Date: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)},
	}

	m := buildProjectPostMap(pages)

	if len(m) != 2 {
		t.Errorf("expected 2 project entries, got %d", len(m))
	}

	forgePosts := m["forge"]
	if len(forgePosts) != 2 {
		t.Fatalf("expected 2 posts for 'forge', got %d", len(forgePosts))
	}
	// Should be sorted newest-first.
	if forgePosts[0].Title != "Post B" {
		t.Errorf("expected newest post first, got %q", forgePosts[0].Title)
	}
	if forgePosts[1].Title != "Post A" {
		t.Errorf("expected oldest post second, got %q", forgePosts[1].Title)
	}

	otherPosts := m["other"]
	if len(otherPosts) != 1 {
		t.Fatalf("expected 1 post for 'other', got %d", len(otherPosts))
	}

	if _, ok := m[""]; ok {
		t.Error("empty project string should not be in the map")
	}
}

func TestBuildProjectPostMapEmpty(t *testing.T) {
	m := buildProjectPostMap(nil)
	if len(m) != 0 {
		t.Errorf("expected empty map, got %d entries", len(m))
	}
}

func TestBuildProjectPageIndex(t *testing.T) {
	pages := []*content.Page{
		{Title: "Forge", Slug: "forge", Section: "projects", Type: content.PageTypeSingle},
		{Title: "Other", Slug: "other", Section: "projects", Type: content.PageTypeSingle},
		{Title: "Projects List", Slug: "", Section: "projects", Type: content.PageTypeList},
		{Title: "Blog Post", Slug: "post", Section: "blog", Type: content.PageTypeSingle},
	}

	m := buildProjectPageIndex(pages)

	if len(m) != 2 {
		t.Errorf("expected 2 project page entries, got %d", len(m))
	}
	if m["forge"] == nil || m["forge"].Title != "Forge" {
		t.Error("expected 'forge' project page")
	}
	if m["other"] == nil || m["other"].Title != "Other" {
		t.Error("expected 'other' project page")
	}
	if _, ok := m["post"]; ok {
		t.Error("blog post should not be in project page index")
	}
}

func TestBuildProjectPageIndexEmpty(t *testing.T) {
	m := buildProjectPageIndex(nil)
	if len(m) != 0 {
		t.Errorf("expected empty map, got %d entries", len(m))
	}
}
