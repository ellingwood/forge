package image

import (
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/aellingwood/forge/internal/config"
)

// createTestJPEG writes a plain-colour JPEG of the given dimensions to path.
func createTestJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
}

// createTestPNG writes a plain-colour PNG of the given dimensions to path.
func createTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 50, G: 100, B: 150, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------
// Cache tests
// ---------------------------------------------------------------

func TestNewCache_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	c, err := NewCache(cacheDir)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	if c.manifest.Version != cacheManifestVersion {
		t.Errorf("version = %q; want %q", c.manifest.Version, cacheManifestVersion)
	}
	if len(c.manifest.Entries) != 0 {
		t.Errorf("entries = %d; want 0", len(c.manifest.Entries))
	}
}

func TestNewCache_LoadExisting(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a valid manifest.
	m := CacheManifest{
		Version: cacheManifestVersion,
		Entries: map[string]*CacheEntry{
			"/img/hero.jpg": {
				ContentHash: "abc123",
				Quality:     75,
				Sizes:       []int{320, 640},
				Formats:     []string{"webp", "jpeg"},
			},
		},
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(cacheDir, "manifest.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := NewCache(cacheDir)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	if len(c.manifest.Entries) != 1 {
		t.Errorf("entries = %d; want 1", len(c.manifest.Entries))
	}
}

func TestNewCache_CorruptManifest(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write invalid JSON.
	if err := os.WriteFile(filepath.Join(cacheDir, "manifest.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := NewCache(cacheDir)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	// Should start fresh.
	if len(c.manifest.Entries) != 0 {
		t.Errorf("entries = %d; want 0 (fresh start)", len(c.manifest.Entries))
	}
}

func TestCache_LookupMiss(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := c.Lookup("nonexistent.jpg", "hash", []int{320}, []string{"webp"}, 75)
	if ok {
		t.Error("expected cache miss")
	}
}

func TestCache_StoreAndLookup(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy cached file.
	dummyFile := filepath.Join(dir, "hero-320w.webp")
	if err := os.WriteFile(dummyFile, []byte("fake webp"), 0o644); err != nil {
		t.Fatal(err)
	}

	variants := []CachedVariant{
		{Width: 320, Height: 200, Format: "webp", Filename: "hero-320w.webp"},
	}

	if err := c.Store("hero.jpg", "hash1", []int{320}, []string{"webp"}, 75, variants); err != nil {
		t.Fatal(err)
	}

	got, ok := c.Lookup("hero.jpg", "hash1", []int{320}, []string{"webp"}, 75)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(got) != 1 || got[0].Filename != "hero-320w.webp" {
		t.Errorf("unexpected variants: %+v", got)
	}
}

func TestCache_InvalidateOnHashChange(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Create the cached file so the file-existence check passes.
	if err := os.WriteFile(filepath.Join(dir, "hero-320w.webp"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	variants := []CachedVariant{
		{Width: 320, Height: 200, Format: "webp", Filename: "hero-320w.webp"},
	}
	_ = c.Store("hero.jpg", "oldhash", []int{320}, []string{"webp"}, 75, variants)

	// Different hash should miss.
	_, ok := c.Lookup("hero.jpg", "newhash", []int{320}, []string{"webp"}, 75)
	if ok {
		t.Error("expected cache miss on hash change")
	}
}

func TestCache_InvalidateOnConfigChange(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hero-320w.webp"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	variants := []CachedVariant{
		{Width: 320, Height: 200, Format: "webp", Filename: "hero-320w.webp"},
	}
	_ = c.Store("hero.jpg", "hash1", []int{320}, []string{"webp"}, 75, variants)

	// Different quality.
	_, ok := c.Lookup("hero.jpg", "hash1", []int{320}, []string{"webp"}, 80)
	if ok {
		t.Error("expected miss on quality change")
	}

	// Different sizes.
	_, ok = c.Lookup("hero.jpg", "hash1", []int{320, 640}, []string{"webp"}, 75)
	if ok {
		t.Error("expected miss on sizes change")
	}

	// Different formats.
	_, ok = c.Lookup("hero.jpg", "hash1", []int{320}, []string{"webp", "jpeg"}, 75)
	if ok {
		t.Error("expected miss on formats change")
	}
}

func TestCache_CopyToOutput(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Write a file into the cache dir.
	if err := os.WriteFile(filepath.Join(dir, "hero-320w.webp"), []byte("cached data"), 0o644); err != nil {
		t.Fatal(err)
	}

	variants := []CachedVariant{
		{Width: 320, Height: 200, Format: "webp", Filename: "hero-320w.webp"},
	}

	outDir := filepath.Join(t.TempDir(), "output")
	result, err := c.CopyToOutput(variants, outDir, "/images")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(result))
	}
	if result[0].URL != "/images/hero-320w.webp" {
		t.Errorf("URL = %q; want %q", result[0].URL, "/images/hero-320w.webp")
	}
	// Verify file exists in output.
	if _, err := os.Stat(filepath.Join(outDir, "hero-320w.webp")); err != nil {
		t.Errorf("expected file in output dir: %v", err)
	}
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// SHA-256 of "hello" is well known.
	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if h != expected {
		t.Errorf("hash = %q; want %q", h, expected)
	}
}

// ---------------------------------------------------------------
// Processor tests
// ---------------------------------------------------------------

func TestProcess_JPEG_NoUpscaling(t *testing.T) {
	projectRoot := t.TempDir()
	srcDir := filepath.Join(projectRoot, "static", "images")
	outputDir := filepath.Join(projectRoot, "public", "images")

	// Create a 800x600 JPEG.
	srcPath := filepath.Join(srcDir, "hero.jpg")
	createTestJPEG(t, srcPath, 800, 600)

	cfg := config.ImageConfig{
		Enabled: true,
		Quality: 75,
		Sizes:   []int{320, 640, 960, 1280}, // 960 and 1280 should be skipped
		Formats: []string{"webp", "original"},
	}

	proc := NewProcessor(cfg, projectRoot)
	pi, err := proc.Process(srcPath, "/images/hero.jpg", outputDir)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	if pi.Width != 800 || pi.Height != 600 {
		t.Errorf("dimensions = %dx%d; want 800x600", pi.Width, pi.Height)
	}

	// Should have 2 sizes (320, 640) x 2 formats (webp, jpeg) = 4 variants.
	if len(pi.Variants) != 4 {
		t.Fatalf("variants = %d; want 4", len(pi.Variants))
	}

	// Verify filenames.
	var filenames []string
	for _, v := range pi.Variants {
		filenames = append(filenames, filepath.Base(v.Path))
	}
	sort.Strings(filenames)
	expected := []string{"hero-320w.jpg", "hero-320w.webp", "hero-640w.jpg", "hero-640w.webp"}
	sort.Strings(expected)
	for i, fn := range expected {
		if filenames[i] != fn {
			t.Errorf("filename[%d] = %q; want %q", i, filenames[i], fn)
		}
	}

	// Verify files exist on disk.
	for _, v := range pi.Variants {
		if _, err := os.Stat(v.Path); err != nil {
			t.Errorf("variant file missing: %s", v.Path)
		}
	}
}

func TestProcess_PNG(t *testing.T) {
	projectRoot := t.TempDir()
	srcDir := filepath.Join(projectRoot, "static", "images")
	outputDir := filepath.Join(projectRoot, "public", "images")

	srcPath := filepath.Join(srcDir, "logo.png")
	createTestPNG(t, srcPath, 500, 500)

	cfg := config.ImageConfig{
		Enabled: true,
		Quality: 80,
		Sizes:   []int{320},
		Formats: []string{"webp", "original"},
	}

	proc := NewProcessor(cfg, projectRoot)
	pi, err := proc.Process(srcPath, "/images/logo.png", outputDir)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	// 1 size x 2 formats (webp, png) = 2 variants.
	if len(pi.Variants) != 2 {
		t.Fatalf("variants = %d; want 2", len(pi.Variants))
	}

	// One should be png, one webp.
	formatSet := make(map[string]bool)
	for _, v := range pi.Variants {
		formatSet[v.Format] = true
	}
	if !formatSet["webp"] || !formatSet["png"] {
		t.Errorf("expected webp and png formats, got %v", formatSet)
	}
}

func TestProcess_ExactWidthMatch(t *testing.T) {
	projectRoot := t.TempDir()
	srcDir := filepath.Join(projectRoot, "static")
	outputDir := filepath.Join(projectRoot, "public")

	// Image exactly 640px wide — 640 should be generated, 960 skipped.
	srcPath := filepath.Join(srcDir, "photo.jpg")
	createTestJPEG(t, srcPath, 640, 480)

	cfg := config.ImageConfig{
		Enabled: true,
		Quality: 75,
		Sizes:   []int{320, 640, 960},
		Formats: []string{"original"},
	}

	proc := NewProcessor(cfg, projectRoot)
	pi, err := proc.Process(srcPath, "/photo.jpg", outputDir)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	// 2 sizes (320, 640) x 1 format = 2 variants.
	if len(pi.Variants) != 2 {
		t.Fatalf("variants = %d; want 2", len(pi.Variants))
	}
}

func TestGetImage_ThreadSafe(t *testing.T) {
	projectRoot := t.TempDir()
	cfg := config.ImageConfig{Enabled: true, Quality: 75}
	proc := NewProcessor(cfg, projectRoot)

	// Should return nil for unregistered image.
	if got := proc.GetImage("/images/missing.jpg"); got != nil {
		t.Errorf("expected nil for unregistered image, got %v", got)
	}

	// Register and retrieve.
	srcDir := filepath.Join(projectRoot, "static")
	outputDir := filepath.Join(projectRoot, "public")
	srcPath := filepath.Join(srcDir, "test.jpg")
	createTestJPEG(t, srcPath, 400, 300)

	cfg.Sizes = []int{320}
	cfg.Formats = []string{"original"}
	proc = NewProcessor(cfg, projectRoot)

	_, err := proc.Process(srcPath, "/images/test.jpg", outputDir)
	if err != nil {
		t.Fatal(err)
	}

	got := proc.GetImage("/images/test.jpg")
	if got == nil {
		t.Fatal("expected registered image")
	}
	if got.OriginalURL != "/images/test.jpg" {
		t.Errorf("OriginalURL = %q; want %q", got.OriginalURL, "/images/test.jpg")
	}
}

func TestProcessDir(t *testing.T) {
	projectRoot := t.TempDir()
	srcDir := filepath.Join(projectRoot, "static", "img")
	outputDir := filepath.Join(projectRoot, "public", "img")

	createTestJPEG(t, filepath.Join(srcDir, "a.jpg"), 800, 600)
	createTestPNG(t, filepath.Join(srcDir, "sub", "b.png"), 500, 500)
	// Create a non-image file that should be skipped.
	if err := os.WriteFile(filepath.Join(srcDir, "readme.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create an SVG that should be skipped.
	if err := os.WriteFile(filepath.Join(srcDir, "icon.svg"), []byte("<svg></svg>"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.ImageConfig{
		Enabled: true,
		Quality: 75,
		Sizes:   []int{320},
		Formats: []string{"original"},
	}
	proc := NewProcessor(cfg, projectRoot)

	if err := proc.ProcessDir(srcDir, outputDir, "/img"); err != nil {
		t.Fatalf("ProcessDir: %v", err)
	}

	// Both images should be registered.
	if proc.GetImage("/img/a.jpg") == nil {
		t.Error("a.jpg not registered")
	}
	if proc.GetImage("/img/sub/b.png") == nil {
		t.Error("sub/b.png not registered")
	}
}

func TestProcess_CacheHit(t *testing.T) {
	projectRoot := t.TempDir()
	srcDir := filepath.Join(projectRoot, "static")
	outputDir := filepath.Join(projectRoot, "public")

	srcPath := filepath.Join(srcDir, "cached.jpg")
	createTestJPEG(t, srcPath, 800, 600)

	cfg := config.ImageConfig{
		Enabled: true,
		Quality: 75,
		Sizes:   []int{320},
		Formats: []string{"original"},
	}

	// First pass: populate cache.
	proc1 := NewProcessor(cfg, projectRoot)
	pi1, err := proc1.Process(srcPath, "/images/cached.jpg", outputDir)
	if err != nil {
		t.Fatal(err)
	}

	// Remove output dir to prove the second pass uses cache.
	if err := os.RemoveAll(outputDir); err != nil {
		t.Fatal(err)
	}

	// Second pass: should use cache.
	proc2 := NewProcessor(cfg, projectRoot)
	pi2, err := proc2.Process(srcPath, "/images/cached.jpg", outputDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(pi2.Variants) != len(pi1.Variants) {
		t.Errorf("cached variants = %d; want %d", len(pi2.Variants), len(pi1.Variants))
	}

	// Verify output files were restored from cache.
	for _, v := range pi2.Variants {
		if _, err := os.Stat(v.Path); err != nil {
			t.Errorf("variant file missing after cache restore: %s", v.Path)
		}
	}
}

// ---------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------

func TestIsSupportedImage(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"photo.jpg", true},
		{"photo.JPEG", true},
		{"photo.png", true},
		{"photo.PNG", true},
		{"photo.gif", false},
		{"photo.svg", false},
		{"photo.webp", false},
		{"document.pdf", false},
	}
	for _, tt := range tests {
		got := isSupportedImage(tt.path)
		if got != tt.want {
			t.Errorf("isSupportedImage(%q) = %v; want %v", tt.path, got, tt.want)
		}
	}
}

func TestNormalizeFormats(t *testing.T) {
	formats := normalizeFormats([]string{"webp", "original"}, "photo.jpg")
	if len(formats) != 2 {
		t.Fatalf("len = %d; want 2", len(formats))
	}
	if formats[0] != "webp" || formats[1] != "jpeg" {
		t.Errorf("formats = %v; want [webp jpeg]", formats)
	}

	// Original for PNG.
	formats = normalizeFormats([]string{"original"}, "logo.png")
	if len(formats) != 1 || formats[0] != "png" {
		t.Errorf("formats = %v; want [png]", formats)
	}

	// Deduplication: if "original" resolves to a format already listed.
	formats = normalizeFormats([]string{"webp", "original", "webp"}, "photo.jpg")
	if len(formats) != 2 {
		t.Errorf("expected dedup, got %v", formats)
	}
}

func TestFileStem(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/path/to/hero.jpg", "hero"},
		{"logo.png", "logo"},
		{"/a/b/file.name.ext", "file.name"},
	}
	for _, tt := range tests {
		got := fileStem(tt.input)
		if got != tt.want {
			t.Errorf("fileStem(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatExtension(t *testing.T) {
	tests := []struct {
		format string
		want   string
	}{
		{"webp", "webp"},
		{"png", "png"},
		{"jpeg", "jpg"},
		{"anything", "jpg"},
	}
	for _, tt := range tests {
		got := formatExtension(tt.format)
		if got != tt.want {
			t.Errorf("formatExtension(%q) = %q; want %q", tt.format, got, tt.want)
		}
	}
}

func TestUrlDir(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/images/hero.jpg", "/images"},
		{"/a/b/c/file.png", "/a/b/c"},
		{"file.jpg", ""},
	}
	for _, tt := range tests {
		got := urlDir(tt.input)
		if got != tt.want {
			t.Errorf("urlDir(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestIntSliceEqual(t *testing.T) {
	if !intSliceEqual([]int{1, 2, 3}, []int{3, 2, 1}) {
		t.Error("expected equal (order-independent)")
	}
	if intSliceEqual([]int{1, 2}, []int{1, 2, 3}) {
		t.Error("expected not equal (different lengths)")
	}
	if intSliceEqual([]int{1, 2}, []int{1, 3}) {
		t.Error("expected not equal (different values)")
	}
}

func TestStringSliceEqual(t *testing.T) {
	if !stringSliceEqual([]string{"a", "b"}, []string{"b", "a"}) {
		t.Error("expected equal (order-independent)")
	}
	if stringSliceEqual([]string{"a"}, []string{"a", "b"}) {
		t.Error("expected not equal (different lengths)")
	}
}

func TestVariantURLs(t *testing.T) {
	// Verify the URL construction in Process follows the expected pattern.
	projectRoot := t.TempDir()
	srcDir := filepath.Join(projectRoot, "static")
	outputDir := filepath.Join(projectRoot, "public")

	srcPath := filepath.Join(srcDir, "photo.jpg")
	createTestJPEG(t, srcPath, 640, 480)

	cfg := config.ImageConfig{
		Enabled: true,
		Quality: 75,
		Sizes:   []int{320},
		Formats: []string{"webp", "original"},
	}
	proc := NewProcessor(cfg, projectRoot)
	pi, err := proc.Process(srcPath, "/static/photo.jpg", outputDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range pi.Variants {
		if !strings.HasPrefix(v.URL, "/static/") {
			t.Errorf("URL %q does not start with /static/", v.URL)
		}
		expectedSuffix := "-320w."
		if !strings.Contains(v.URL, expectedSuffix) {
			t.Errorf("URL %q does not contain %q", v.URL, expectedSuffix)
		}
	}
}
