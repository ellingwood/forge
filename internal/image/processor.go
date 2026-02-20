package image

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/aellingwood/forge/internal/config"
	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"
)

// Processor generates responsive image variants (resized + format-converted)
// and maintains an in-memory registry of processed images keyed by source URL.
type Processor struct {
	config   config.ImageConfig
	cache    *Cache
	mu       sync.Mutex
	registry map[string]*ProcessedImage // keyed by source URL
}

// ProcessedImage holds metadata about a source image and all generated
// variants.
type ProcessedImage struct {
	OriginalURL string
	Width       int
	Height      int
	Variants    []Variant
}

// Variant describes a single generated image file.
type Variant struct {
	Width  int
	Height int
	Format string // "webp", "jpeg", "png"
	URL    string // URL path for use in HTML
	Path   string // filesystem path
}

// NewProcessor creates a Processor with the given image configuration.
// The build cache is initialised at {projectRoot}/.forge/imagecache/.
func NewProcessor(cfg config.ImageConfig, projectRoot string) *Processor {
	cacheDir := filepath.Join(projectRoot, ".forge", "imagecache")
	cache, err := NewCache(cacheDir)
	if err != nil {
		// If we cannot initialise the cache, create a processor without
		// caching. This is a best-effort optimisation.
		cache = nil
	}
	return &Processor{
		config:   cfg,
		cache:    cache,
		registry: make(map[string]*ProcessedImage),
	}
}

// Process opens the image at srcPath, generates resized variants according to
// the processor's configuration, writes them to outputDir, and registers the
// result under srcURL.
//
// File naming: {stem}-{width}w.{ext} (e.g. hero-640w.webp).
// No upscaling: sizes larger than the source width are silently skipped.
func (p *Processor) Process(srcPath, srcURL, outputDir string) (*ProcessedImage, error) {
	// Open and decode the source image to read dimensions.
	srcImg, err := imaging.Open(srcPath, imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("opening image %s: %w", srcPath, err)
	}
	bounds := srcImg.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Determine which sizes to generate (no upscaling).
	var sizes []int
	for _, s := range p.config.Sizes {
		if s <= srcWidth {
			sizes = append(sizes, s)
		}
	}

	// Determine which formats to generate.
	formats := normalizeFormats(p.config.Formats, srcPath)

	// Build the stem from the source filename.
	stem := fileStem(srcPath)

	// Try the cache first.
	if p.cache != nil {
		hash, hashErr := HashFile(srcPath)
		if hashErr == nil {
			if cached, ok := p.cache.Lookup(srcPath, hash, sizes, formats, p.config.Quality); ok {
				variants, cpErr := p.cache.CopyToOutput(cached, outputDir, urlDir(srcURL))
				if cpErr == nil {
					pi := &ProcessedImage{
						OriginalURL: srcURL,
						Width:       srcWidth,
						Height:      srcHeight,
						Variants:    variants,
					}
					p.register(srcURL, pi)
					return pi, nil
				}
				// Cache copy failed — fall through and regenerate.
			}
		}
	}

	// Ensure output directory exists.
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	var variants []Variant
	var cachedVariants []CachedVariant

	for _, size := range sizes {
		// Resize preserving aspect ratio (height=0 → auto).
		resized := imaging.Resize(srcImg, size, 0, imaging.Lanczos)
		resizedBounds := resized.Bounds()
		resizedHeight := resizedBounds.Dy()

		for _, format := range formats {
			ext := formatExtension(format)
			filename := fmt.Sprintf("%s-%dw.%s", stem, size, ext)
			outPath := filepath.Join(outputDir, filename)

			if err := encodeImage(resized, outPath, format, p.config.Quality); err != nil {
				return nil, fmt.Errorf("encoding %s: %w", outPath, err)
			}

			url := strings.TrimRight(urlDir(srcURL), "/") + "/" + filename

			variants = append(variants, Variant{
				Width:  size,
				Height: resizedHeight,
				Format: format,
				URL:    url,
				Path:   outPath,
			})
			cachedVariants = append(cachedVariants, CachedVariant{
				Width:    size,
				Height:   resizedHeight,
				Format:   format,
				Filename: filename,
			})
		}
	}

	// Store in cache.
	if p.cache != nil {
		hash, hashErr := HashFile(srcPath)
		if hashErr == nil {
			// Copy generated files into cache dir for future builds.
			for _, cv := range cachedVariants {
				src := filepath.Join(outputDir, cv.Filename)
				dst := filepath.Join(p.cache.dir, cv.Filename)
				_ = copyFile(src, dst) // best effort
			}
			_ = p.cache.Store(srcPath, hash, sizes, formats, p.config.Quality, cachedVariants)
		}
	}

	pi := &ProcessedImage{
		OriginalURL: srcURL,
		Width:       srcWidth,
		Height:      srcHeight,
		Variants:    variants,
	}
	p.register(srcURL, pi)
	return pi, nil
}

