package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// LoadDataFiles walks the given directory and parses all YAML (.yaml, .yml),
// JSON (.json), and TOML (.toml) files into a nested map structure.
// The map key is the filename without extension. Nested directories create
// nested maps: data/people/team.yaml -> result["people"]["team"].
// Returns an empty map (not nil) if the directory doesn't exist.
func LoadDataFiles(dataDir string) (map[string]any, error) {
	result := make(map[string]any)

	// If the directory doesn't exist, return an empty map with no error.
	if _, err := os.Stat(dataDir); errors.Is(err, fs.ErrNotExist) {
		return result, nil
	}

	err := filepath.WalkDir(dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" && ext != ".toml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading data file %s: %w", path, err)
		}

		var parsed any
		switch ext {
		case ".yaml", ".yml":
			if err := yaml.Unmarshal(data, &parsed); err != nil {
				return fmt.Errorf("parsing %s: %w", path, err)
			}
		case ".json":
			if err := json.Unmarshal(data, &parsed); err != nil {
				return fmt.Errorf("parsing %s: %w", path, err)
			}
		case ".toml":
			if err := toml.Unmarshal(data, &parsed); err != nil {
				return fmt.Errorf("parsing %s: %w", path, err)
			}
		}

		// Compute relative path from dataDir.
		relPath, err := filepath.Rel(dataDir, path)
		if err != nil {
			return fmt.Errorf("computing relative path for %s: %w", path, err)
		}
		relPath = filepath.ToSlash(relPath)

		// Split into directory components and filename (without extension).
		parts := strings.Split(relPath, "/")
		name := strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(parts[len(parts)-1]))
		parts[len(parts)-1] = name

		// Navigate into nested maps, creating intermediate maps as needed.
		current := result
		for i := 0; i < len(parts)-1; i++ {
			key := parts[i]
			if existing, ok := current[key]; ok {
				if m, ok := existing.(map[string]any); ok {
					current = m
				} else {
					// A file and a directory share the same name; overwrite.
					m := make(map[string]any)
					current[key] = m
					current = m
				}
			} else {
				m := make(map[string]any)
				current[key] = m
				current = m
			}
		}

		// Set the final key.
		current[parts[len(parts)-1]] = parsed

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
