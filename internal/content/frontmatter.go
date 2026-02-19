package content

import (
	"bytes"
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// Frontmatter delimiters.
var (
	yamlDelimiter = []byte("---")
	tomlDelimiter = []byte("+++")
)

// Date formats supported for parsing date fields in frontmatter.
var dateFormats = []string{
	"2006-01-02",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05-07:00",
	time.RFC3339,
}

// ParseFrontmatter detects and parses frontmatter from raw content bytes.
// It supports YAML (--- delimiters) and TOML (+++ delimiters).
// It returns the parsed metadata as a map, the remaining body content, and any
// error encountered during parsing.
//
// If no frontmatter delimiters are found, it returns nil metadata, the full
// content as body, and no error.
func ParseFrontmatter(raw []byte) (metadata map[string]any, body []byte, err error) {
	trimmed := bytes.TrimLeft(raw, " \t\n\r")

	var delimiter []byte
	var format string

	switch {
	case bytes.HasPrefix(trimmed, yamlDelimiter):
		delimiter = yamlDelimiter
		format = "yaml"
	case bytes.HasPrefix(trimmed, tomlDelimiter):
		delimiter = tomlDelimiter
		format = "toml"
	default:
		// No frontmatter found.
		return nil, raw, nil
	}

	// Find the end of the opening delimiter line.
	rest := trimmed[len(delimiter):]
	// Skip to end of the delimiter line (past any trailing whitespace/newline).
	nlIdx := bytes.IndexByte(rest, '\n')
	if nlIdx == -1 {
		// Only the opening delimiter, no closing one.
		return nil, raw, nil
	}
	rest = rest[nlIdx+1:]

	// Find the closing delimiter.
	before, after, ok := bytes.Cut(rest, delimiter)
	if !ok {
		return nil, raw, fmt.Errorf("closing frontmatter delimiter %q not found", string(delimiter))
	}

	frontmatterContent := before
	afterClosing := after

	// Skip to end of closing delimiter line.
	nlIdx = bytes.IndexByte(afterClosing, '\n')
	if nlIdx == -1 {
		body = nil
	} else {
		body = afterClosing[nlIdx+1:]
	}

	// Handle empty frontmatter.
	if len(bytes.TrimSpace(frontmatterContent)) == 0 {
		return make(map[string]any), body, nil
	}

	metadata = make(map[string]any)

	switch format {
	case "yaml":
		if err := yaml.Unmarshal(frontmatterContent, &metadata); err != nil {
			return nil, nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
		}
	case "toml":
		if err := toml.Unmarshal(frontmatterContent, &metadata); err != nil {
			return nil, nil, fmt.Errorf("failed to parse TOML frontmatter: %w", err)
		}
	}

	return metadata, body, nil
}

// PopulatePage maps metadata fields from parsed frontmatter into the
// corresponding fields on a Page struct. It returns an error if the required
// "title" field is missing or empty.
func PopulatePage(page *Page, metadata map[string]any) error {
	// Title is required.
	titleVal, ok := metadata["title"]
	if !ok {
		return fmt.Errorf("frontmatter: required field \"title\" is missing")
	}
	title, ok := titleVal.(string)
	if !ok || title == "" {
		return fmt.Errorf("frontmatter: required field \"title\" must be a non-empty string")
	}
	page.Title = title

	// String fields.
	if v, ok := metadata["slug"]; ok {
		if s, ok := v.(string); ok {
			page.Slug = s
		}
	}
	if v, ok := metadata["description"]; ok {
		if s, ok := v.(string); ok {
			page.Description = s
		}
	}
	if v, ok := metadata["summary"]; ok {
		if s, ok := v.(string); ok {
			page.Summary = s
		}
	}
	if v, ok := metadata["layout"]; ok {
		if s, ok := v.(string); ok {
			page.Layout = s
		}
	}
	if v, ok := metadata["author"]; ok {
		if s, ok := v.(string); ok {
			page.Author = s
		}
	}
	if v, ok := metadata["series"]; ok {
		if s, ok := v.(string); ok {
			page.Series = s
		}
	}

	// Boolean fields.
	if v, ok := metadata["draft"]; ok {
		if b, ok := v.(bool); ok {
			page.Draft = b
		}
	}

	// Date fields.
	if v, ok := metadata["date"]; ok {
		t, err := parseDate(v)
		if err != nil {
			return fmt.Errorf("frontmatter: invalid \"date\": %w", err)
		}
		page.Date = t
	}
	if v, ok := metadata["lastmod"]; ok {
		t, err := parseDate(v)
		if err != nil {
			return fmt.Errorf("frontmatter: invalid \"lastmod\": %w", err)
		}
		page.Lastmod = t
	}
	if v, ok := metadata["expiryDate"]; ok {
		t, err := parseDate(v)
		if err != nil {
			return fmt.Errorf("frontmatter: invalid \"expiryDate\": %w", err)
		}
		page.ExpiryDate = t
	}

	// Weight (int or float64).
	if v, ok := metadata["weight"]; ok {
		w, err := toInt(v)
		if err != nil {
			return fmt.Errorf("frontmatter: invalid \"weight\": %w", err)
		}
		page.Weight = w
	}

	// String slice fields.
	if v, ok := metadata["tags"]; ok {
		s, err := toStringSlice(v)
		if err != nil {
			return fmt.Errorf("frontmatter: invalid \"tags\": %w", err)
		}
		page.Tags = s
	}
	if v, ok := metadata["categories"]; ok {
		s, err := toStringSlice(v)
		if err != nil {
			return fmt.Errorf("frontmatter: invalid \"categories\": %w", err)
		}
		page.Categories = s
	}
	if v, ok := metadata["aliases"]; ok {
		s, err := toStringSlice(v)
		if err != nil {
			return fmt.Errorf("frontmatter: invalid \"aliases\": %w", err)
		}
		page.Aliases = s
	}

	// Cover image.
	if v, ok := metadata["cover"]; ok {
		cover, err := parseCoverImage(v)
		if err != nil {
			return fmt.Errorf("frontmatter: invalid \"cover\": %w", err)
		}
		page.Cover = cover
	}

	// Params.
	if v, ok := metadata["params"]; ok {
		if m, ok := v.(map[string]any); ok {
			page.Params = m
		}
	}

	return nil
}

// parseDate attempts to parse a date value that may be a string or a
// time.Time (some YAML/TOML parsers auto-detect dates).
func parseDate(v any) (time.Time, error) {
	switch val := v.(type) {
	case time.Time:
		return val, nil
	case string:
		for _, format := range dateFormats {
			if t, err := time.Parse(format, val); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("unable to parse date string %q", val)
	default:
		return time.Time{}, fmt.Errorf("unsupported date type %T", v)
	}
}

// toStringSlice converts a value to a []string. It handles both []string
// (from some parsers) and []any (common from YAML/TOML parsers).
func toStringSlice(v any) ([]string, error) {
	switch val := v.(type) {
	case []string:
		return val, nil
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("expected string in slice, got %T", item)
			}
			result = append(result, s)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("expected string slice, got %T", v)
	}
}

// toInt converts a numeric value to int. It handles int, int64, float64,
// and other common numeric types returned by YAML/TOML parsers.
func toInt(v any) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case float64:
		return int(val), nil
	default:
		return 0, fmt.Errorf("expected numeric type, got %T", v)
	}
}

// parseCoverImage converts a map value into a CoverImage struct.
func parseCoverImage(v any) (*CoverImage, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map, got %T", v)
	}

	cover := &CoverImage{}
	if img, ok := m["image"]; ok {
		if s, ok := img.(string); ok {
			cover.Image = s
		}
	}
	if alt, ok := m["alt"]; ok {
		if s, ok := alt.(string); ok {
			cover.Alt = s
		}
	}
	if caption, ok := m["caption"]; ok {
		if s, ok := caption.(string); ok {
			cover.Caption = s
		}
	}

	return cover, nil
}
