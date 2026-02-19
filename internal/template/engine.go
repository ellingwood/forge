package template

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

// Engine wraps Go's html/template with layout resolution, custom functions,
// and theme/user layout overlaying.
type Engine struct {
	templates *template.Template
	funcMap   template.FuncMap
}

// NewEngine creates a template Engine by loading .html files from the theme
// layouts directory and optionally overlaying user layout files on top. User
// layouts with the same relative path override theme layouts.
func NewEngine(themePath, userLayoutPath string) (*Engine, error) {
	e := &Engine{
		funcMap: FuncMap(),
	}

	// Create root template with our custom functions.
	e.templates = template.New("").Funcs(e.funcMap)

	// Collect template files: theme first, user overrides second.
	themeLayoutDir := filepath.Join(themePath, "layouts")
	files, err := collectTemplateFiles(themeLayoutDir)
	if err != nil {
		return nil, fmt.Errorf("loading theme templates from %s: %w", themeLayoutDir, err)
	}

	// If user layout path is provided, overlay user templates.
	if userLayoutPath != "" {
		userFiles, err := collectTemplateFiles(userLayoutPath)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading user templates from %s: %w", userLayoutPath, err)
		}
		// User files override theme files with the same name.
		maps.Copy(files, userFiles)
	}

	// Parse all collected template files.
	for name, filePath := range files {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("reading template %s: %w", filePath, err)
		}
		t := e.templates.New(name)
		if _, err := t.Parse(string(content)); err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", name, err)
		}
	}

	// Wire up the partial function now that all templates are parsed.
	e.funcMap["partial"] = func(name string, ctx any) (template.HTML, error) {
		return e.executePartial(name, ctx)
	}
	// Re-register the func map so the partial function is available.
	// We need to re-parse because Go templates bind functions at parse time.
	// Instead, we rebuild from scratch with the partial function in place.
	root := template.New("").Funcs(e.funcMap)
	for name, filePath := range files {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("reading template %s: %w", filePath, err)
		}
		t := root.New(name)
		if _, err := t.Parse(string(content)); err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", name, err)
		}
	}
	e.templates = root

	return e, nil
}

// executePartial executes a partial template and returns the rendered HTML.
func (e *Engine) executePartial(name string, ctx any) (template.HTML, error) {
	// Look for the partial template. Try with and without "partials/" prefix.
	tmplName := name
	if !strings.HasPrefix(name, "partials/") {
		tmplName = "partials/" + name
	}

	t := e.templates.Lookup(tmplName)
	if t == nil {
		// Try without the prefix in case user specified the full path.
		t = e.templates.Lookup(name)
	}
	if t == nil {
		return "", fmt.Errorf("partial template %q not found", name)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("executing partial %q: %w", name, err)
	}
	return template.HTML(buf.String()), nil
}

// collectTemplateFiles walks a directory and returns a map of template name
// (relative path) to absolute file path for all .html files.
func collectTemplateFiles(dir string) (map[string]string, error) {
	files := make(map[string]string)

	info, err := os.Stat(dir)
	if err != nil {
		return files, err
	}
	if !info.IsDir() {
		return files, fmt.Errorf("%s is not a directory", dir)
	}

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}
		// Template name is the path relative to the layouts directory.
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		// Normalize to forward slashes for consistent template names.
		name := filepath.ToSlash(rel)
		files[name] = path
		return nil
	})

	return files, err
}

// Resolve returns the name of the first matching template for the given page
// type, section, and layout, following the layout resolution order described
// in the spec. If no matching template is found, an empty string is returned.
func (e *Engine) Resolve(pageType, section, layout string) string {
	var candidates []string

	switch pageType {
	case "single":
		if layout != "" {
			candidates = append(candidates, section+"/"+layout+".html")
		}
		candidates = append(candidates, section+"/single.html")
		if layout != "" {
			candidates = append(candidates, "_default/"+layout+".html")
		}
		candidates = append(candidates, "_default/single.html")

	case "list":
		candidates = append(candidates,
			section+"/list.html",
			"_default/list.html",
		)

	case "home":
		candidates = append(candidates,
			"index.html",
			"_default/list.html",
		)

	case "taxonomy":
		candidates = append(candidates,
			section+"/taxonomy.html",
			"_default/taxonomy.html",
			"_default/list.html",
		)

	case "taxonomylist":
		candidates = append(candidates,
			section+"/terms.html",
			"_default/terms.html",
			"_default/list.html",
		)
	}

	for _, name := range candidates {
		if e.templates.Lookup(name) != nil {
			return name
		}
	}
	return ""
}

// Execute renders the named template with the given PageContext and returns
// the output bytes.
func (e *Engine) Execute(templateName string, ctx *PageContext) ([]byte, error) {
	t := e.templates.Lookup(templateName)
	if t == nil {
		return nil, fmt.Errorf("template %q not found", templateName)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return nil, fmt.Errorf("executing template %q: %w", templateName, err)
	}
	return buf.Bytes(), nil
}

// HasTemplate reports whether a template with the given name exists.
func (e *Engine) HasTemplate(name string) bool {
	return e.templates.Lookup(name) != nil
}
