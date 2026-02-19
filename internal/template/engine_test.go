package template

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testdataThemePath returns the path to testdata used as a "theme" directory.
// The testdata directory mimics a theme with a layouts/ subdirectory.
func testdataThemePath(t *testing.T) string {
	t.Helper()
	// The testdata directory acts as the theme root; it contains layouts/.
	return filepath.Join("testdata")
}

func TestNewEngine(t *testing.T) {
	themePath := testdataThemePath(t)
	eng, err := NewEngine(themePath, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	// Verify that key templates were loaded.
	expected := []string{
		"_default/baseof.html",
		"_default/single.html",
		"_default/list.html",
		"index.html",
		"partials/header.html",
	}
	for _, name := range expected {
		if !eng.HasTemplate(name) {
			t.Errorf("expected template %q to be loaded", name)
		}
	}
}

func TestResolve(t *testing.T) {
	themePath := testdataThemePath(t)
	eng, err := NewEngine(themePath, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	tests := []struct {
		name     string
		pageType string
		section  string
		layout   string
		want     string
	}{
		{
			name:     "single page falls back to _default/single.html",
			pageType: "single",
			section:  "blog",
			layout:   "",
			want:     "_default/single.html",
		},
		{
			name:     "single page with layout falls back to _default/single.html",
			pageType: "single",
			section:  "blog",
			layout:   "post",
			want:     "_default/single.html",
		},
		{
			name:     "list page falls back to _default/list.html",
			pageType: "list",
			section:  "blog",
			layout:   "",
			want:     "_default/list.html",
		},
		{
			name:     "home page resolves to index.html",
			pageType: "home",
			section:  "",
			layout:   "",
			want:     "index.html",
		},
		{
			name:     "taxonomy falls back to _default/list.html",
			pageType: "taxonomy",
			section:  "tags",
			layout:   "",
			want:     "_default/list.html",
		},
		{
			name:     "taxonomylist falls back to _default/list.html",
			pageType: "taxonomylist",
			section:  "tags",
			layout:   "",
			want:     "_default/list.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eng.Resolve(tt.pageType, tt.section, tt.layout)
			if got != tt.want {
				t.Errorf("Resolve(%q, %q, %q) = %q, want %q",
					tt.pageType, tt.section, tt.layout, got, tt.want)
			}
		})
	}
}

func TestExecute(t *testing.T) {
	// Create an isolated template set to avoid define "main" conflicts.
	themeDir := t.TempDir()
	layoutDir := filepath.Join(themeDir, "layouts", "_default")
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		t.Fatal(err)
	}

	baseof := `<!DOCTYPE html><html><body>{{ block "main" . }}{{ end }}</body></html>`
	single := `{{ define "main" }}<h1>{{ .Title }}</h1>{{ .Content }}{{ end }}`

	if err := os.WriteFile(filepath.Join(layoutDir, "baseof.html"), []byte(baseof), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layoutDir, "single.html"), []byte(single), 0o644); err != nil {
		t.Fatal(err)
	}

	eng, err := NewEngine(themeDir, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	ctx := &PageContext{
		Title:   "Test Post",
		Content: template.HTML("<p>Hello World</p>"),
		Site: &SiteContext{
			Title: "My Site",
			Pages: []*PageContext{
				{Title: "Page One"},
				{Title: "Page Two"},
			},
		},
	}

	// Execute baseof which renders the block "main" defined by single.html.
	out, err := eng.Execute("_default/baseof.html", ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := string(out)
	if !strings.Contains(output, "<h1>Test Post</h1>") {
		t.Errorf("output should contain title, got: %s", output)
	}
	if !strings.Contains(output, "<p>Hello World</p>") {
		t.Errorf("output should contain content, got: %s", output)
	}
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Errorf("output should contain doctype from baseof, got: %s", output)
	}
}

