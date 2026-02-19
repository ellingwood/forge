package deploy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// URLRewriteFunctionCode is the CloudFront Function (cloudfront-js-2.0) source
// that rewrites viewer-request URIs to append index.html for clean URLs.
const URLRewriteFunctionCode = `function handler(event) {
    var request = event.request;
    var uri = request.uri;

    // Has a file extension — pass through
    if (uri.match(/\.[a-zA-Z0-9]+$/)) {
        return request;
    }
    // Trailing slash — append index.html
    if (uri.endsWith('/')) {
        request.uri = uri + 'index.html';
        return request;
    }
    // No extension, no trailing slash — append /index.html
    request.uri = uri + '/index.html';
    return request;
}
`

// DeployConfig holds deployment configuration.
type DeployConfig struct {
	Bucket       string
	Region       string
	Distribution string // CloudFront distribution ID (optional)
	URLRewrite   bool   // whether to manage a CloudFront URL rewrite function
	DryRun       bool
	Verbose      bool
}

// DeployResult holds the results of a deployment.
type DeployResult struct {
	Uploaded int
	Deleted  int
	Skipped  int
	Errors   []error
}

// FileEntry represents a local file to deploy.
type FileEntry struct {
	Path         string // relative path from public dir (e.g. "blog/index.html")
	ContentType  string // MIME type
	CacheControl string // Cache-Control header value
	Hash         string // hex-encoded SHA-256 hash
}

// S3Client is an interface for S3 operations used during deployment.
type S3Client interface {
	PutObject(ctx context.Context, key string, body io.Reader, contentType, cacheControl, sha256Hash string) error
	DeleteObject(ctx context.Context, key string) error
	ListObjects(ctx context.Context, prefix string) (map[string]string, error) // returns key -> hash metadata
}

// CloudFrontClient is an interface for CloudFront operations.
type CloudFrontClient interface {
	CreateInvalidation(ctx context.Context, distributionID string, paths []string) error
}

// CloudFrontFunctionClient is an interface for managing CloudFront Functions.
type CloudFrontFunctionClient interface {
	// EnsureURLRewriteFunction creates or updates a CloudFront Function that
	// rewrites URIs to append index.html, then associates it with the
	// distribution's default cache behavior as a viewer-request function.
	// Returns the function ARN on success.
	EnsureURLRewriteFunction(ctx context.Context, distributionID, functionName, functionCode string) (string, error)
}

// ContentTypeForExt returns the MIME type for a file extension.
// The ext parameter should include the leading dot (e.g. ".html").
func ContentTypeForExt(ext string) string {
	ext = strings.ToLower(ext)

	// Well-known types that we want to be explicit about
	switch ext {
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".xml":
		return "application/xml; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".avif":
		return "image/avif"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".otf":
		return "font/otf"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".csv":
		return "text/csv; charset=utf-8"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".wasm":
		return "application/wasm"
	}

	// Fall back to the standard library
	ct := mime.TypeByExtension(ext)
	if ct != "" {
		return ct
	}
	return "application/octet-stream"
}

// CacheControlForExt returns the Cache-Control header for a file extension.
// The ext parameter should include the leading dot (e.g. ".html").
//
// Policy:
//   - HTML files: "public, max-age=0, must-revalidate"
//   - CSS/JS files: "public, max-age=31536000, immutable" (fingerprinted)
//   - Image files: "public, max-age=86400"
//   - Other files: "public, max-age=3600"
func CacheControlForExt(ext string) string {
	ext = strings.ToLower(ext)
	switch ext {
	case ".html", ".htm":
		return "public, max-age=0, must-revalidate"
	case ".css", ".js", ".mjs":
		return "public, max-age=31536000, immutable"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif", ".svg", ".ico":
		return "public, max-age=86400"
	default:
		return "public, max-age=3600"
	}
}

