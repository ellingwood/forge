// Package image provides responsive image processing, format conversion,
// and build caching for the Forge static site generator.
package image

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// cacheManifestVersion is bumped when the cache format changes.
const cacheManifestVersion = "1"

// Cache manages processed image variants on disk so that unchanged images
// are not re-processed across builds. All methods are safe for concurrent use.
type Cache struct {
	mu       sync.Mutex
	dir      string        // e.g. .forge/imagecache/
	manifest CacheManifest // loaded from manifest.json
}

// CacheManifest is the top-level structure persisted as manifest.json.
type CacheManifest struct {
	Version string                 `json:"version"`
	Entries map[string]*CacheEntry `json:"entries"` // keyed by source path
}

// CacheEntry records the processing state of a single source image.
type CacheEntry struct {
	ContentHash string          `json:"contentHash"` // SHA-256 of source file
	ModTime     int64           `json:"modTime"`
	Sizes       []int           `json:"sizes"`
	Formats     []string        `json:"formats"`
	Quality     int             `json:"quality"`
	Variants    []CachedVariant `json:"variants"`
}

// CachedVariant describes one generated file stored in the cache directory.
type CachedVariant struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Format   string `json:"format"`
	Filename string `json:"filename"` // just the filename, stored in cache dir
}

// NewCache creates a Cache rooted at cacheDir. If a manifest.json already
// exists there it is loaded; otherwise an empty manifest is initialised.
func NewCache(cacheDir string) (*Cache, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	c := &Cache{
		dir: cacheDir,
		manifest: CacheManifest{
			Version: cacheManifestVersion,
			Entries: make(map[string]*CacheEntry),
		},
	}

	manifestPath := filepath.Join(cacheDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, fmt.Errorf("reading cache manifest: %w", err)
	}

	var m CacheManifest
	if err := json.Unmarshal(data, &m); err != nil {
		// Corrupt manifest — start fresh.
		return c, nil
	}
	if m.Version != cacheManifestVersion {
		// Version mismatch — start fresh.
		return c, nil
	}
	if m.Entries == nil {
		m.Entries = make(map[string]*CacheEntry)
	}
	c.manifest = m
	return c, nil
}

// Lookup checks whether a cached result exists for srcPath that matches the
// given contentHash and processing parameters. On a hit it returns the
// cached variants and true; on a miss it returns nil, false.
func (c *Cache) Lookup(srcPath string, contentHash string, sizes []int, formats []string, quality int) ([]CachedVariant, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.manifest.Entries[srcPath]
	if !ok {
		return nil, false
	}
	if entry.ContentHash != contentHash {
		return nil, false
	}
	if entry.Quality != quality {
		return nil, false
	}
	if !intSliceEqual(entry.Sizes, sizes) {
		return nil, false
	}
	if !stringSliceEqual(entry.Formats, formats) {
		return nil, false
	}
	// Verify all cached files still exist on disk.
	for _, v := range entry.Variants {
		p := filepath.Join(c.dir, v.Filename)
		if _, err := os.Stat(p); err != nil {
			return nil, false
		}
	}
	return entry.Variants, true
}

// Store adds or updates a cache entry for srcPath and persists the manifest.
func (c *Cache) Store(srcPath string, contentHash string, sizes []int, formats []string, quality int, variants []CachedVariant) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.manifest.Entries[srcPath] = &CacheEntry{
		ContentHash: contentHash,
		Sizes:       sizes,
		Formats:     formats,
		Quality:     quality,
		Variants:    variants,
	}
	return c.SaveManifest()
}

// CopyToOutput copies cached variant files from the cache directory into
// outputDir and returns a slice of Variant with URLs constructed from
// urlPrefix.
func (c *Cache) CopyToOutput(variants []CachedVariant, outputDir, urlPrefix string) ([]Variant, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}
	result := make([]Variant, 0, len(variants))
	for _, cv := range variants {
		src := filepath.Join(c.dir, cv.Filename)
		dst := filepath.Join(outputDir, cv.Filename)
		if err := copyFile(src, dst); err != nil {
			return nil, fmt.Errorf("copying cached variant %s: %w", cv.Filename, err)
		}
		url := strings.TrimRight(urlPrefix, "/") + "/" + cv.Filename
		result = append(result, Variant{
			Width:  cv.Width,
			Height: cv.Height,
			Format: cv.Format,
			URL:    url,
			Path:   dst,
		})
	}
	return result, nil
}

// SaveManifest writes the current manifest to manifest.json in the cache
// directory.
func (c *Cache) SaveManifest() error {
	data, err := json.MarshalIndent(c.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling cache manifest: %w", err)
	}
	return os.WriteFile(filepath.Join(c.dir, "manifest.json"), data, 0o644)
}

// HashFile computes the SHA-256 hex digest of the file at path.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// copyFile copies a single file from src to dst, creating parent directories
// as needed.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// intSliceEqual reports whether a and b contain the same ints in the same order.
func intSliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	// Compare sorted copies so ordering doesn't matter.
	ac := make([]int, len(a))
	bc := make([]int, len(b))
	copy(ac, a)
	copy(bc, b)
	sort.Ints(ac)
	sort.Ints(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}

// stringSliceEqual reports whether a and b contain the same strings
// (order-insensitive).
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac := make([]string, len(a))
	bc := make([]string, len(b))
	copy(ac, a)
	copy(bc, b)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}