// ProcessDir walks srcDir, processing every JPEG/PNG image found. Generated
// variants are written to outputDir and URLs are prefixed with urlPrefix.
// Processing is parallelised across runtime.NumCPU() workers.
func (p *Processor) ProcessDir(srcDir, outputDir, urlPrefix string) error {
	type job struct {
		srcPath string
		srcURL  string
		outDir  string
	}

	var jobs []job

	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isSupportedImage(path) {
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		srcURL := strings.TrimRight(urlPrefix, "/") + "/" + filepath.ToSlash(rel)
		outSubDir := filepath.Join(outputDir, filepath.Dir(rel))

		jobs = append(jobs, job{
			srcPath: path,
			srcURL:  srcURL,
			outDir:  outSubDir,
		})
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking source directory %s: %w", srcDir, err)
	}

	if len(jobs) == 0 {
		return nil
	}

	// Bounded worker pool.
	numWorkers := runtime.NumCPU()
	sem := make(chan struct{}, numWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, j := range jobs {
		j := j // capture
		wg.Add(1)
		sem <- struct{}{} // acquire
		go func() {
			defer wg.Done()
			defer func() { <-sem }() // release

			if _, err := p.Process(j.srcPath, j.srcURL, j.outDir); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return firstErr
}

// GetImage returns the ProcessedImage for the given source URL, or nil if it
// has not been processed. This method is safe for concurrent use.
func (p *Processor) GetImage(srcURL string) *ProcessedImage {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.registry[srcURL]
}

// register stores a ProcessedImage in the registry under the given URL.
func (p *Processor) register(srcURL string, pi *ProcessedImage) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.registry[srcURL] = pi
}

// isSupportedImage reports whether the file at path is a JPEG or PNG based
// on its extension.
func isSupportedImage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png":
		return true
	}
	return false
}

// normalizeFormats converts config format strings (e.g. "webp", "original")
// into concrete format names. "original" is replaced with the source file's
// format.
func normalizeFormats(configFormats []string, srcPath string) []string {
	srcFmt := sourceFormat(srcPath)
	var formats []string
	seen := make(map[string]bool)
	for _, f := range configFormats {
		f = strings.ToLower(f)
		if f == "original" {
			f = srcFmt
		}
		if !seen[f] {
			seen[f] = true
			formats = append(formats, f)
		}
	}
	return formats
}

// sourceFormat returns "jpeg" or "png" based on the file extension.
func sourceFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "png"
	default:
		return "jpeg"
	}
}

// formatExtension returns the file extension (without dot) for a format name.
func formatExtension(format string) string {
	switch format {
	case "webp":
		return "webp"
	case "png":
		return "png"
	default:
		return "jpg"
	}
}

// fileStem returns the filename without its extension.
func fileStem(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// urlDir returns the directory portion of a URL path.
func urlDir(u string) string {
	idx := strings.LastIndex(u, "/")
	if idx < 0 {
		return ""
	}
	return u[:idx]
}

// encodeImage writes img to outPath in the specified format.
func encodeImage(img image.Image, outPath, format string, quality int) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	switch format {
	case "webp":
		opts := webp.Options{Quality: quality}
		if err := webp.Encode(f, img, opts); err != nil {
			return fmt.Errorf("encoding webp: %w", err)
		}
	case "png":
		if err := png.Encode(f, img); err != nil {
			return fmt.Errorf("encoding png: %w", err)
		}
	default: // jpeg
		if err := jpeg.Encode(f, img, &jpeg.Options{Quality: quality}); err != nil {
			return fmt.Errorf("encoding jpeg: %w", err)
		}
	}
	return f.Close()
}