// HashFile computes the SHA-256 hash of a file and returns it as a hex string.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening file for hashing: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hashing file: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ScanFiles walks the public directory and returns a list of FileEntry.
// It determines Content-Type from file extension and sets Cache-Control
// based on the file type.
func ScanFiles(publicDir string) ([]FileEntry, error) {
	var entries []FileEntry

	err := filepath.Walk(publicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(publicDir, path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}
		// Normalize to forward slashes for S3 keys
		relPath = filepath.ToSlash(relPath)

		ext := filepath.Ext(path)
		hash, err := HashFile(path)
		if err != nil {
			return err
		}

		entries = append(entries, FileEntry{
			Path:         relPath,
			ContentType:  ContentTypeForExt(ext),
			CacheControl: CacheControlForExt(ext),
			Hash:         hash,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning files: %w", err)
	}

	return entries, nil
}

// DiffFiles compares local files against a map of remote S3 object hashes.
// Returns lists of files to upload (new or changed) and keys to delete (remote only).
// The remoteHashes map has keys as S3 object keys (same format as FileEntry.Path)
// and values as SHA-256 hex hashes stored in object metadata.
func DiffFiles(local []FileEntry, remoteHashes map[string]string) (toUpload []FileEntry, toDelete []string) {
	localMap := make(map[string]FileEntry, len(local))
	for _, entry := range local {
		localMap[entry.Path] = entry
	}

	// Find files to upload: new or changed
	for _, entry := range local {
		remoteHash, exists := remoteHashes[entry.Path]
		if !exists || remoteHash != entry.Hash {
			toUpload = append(toUpload, entry)
		}
	}

	// Find files to delete: exist remotely but not locally
	for key := range remoteHashes {
		if _, exists := localMap[key]; !exists {
			toDelete = append(toDelete, key)
		}
	}

	return toUpload, toDelete
}

// Deploy executes the deployment using the provided clients.
//
// Steps:
//  1. Scan local files
//  2. List remote objects via S3Client
//  3. Diff to find uploads and deletes
//  4. If DryRun, print plan and return
//  5. Upload new/changed files
//  6. Delete removed files
//  7. If URLRewrite enabled, ensure CloudFront URL rewrite function
//  8. If Distribution is set, invalidate CloudFront with "/*"
//  9. Return results
func Deploy(ctx context.Context, cfg DeployConfig, publicDir string, s3 S3Client, cf CloudFrontClient, cfFunc CloudFrontFunctionClient) (*DeployResult, error) {
	result := &DeployResult{}

	// 1. Scan local files
	localFiles, err := ScanFiles(publicDir)
	if err != nil {
		return nil, fmt.Errorf("scanning local files: %w", err)
	}

	// 2. List remote objects
	remoteHashes, err := s3.ListObjects(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("listing remote objects: %w", err)
	}

	// 3. Diff
	toUpload, toDelete := DiffFiles(localFiles, remoteHashes)
	result.Skipped = len(localFiles) - len(toUpload)

	// 4. Dry run
	if cfg.DryRun {
		if cfg.Verbose {
			for _, f := range toUpload {
				fmt.Printf("[dry-run] upload: %s (%s)\n", f.Path, f.ContentType)
			}
			for _, key := range toDelete {
				fmt.Printf("[dry-run] delete: %s\n", key)
			}
		}
		if cfg.URLRewrite && cfg.Distribution != "" {
			fmt.Println("[dry-run] ensure CloudFront URL rewrite function: forge-url-rewrite")
		}
		if cfg.Distribution != "" {
			fmt.Printf("[dry-run] invalidate CloudFront distribution: %s\n", cfg.Distribution)
		}
		result.Uploaded = len(toUpload)
		result.Deleted = len(toDelete)
		return result, nil
	}

	// 5. Upload new/changed files
	for _, entry := range toUpload {
		fullPath := filepath.Join(publicDir, filepath.FromSlash(entry.Path))
		f, err := os.Open(fullPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("opening %s: %w", entry.Path, err))
			continue
		}

		err = s3.PutObject(ctx, entry.Path, f, entry.ContentType, entry.CacheControl, entry.Hash)
		f.Close()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("uploading %s: %w", entry.Path, err))
			continue
		}
		result.Uploaded++
		if cfg.Verbose {
			fmt.Printf("uploaded: %s\n", entry.Path)
		}
	}

	// 6. Delete removed files
	for _, key := range toDelete {
		if err := s3.DeleteObject(ctx, key); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("deleting %s: %w", key, err))
			continue
		}
		result.Deleted++
		if cfg.Verbose {
			fmt.Printf("deleted: %s\n", key)
		}
	}

	// 7. Ensure CloudFront URL rewrite function if enabled
	if cfg.URLRewrite && cfg.Distribution != "" && cfFunc != nil {
		arn, err := cfFunc.EnsureURLRewriteFunction(ctx, cfg.Distribution,
			"forge-url-rewrite", URLRewriteFunctionCode)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("CloudFront URL rewrite function: %w", err))
		} else if cfg.Verbose {
			fmt.Printf("ensured CloudFront URL rewrite function: %s\n", arn)
		}
	}

	// 8. Invalidate CloudFront if distribution is set
	if cfg.Distribution != "" && cf != nil {
		if err := cf.CreateInvalidation(ctx, cfg.Distribution, []string{"/*"}); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("CloudFront invalidation: %w", err))
		} else if cfg.Verbose {
			fmt.Printf("invalidated CloudFront distribution: %s\n", cfg.Distribution)
		}
	}

	return result, nil
}
