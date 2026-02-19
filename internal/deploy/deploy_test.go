package deploy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// mockS3Client for testing
type mockS3Client struct {
	objects   map[string]string // key -> hash
	uploaded  []string
	deleted   []string
	putErr    error
	deleteErr error
}

func (m *mockS3Client) PutObject(_ context.Context, key string, _ io.Reader, _, _, _ string) error {
	if m.putErr != nil {
		return m.putErr
	}
	m.uploaded = append(m.uploaded, key)
	return nil
}

func (m *mockS3Client) DeleteObject(_ context.Context, key string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deleted = append(m.deleted, key)
	return nil
}

func (m *mockS3Client) ListObjects(_ context.Context, _ string) (map[string]string, error) {
	if m.objects == nil {
		return map[string]string{}, nil
	}
	return m.objects, nil
}

// mockCloudFrontClient for testing
type mockCloudFrontClient struct {
	invalidations []struct {
		distributionID string
		paths          []string
	}
	err error
}

func (m *mockCloudFrontClient) CreateInvalidation(_ context.Context, distributionID string, paths []string) error {
	if m.err != nil {
		return m.err
	}
	m.invalidations = append(m.invalidations, struct {
		distributionID string
		paths          []string
	}{distributionID, paths})
	return nil
}

// createTempFile creates a file in the given directory with the given content.
func createTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// sha256Hex computes the SHA-256 hash of the given data and returns it as a hex string.
func sha256Hex(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

func TestScanFiles(t *testing.T) {
	dir := t.TempDir()

	createTempFile(t, dir, "index.html", "<html>hello</html>")
	createTempFile(t, dir, "style.css", "body { color: red; }")
	createTempFile(t, dir, "app.js", "console.log('hi');")
	createTempFile(t, dir, "images/logo.png", "fakepngdata")
	createTempFile(t, dir, "blog/post.html", "<html>post</html>")

	entries, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	if len(entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(entries))
	}

	// Build a map for easy lookup
	entryMap := make(map[string]FileEntry)
	for _, e := range entries {
		entryMap[e.Path] = e
	}

	// Check HTML file
	e, ok := entryMap["index.html"]
	if !ok {
		t.Fatal("expected index.html entry")
	}
	if e.ContentType != "text/html; charset=utf-8" {
		t.Errorf("expected text/html; charset=utf-8, got %s", e.ContentType)
	}
	if e.CacheControl != "public, max-age=0, must-revalidate" {
		t.Errorf("expected HTML cache control, got %s", e.CacheControl)
	}
	if e.Hash != sha256Hex("<html>hello</html>") {
		t.Errorf("hash mismatch for index.html")
	}

	// Check CSS file
	e, ok = entryMap["style.css"]
	if !ok {
		t.Fatal("expected style.css entry")
	}
	if e.ContentType != "text/css; charset=utf-8" {
		t.Errorf("expected text/css; charset=utf-8, got %s", e.ContentType)
	}
	if e.CacheControl != "public, max-age=31536000, immutable" {
		t.Errorf("expected CSS cache control, got %s", e.CacheControl)
	}

	// Check JS file
	e, ok = entryMap["app.js"]
	if !ok {
		t.Fatal("expected app.js entry")
	}
	if e.ContentType != "application/javascript; charset=utf-8" {
		t.Errorf("expected application/javascript; charset=utf-8, got %s", e.ContentType)
	}
	if e.CacheControl != "public, max-age=31536000, immutable" {
		t.Errorf("expected JS cache control, got %s", e.CacheControl)
	}

	// Check image file with subdirectory
	e, ok = entryMap["images/logo.png"]
	if !ok {
		t.Fatal("expected images/logo.png entry")
	}
	if e.ContentType != "image/png" {
		t.Errorf("expected image/png, got %s", e.ContentType)
	}
	if e.CacheControl != "public, max-age=86400" {
		t.Errorf("expected image cache control, got %s", e.CacheControl)
	}

	// Check nested HTML file
	e, ok = entryMap["blog/post.html"]
	if !ok {
		t.Fatal("expected blog/post.html entry")
	}
	if e.ContentType != "text/html; charset=utf-8" {
		t.Errorf("expected text/html; charset=utf-8, got %s", e.ContentType)
	}
}

