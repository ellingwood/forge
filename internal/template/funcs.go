package template

import (
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// FuncMap returns the custom template functions available to all Forge templates.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		// String functions
		"markdownify": markdownify,
		"plainify":    plainify,
		"truncate":    truncate,
		"slugify":     slugify,
		"safeHTML":    safeHTML,

		// Collection functions
		"first":   first,
		"last":    last,
		"where":   where,
		"sortBy":  sortBy,
		"shuffle": shuffle,
		"groupBy": groupBy,

		// Date functions
		"dateFormat": dateFormat,
		"now":        now,

		// URL functions
		"relURL": relURL,
		"absURL": absURL,

		// Data functions
		"readFile": readFile,

		// Helpers
		"dict":  dict,
		"slice": sliceHelper,

		// Partial helper — registered here for the func map; actual implementation
		// is overridden in Engine after templates are parsed.
		"partial": func(name string, ctx any) template.HTML {
			return ""
		},
	}
}

// --- String functions ---

// markdownify renders inline markdown to HTML. For now this does a simple
// pass-through returning the string marked as safe HTML. A full goldmark-based
// implementation can be wired in later.
func markdownify(s string) template.HTML {
	// Simple inline markdown: bold, italic, code, links
	result := s

	// Bold: **text** or __text__
	boldRe := regexp.MustCompile(`\*\*(.+?)\*\*|__(.+?)__`)
	result = boldRe.ReplaceAllStringFunc(result, func(m string) string {
		inner := strings.TrimPrefix(strings.TrimSuffix(m, "**"), "**")
		if strings.HasPrefix(m, "__") {
			inner = strings.TrimPrefix(strings.TrimSuffix(m, "__"), "__")
		}
		return "<strong>" + inner + "</strong>"
	})

	// Italic: *text* or _text_
	italicRe := regexp.MustCompile(`\*(.+?)\*|_(.+?)_`)
	result = italicRe.ReplaceAllStringFunc(result, func(m string) string {
		inner := strings.TrimPrefix(strings.TrimSuffix(m, "*"), "*")
		if strings.HasPrefix(m, "_") {
			inner = strings.TrimPrefix(strings.TrimSuffix(m, "_"), "_")
		}
		return "<em>" + inner + "</em>"
	})

	// Inline code: `text`
	codeRe := regexp.MustCompile("`(.+?)`")
	result = codeRe.ReplaceAllString(result, "<code>$1</code>")

	return template.HTML(result)
}

// plainify strips HTML tags from a string.
func plainify(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(s, "")
}

// truncate truncates a string to n characters, appending "..." if truncated.
func truncate(n int, s string) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-3]) + "..."
}

// slugify converts a string to a URL-safe slug: lowercase, hyphens for
// spaces/special chars, stripped of non-alphanumeric characters.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen && b.Len() > 0 {
			b.WriteRune('-')
			prevHyphen = true
		}
	}
	result := strings.TrimRight(b.String(), "-")
	return result
}

// safeHTML marks a string as safe HTML so Go templates will not escape it.
func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

// --- Collection functions ---

// first returns the first n items from a slice. If the slice has fewer than n
// items, the full slice is returned.
func first(n int, items any) any {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return items
	}
	if n > v.Len() {
		n = v.Len()
	}
	if n < 0 {
		n = 0
	}
	return v.Slice(0, n).Interface()
}

// last returns the last n items from a slice.
func last(n int, items any) any {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return items
	}
	if n > v.Len() {
		n = v.Len()
	}
	if n < 0 {
		n = 0
	}
	return v.Slice(v.Len()-n, v.Len()).Interface()
}

// where filters a slice of structs/pointers-to-structs, returning only those
// items whose field named key equals value.
func where(items any, key string, value any) any {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return items
	}

	resultSlice := reflect.MakeSlice(v.Type(), 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		field := item
		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}
		if field.Kind() == reflect.Struct {
			f := field.FieldByName(key)
			if f.IsValid() && fmt.Sprintf("%v", f.Interface()) == fmt.Sprintf("%v", value) {
				resultSlice = reflect.Append(resultSlice, v.Index(i))
			}
		}
	}
	return resultSlice.Interface()
}

// sortBy sorts a []*PageContext slice by a given field name. Supported fields
// are "Title", "Date", "WordCount", and "ReadingTime". Unknown fields return
// the slice unchanged.
func sortBy(items []*PageContext, field string) []*PageContext {
	sorted := make([]*PageContext, len(items))
	copy(sorted, items)

	sort.SliceStable(sorted, func(i, j int) bool {
		switch field {
		case "Title":
			return strings.ToLower(sorted[i].Title) < strings.ToLower(sorted[j].Title)
		case "Date":
			return sorted[i].Date.Before(sorted[j].Date)
		case "WordCount":
			return sorted[i].WordCount < sorted[j].WordCount
		case "ReadingTime":
			return sorted[i].ReadingTime < sorted[j].ReadingTime
		default:
			return false
		}
	})
	return sorted
}

// shuffle returns a new slice with the elements in random order.
func shuffle(items any) any {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return items
	}

	result := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
	reflect.Copy(result, v)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := result.Len() - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		tmp := result.Index(i).Interface()
		result.Index(i).Set(result.Index(j))
		result.Index(j).Set(reflect.ValueOf(tmp))
	}
	return result.Interface()
}

// groupBy groups a slice of structs/pointers-to-structs by the value of the
// named field, returning a map from field-value-as-string to a slice of items.
func groupBy(items any, key string) map[string]any {
	v := reflect.ValueOf(items)
	result := make(map[string]any)
	if v.Kind() != reflect.Slice {
		return result
	}

	groups := make(map[string]reflect.Value)
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		field := item
		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}
		if field.Kind() != reflect.Struct {
			continue
		}
		f := field.FieldByName(key)
		if !f.IsValid() {
			continue
		}
		k := fmt.Sprintf("%v", f.Interface())
		if _, ok := groups[k]; !ok {
			groups[k] = reflect.MakeSlice(v.Type(), 0, 0)
		}
		groups[k] = reflect.Append(groups[k], v.Index(i))
	}

	for k, gv := range groups {
		result[k] = gv.Interface()
	}
	return result
}

// --- Date functions ---

// dateFormat formats a time.Time value using the given Go time layout string.
func dateFormat(layout string, t time.Time) string {
	return t.Format(layout)
}

// now returns the current time.
func now() time.Time {
	return time.Now()
}

// --- URL functions ---

// relURL ensures a path has a leading slash.
func relURL(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

// absURL combines a base URL and a path into an absolute URL.
func absURL(baseURL, path string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return baseURL + path
}

// --- Data functions ---

// readFile reads and returns the contents of a file. The path is resolved
// relative to the current working directory.
func readFile(path string) (string, error) {
	absPath := path
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("readFile: getting working directory: %w", err)
		}
		absPath = filepath.Join(wd, path)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("readFile: %w", err)
	}
	return string(data), nil
}

// --- Helpers ---

// dict creates a map[string]any from alternating key-value pairs.
// Example usage in templates: {{ dict "key1" "val1" "key2" "val2" }}
func dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict: odd number of arguments")
	}
	m := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key at position %d is not a string", i)
		}
		m[key] = values[i+1]
	}
	return m, nil
}

// sliceHelper creates a slice from its arguments.
// Registered as "slice" in the template func map.
func sliceHelper(values ...any) []any {
	return values
}