func TestExecutePartial(t *testing.T) {
	themePath := testdataThemePath(t)
	eng, err := NewEngine(themePath, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	ctx := &PageContext{
		Title: "Test Page",
		Site: &SiteContext{
			Title: "My Site",
		},
	}

	result, err := eng.executePartial("header.html", ctx)
	if err != nil {
		t.Fatalf("executePartial failed: %v", err)
	}

	if !strings.Contains(string(result), "My Site") {
		t.Errorf("partial should contain site title, got: %s", result)
	}
	if !strings.Contains(string(result), "<header>") {
		t.Errorf("partial should contain header tag, got: %s", result)
	}
}

func TestFuncMap(t *testing.T) {
	fm := FuncMap()

	t.Run("truncate", func(t *testing.T) {
		fn := fm["truncate"].(func(int, string) string)

		tests := []struct {
			n    int
			s    string
			want string
		}{
			{10, "short", "short"},
			{10, "a longer string here", "a longe..."},
			{3, "abcdef", "abc"},
			{5, "hello", "hello"},
			{8, "hello world", "hello..."},
		}
		for _, tt := range tests {
			got := fn(tt.n, tt.s)
			if got != tt.want {
				t.Errorf("truncate(%d, %q) = %q, want %q", tt.n, tt.s, got, tt.want)
			}
		}
	})

	t.Run("slugify", func(t *testing.T) {
		fn := fm["slugify"].(func(string) string)

		tests := []struct {
			input string
			want  string
		}{
			{"Hello World", "hello-world"},
			{"My First Post!", "my-first-post"},
			{"  spaces  everywhere  ", "spaces-everywhere"},
			{"CamelCase", "camelcase"},
			{"already-slugified", "already-slugified"},
			{"special@#$chars", "special-chars"},
		}
		for _, tt := range tests {
			got := fn(tt.input)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		}
	})

	t.Run("safeHTML", func(t *testing.T) {
		fn := fm["safeHTML"].(func(string) template.HTML)
		input := "<strong>bold</strong>"
		got := fn(input)
		if string(got) != input {
			t.Errorf("safeHTML(%q) = %q, want %q", input, got, input)
		}
	})

	t.Run("dateFormat", func(t *testing.T) {
		fn := fm["dateFormat"].(func(string, time.Time) string)
		date := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
		got := fn("January 2, 2006", date)
		want := "January 15, 2025"
		if got != want {
			t.Errorf("dateFormat = %q, want %q", got, want)
		}
	})

	t.Run("first", func(t *testing.T) {
		fn := fm["first"].(func(int, any) any)

		input := []string{"a", "b", "c", "d", "e"}
		got := fn(3, input).([]string)
		if len(got) != 3 {
			t.Fatalf("first(3, ...) returned %d items, want 3", len(got))
		}
		if got[0] != "a" || got[1] != "b" || got[2] != "c" {
			t.Errorf("first(3, ...) = %v, want [a b c]", got)
		}

		// Test with n > len
		got = fn(10, input).([]string)
		if len(got) != 5 {
			t.Errorf("first(10, ...) returned %d items, want 5", len(got))
		}
	})

	t.Run("last", func(t *testing.T) {
		fn := fm["last"].(func(int, any) any)

		input := []string{"a", "b", "c", "d", "e"}
		got := fn(2, input).([]string)
		if len(got) != 2 {
			t.Fatalf("last(2, ...) returned %d items, want 2", len(got))
		}
		if got[0] != "d" || got[1] != "e" {
			t.Errorf("last(2, ...) = %v, want [d e]", got)
		}
	})

	t.Run("relURL", func(t *testing.T) {
		fn := fm["relURL"].(func(string) string)

		tests := []struct {
			input string
			want  string
		}{
			{"/blog/post/", "/blog/post/"},
			{"blog/post/", "/blog/post/"},
			{"/", "/"},
		}
		for _, tt := range tests {
			got := fn(tt.input)
			if got != tt.want {
				t.Errorf("relURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		}
	})

	t.Run("absURL", func(t *testing.T) {
		fn := fm["absURL"].(func(string, string) string)

		got := fn("https://example.com", "/blog/post/")
		want := "https://example.com/blog/post/"
		if got != want {
			t.Errorf("absURL = %q, want %q", got, want)
		}

		got = fn("https://example.com/", "blog/post/")
		want = "https://example.com/blog/post/"
		if got != want {
			t.Errorf("absURL = %q, want %q", got, want)
		}
	})

	t.Run("plainify", func(t *testing.T) {
		fn := fm["plainify"].(func(string) string)
		got := fn("<p>Hello <strong>World</strong></p>")
		want := "Hello World"
		if got != want {
			t.Errorf("plainify = %q, want %q", got, want)
		}
	})

	t.Run("dict", func(t *testing.T) {
		fn := fm["dict"].(func(...any) (map[string]any, error))
		m, err := fn("key1", "val1", "key2", 42)
		if err != nil {
			t.Fatalf("dict failed: %v", err)
		}
		if m["key1"] != "val1" {
			t.Errorf("dict key1 = %v, want val1", m["key1"])
		}
		if m["key2"] != 42 {
			t.Errorf("dict key2 = %v, want 42", m["key2"])
		}

		// Odd number of args should error.
		_, err = fn("key1")
		if err == nil {
			t.Error("dict with odd args should return error")
		}
	})

	t.Run("slice", func(t *testing.T) {
		fn := fm["slice"].(func(...any) []any)
		got := fn("a", "b", "c")
		if len(got) != 3 {
			t.Errorf("slice returned %d items, want 3", len(got))
		}
	})

	t.Run("where", func(t *testing.T) {
		fn := fm["where"].(func(any, string, any) any)
		pages := []*PageContext{
			{Title: "Go Post", Section: "blog"},
			{Title: "Rust Post", Section: "blog"},
			{Title: "About", Section: "pages"},
		}
		got := fn(pages, "Section", "blog").([]*PageContext)
		if len(got) != 2 {
			t.Fatalf("where returned %d items, want 2", len(got))
		}
		if got[0].Title != "Go Post" || got[1].Title != "Rust Post" {
			t.Errorf("where returned unexpected items: %v", got)
		}
	})

	t.Run("sortBy", func(t *testing.T) {
		fn := fm["sortBy"].(func([]*PageContext, string) []*PageContext)
		pages := []*PageContext{
			{Title: "Banana"},
			{Title: "Apple"},
			{Title: "Cherry"},
		}
		sorted := fn(pages, "Title")
		if sorted[0].Title != "Apple" || sorted[1].Title != "Banana" || sorted[2].Title != "Cherry" {
			t.Errorf("sortBy Title returned unexpected order: %v, %v, %v",
				sorted[0].Title, sorted[1].Title, sorted[2].Title)
		}
		// Original should be unchanged.
		if pages[0].Title != "Banana" {
			t.Error("sortBy should not mutate original slice")
		}
	})

	t.Run("groupBy", func(t *testing.T) {
		fn := fm["groupBy"].(func(any, string) map[string]any)
		pages := []*PageContext{
			{Title: "Post 1", Section: "blog"},
			{Title: "Post 2", Section: "blog"},
			{Title: "Project 1", Section: "projects"},
		}
		groups := fn(pages, "Section")
		blogPages := groups["blog"].([]*PageContext)
		if len(blogPages) != 2 {
			t.Errorf("groupBy blog should have 2 items, got %d", len(blogPages))
		}
		projectPages := groups["projects"].([]*PageContext)
		if len(projectPages) != 1 {
			t.Errorf("groupBy projects should have 1 item, got %d", len(projectPages))
		}
	})

	t.Run("markdownify", func(t *testing.T) {
		fn := fm["markdownify"].(func(string) template.HTML)
		got := fn("**bold** text")
		if !strings.Contains(string(got), "<strong>bold</strong>") {
			t.Errorf("markdownify should render bold, got: %s", got)
		}
	})

	t.Run("now", func(t *testing.T) {
		fn := fm["now"].(func() time.Time)
		result := fn()
		if time.Since(result) > time.Second {
			t.Error("now() should return approximately the current time")
		}
	})

	t.Run("shuffle", func(t *testing.T) {
		fn := fm["shuffle"].(func(any) any)
		input := []string{"a", "b", "c", "d", "e"}
		got := fn(input).([]string)
		if len(got) != 5 {
			t.Errorf("shuffle should return same length, got %d", len(got))
		}
		// Verify all elements are present.
		seen := make(map[string]bool)
		for _, v := range got {
			seen[v] = true
		}
		for _, v := range input {
			if !seen[v] {
				t.Errorf("shuffle lost element %q", v)
			}
		}
	})
}

func TestUserOverride(t *testing.T) {
	// Create a minimal theme directory with a single template.
	themeDir := t.TempDir()
	themeLayoutDir := filepath.Join(themeDir, "layouts")
	if err := os.MkdirAll(filepath.Join(themeLayoutDir, "_default"), 0o755); err != nil {
		t.Fatal(err)
	}

	themeContent := `<h1>THEME: {{ .Title }}</h1>`
	if err := os.WriteFile(filepath.Join(themeLayoutDir, "_default", "single.html"), []byte(themeContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a user layout directory that overrides the same template.
	userDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(userDir, "_default"), 0o755); err != nil {
		t.Fatal(err)
	}

	overrideContent := `<article><h1>CUSTOM: {{ .Title }}</h1>{{ .Content }}</article>`
	if err := os.WriteFile(filepath.Join(userDir, "_default", "single.html"), []byte(overrideContent), 0o644); err != nil {
		t.Fatal(err)
	}

	eng, err := NewEngine(themeDir, userDir)
	if err != nil {
		t.Fatalf("NewEngine with user override failed: %v", err)
	}

	ctx := &PageContext{
		Title:   "Override Test",
		Content: template.HTML("<p>Overridden</p>"),
		Site:    &SiteContext{Title: "Test Site"},
	}

	// Execute the single template directly -- the user's version should win.
	out, err := eng.Execute("_default/single.html", ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := string(out)
	if !strings.Contains(output, "CUSTOM:") {
		t.Errorf("output should use user override template, got: %s", output)
	}
	if strings.Contains(output, "THEME:") {
		t.Errorf("output should NOT contain theme content, got: %s", output)
	}
	if !strings.Contains(output, "Override Test") {
		t.Errorf("output should contain title, got: %s", output)
	}
	if !strings.Contains(output, "<article>") {
		t.Errorf("output should contain article tag from override, got: %s", output)
	}
}

func TestHasTemplate(t *testing.T) {
	themePath := testdataThemePath(t)
	eng, err := NewEngine(themePath, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	if !eng.HasTemplate("_default/single.html") {
		t.Error("HasTemplate should return true for existing template")
	}
	if eng.HasTemplate("nonexistent.html") {
		t.Error("HasTemplate should return false for non-existing template")
	}
}

func TestReadFile(t *testing.T) {
	// Create a temp file to read.
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello file"), 0o644); err != nil {
		t.Fatal(err)
	}

	content, err := readFile(filePath)
	if err != nil {
		t.Fatalf("readFile failed: %v", err)
	}
	if content != "hello file" {
		t.Errorf("readFile = %q, want %q", content, "hello file")
	}

	// Non-existent file should error.
	_, err = readFile(filepath.Join(tmp, "nonexistent.txt"))
	if err == nil {
		t.Error("readFile should return error for non-existent file")
	}
}

func TestExecuteListTemplate(t *testing.T) {
	themePath := testdataThemePath(t)
	eng, err := NewEngine(themePath, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	resolved := eng.Resolve("list", "blog", "")
	if resolved != "_default/list.html" {
		t.Errorf("Resolve for list = %q, want %q", resolved, "_default/list.html")
	}
}

func TestNewEngineInvalidPath(t *testing.T) {
	_, err := NewEngine("/nonexistent/path", "")
	if err == nil {
		t.Error("NewEngine should fail with non-existent theme path")
	}
}

func TestResolveTaxonomy(t *testing.T) {
	// Create a temp directory with taxonomy-specific templates.
	tmp := t.TempDir()
	layoutDir := filepath.Join(tmp, "layouts", "_default")
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create taxonomy and terms templates.
	templates := map[string]string{
		"_default/taxonomy.html": `<h1>Taxonomy: {{ .Title }}</h1>`,
		"_default/terms.html":    `<h1>Terms: {{ .Title }}</h1>`,
		"_default/single.html":   `<h1>{{ .Title }}</h1>`,
		"_default/list.html":     `<h1>List: {{ .Title }}</h1>`,
	}
	for name, content := range templates {
		fullPath := filepath.Join(tmp, "layouts", name)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	eng, err := NewEngine(tmp, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	got := eng.Resolve("taxonomy", "tags", "")
	if got != "_default/taxonomy.html" {
		t.Errorf("Resolve taxonomy = %q, want %q", got, "_default/taxonomy.html")
	}

	got = eng.Resolve("taxonomylist", "tags", "")
	if got != "_default/terms.html" {
		t.Errorf("Resolve taxonomylist = %q, want %q", got, "_default/terms.html")
	}
}