func TestContentTypeForExt(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".html", "text/html; charset=utf-8"},
		{".htm", "text/html; charset=utf-8"},
		{".css", "text/css; charset=utf-8"},
		{".js", "application/javascript; charset=utf-8"},
		{".mjs", "application/javascript; charset=utf-8"},
		{".json", "application/json; charset=utf-8"},
		{".xml", "application/xml; charset=utf-8"},
		{".svg", "image/svg+xml"},
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".gif", "image/gif"},
		{".webp", "image/webp"},
		{".ico", "image/x-icon"},
		{".woff", "font/woff"},
		{".woff2", "font/woff2"},
		{".pdf", "application/pdf"},
		{".txt", "text/plain; charset=utf-8"},
		{".wasm", "application/wasm"},
		{".unknown123", "application/octet-stream"},
	}

	for _, tc := range tests {
		t.Run(tc.ext, func(t *testing.T) {
			got := ContentTypeForExt(tc.ext)
			if got != tc.expected {
				t.Errorf("ContentTypeForExt(%q) = %q, want %q", tc.ext, got, tc.expected)
			}
		})
	}
}

func TestCacheControlForExt(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".html", "public, max-age=0, must-revalidate"},
		{".htm", "public, max-age=0, must-revalidate"},
		{".css", "public, max-age=31536000, immutable"},
		{".js", "public, max-age=31536000, immutable"},
		{".mjs", "public, max-age=31536000, immutable"},
		{".png", "public, max-age=86400"},
		{".jpg", "public, max-age=86400"},
		{".jpeg", "public, max-age=86400"},
		{".gif", "public, max-age=86400"},
		{".webp", "public, max-age=86400"},
		{".svg", "public, max-age=86400"},
		{".ico", "public, max-age=86400"},
		{".pdf", "public, max-age=3600"},
		{".woff2", "public, max-age=3600"},
		{".json", "public, max-age=3600"},
		{".xml", "public, max-age=3600"},
		{".txt", "public, max-age=3600"},
	}

	for _, tc := range tests {
		t.Run(tc.ext, func(t *testing.T) {
			got := CacheControlForExt(tc.ext)
			if got != tc.expected {
				t.Errorf("CacheControlForExt(%q) = %q, want %q", tc.ext, got, tc.expected)
			}
		})
	}
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	content := "hello world\n"
	path := createTempFile(t, dir, "test.txt", content)

	got, err := HashFile(path)
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	expected := sha256Hex(content)
	if got != expected {
		t.Errorf("HashFile = %q, want %q", got, expected)
	}

	// Test non-existent file
	_, err = HashFile(filepath.Join(dir, "nonexistent.txt"))
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestDiffFiles(t *testing.T) {
	local := []FileEntry{
		{Path: "index.html", Hash: "aaa"},    // unchanged
		{Path: "style.css", Hash: "bbb_new"}, // changed
		{Path: "new.js", Hash: "ccc"},         // new file
	}

	remoteHashes := map[string]string{
		"index.html": "aaa",     // same hash -> skip
		"style.css":  "bbb_old", // different hash -> upload
		"old.html":   "ddd",     // not in local -> delete
	}

	toUpload, toDelete := DiffFiles(local, remoteHashes)

	// Verify uploads: style.css (changed) and new.js (new)
	if len(toUpload) != 2 {
		t.Fatalf("expected 2 uploads, got %d", len(toUpload))
	}
	uploadPaths := make([]string, len(toUpload))
	for i, e := range toUpload {
		uploadPaths[i] = e.Path
	}
	sort.Strings(uploadPaths)
	if uploadPaths[0] != "new.js" || uploadPaths[1] != "style.css" {
		t.Errorf("unexpected upload paths: %v", uploadPaths)
	}

	// Verify deletes: old.html
	if len(toDelete) != 1 {
		t.Fatalf("expected 1 delete, got %d", len(toDelete))
	}
	if toDelete[0] != "old.html" {
		t.Errorf("expected delete of old.html, got %s", toDelete[0])
	}
}

func TestDiffFiles_AllNew(t *testing.T) {
	local := []FileEntry{
		{Path: "a.html", Hash: "aaa"},
		{Path: "b.css", Hash: "bbb"},
	}
	remoteHashes := map[string]string{}

	toUpload, toDelete := DiffFiles(local, remoteHashes)
	if len(toUpload) != 2 {
		t.Errorf("expected 2 uploads, got %d", len(toUpload))
	}
	if len(toDelete) != 0 {
		t.Errorf("expected 0 deletes, got %d", len(toDelete))
	}
}

func TestDiffFiles_AllUnchanged(t *testing.T) {
	local := []FileEntry{
		{Path: "a.html", Hash: "aaa"},
	}
	remoteHashes := map[string]string{
		"a.html": "aaa",
	}

	toUpload, toDelete := DiffFiles(local, remoteHashes)
	if len(toUpload) != 0 {
		t.Errorf("expected 0 uploads, got %d", len(toUpload))
	}
	if len(toDelete) != 0 {
		t.Errorf("expected 0 deletes, got %d", len(toDelete))
	}
}

func TestDeploy_DryRun(t *testing.T) {
	dir := t.TempDir()
	createTempFile(t, dir, "index.html", "<html>test</html>")
	createTempFile(t, dir, "style.css", "body{}")

	s3 := &mockS3Client{
		objects: map[string]string{
			"old.html": "oldhash",
		},
	}
	cf := &mockCloudFrontClient{}

	cfg := DeployConfig{
		Bucket:       "test-bucket",
		Region:       "us-east-1",
		Distribution: "DIST123",
		DryRun:       true,
	}

	result, err := Deploy(context.Background(), cfg, dir, s3, cf)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// In dry run, the result should report counts but no actual operations
	if result.Uploaded != 2 {
		t.Errorf("expected 2 uploads (planned), got %d", result.Uploaded)
	}
	if result.Deleted != 1 {
		t.Errorf("expected 1 delete (planned), got %d", result.Deleted)
	}

	// Verify no actual S3 operations happened
	if len(s3.uploaded) != 0 {
		t.Errorf("expected no actual uploads in dry run, got %d", len(s3.uploaded))
	}
	if len(s3.deleted) != 0 {
		t.Errorf("expected no actual deletes in dry run, got %d", len(s3.deleted))
	}

	// Verify no CloudFront invalidation
	if len(cf.invalidations) != 0 {
		t.Errorf("expected no invalidations in dry run, got %d", len(cf.invalidations))
	}
}

func TestDeploy_UploadAndDelete(t *testing.T) {
	dir := t.TempDir()
	createTempFile(t, dir, "index.html", "<html>test</html>")
	createTempFile(t, dir, "style.css", "body{}")

	indexHash := sha256Hex("<html>test</html>")

	s3 := &mockS3Client{
		objects: map[string]string{
			"index.html": indexHash, // unchanged
			"old.html":   "oldhash", // should be deleted
		},
	}
	cf := &mockCloudFrontClient{}

	cfg := DeployConfig{
		Bucket: "test-bucket",
		Region: "us-east-1",
	}

	result, err := Deploy(context.Background(), cfg, dir, s3, cf)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// index.html is unchanged, style.css is new -> 1 upload
	if result.Uploaded != 1 {
		t.Errorf("expected 1 upload, got %d", result.Uploaded)
	}
	// old.html should be deleted
	if result.Deleted != 1 {
		t.Errorf("expected 1 delete, got %d", result.Deleted)
	}
	// index.html was skipped
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Skipped)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %v", result.Errors)
	}

	// Verify actual S3 operations
	if len(s3.uploaded) != 1 || s3.uploaded[0] != "style.css" {
		t.Errorf("expected upload of style.css, got %v", s3.uploaded)
	}
	if len(s3.deleted) != 1 || s3.deleted[0] != "old.html" {
		t.Errorf("expected delete of old.html, got %v", s3.deleted)
	}

	// No distribution set, so no invalidation
	if len(cf.invalidations) != 0 {
		t.Errorf("expected no invalidations, got %d", len(cf.invalidations))
	}
}

func TestDeploy_WithCloudFront(t *testing.T) {
	dir := t.TempDir()
	createTempFile(t, dir, "index.html", "<html>test</html>")

	s3 := &mockS3Client{
		objects: map[string]string{},
	}
	cf := &mockCloudFrontClient{}

	cfg := DeployConfig{
		Bucket:       "test-bucket",
		Region:       "us-east-1",
		Distribution: "E1234567890",
	}

	result, err := Deploy(context.Background(), cfg, dir, s3, cf)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	if result.Uploaded != 1 {
		t.Errorf("expected 1 upload, got %d", result.Uploaded)
	}

	// Verify CloudFront invalidation was called
	if len(cf.invalidations) != 1 {
		t.Fatalf("expected 1 invalidation, got %d", len(cf.invalidations))
	}
	inv := cf.invalidations[0]
	if inv.distributionID != "E1234567890" {
		t.Errorf("expected distribution E1234567890, got %s", inv.distributionID)
	}
	if len(inv.paths) != 1 || inv.paths[0] != "/*" {
		t.Errorf("expected invalidation paths [/*], got %v", inv.paths)
	}
}

func TestDeploy_NoCloudFrontWithoutDistribution(t *testing.T) {
	dir := t.TempDir()
	createTempFile(t, dir, "index.html", "<html>test</html>")

	s3 := &mockS3Client{
		objects: map[string]string{},
	}
	cf := &mockCloudFrontClient{}

	cfg := DeployConfig{
		Bucket: "test-bucket",
		Region: "us-east-1",
		// Distribution is empty
	}

	_, err := Deploy(context.Background(), cfg, dir, s3, cf)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	if len(cf.invalidations) != 0 {
		t.Errorf("expected no invalidations without distribution, got %d", len(cf.invalidations))
	}
}

func TestScanFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	entries, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty dir, got %d", len(entries))
	}
}

func TestScanFiles_NonExistentDir(t *testing.T) {
	_, err := ScanFiles("/nonexistent/dir/that/does/not/exist")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}
