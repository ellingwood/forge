package build

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// WriteFile writes data to a file determined by the page URL.
// For a URL like "/blog/my-post/", it writes to outputDir/blog/my-post/index.html.
// For the root URL "/", it writes to outputDir/index.html.
func WriteFile(outputDir, url string, data []byte) error {
	// Determine the file path from the URL.
	relPath := strings.TrimPrefix(url, "/")
	var filePath string
	if relPath == "" {
		// Root URL "/" -> index.html
		filePath = filepath.Join(outputDir, "index.html")
	} else if strings.HasSuffix(relPath, "/") {
		// Directory URL like "blog/my-post/" -> blog/my-post/index.html
		filePath = filepath.Join(outputDir, relPath, "index.html")
	} else {
		// URL without trailing slash like "blog/my-post" -> blog/my-post/index.html
		filePath = filepath.Join(outputDir, relPath, "index.html")
	}

	// Ensure the parent directory exists.
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("writing file %s: %w", filePath, err)
	}
	return nil
}

// CopyDir recursively copies the contents of src into dst.
// If dst does not exist, it is created. Existing files at dst are overwritten.
func CopyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source %s: %w", src, err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("creating destination %s: %w", dst, err)
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}
		dstPath := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}

		return CopyFile(path, dstPath)
	})
}

// CopyFile copies a single file from src to dst. The destination directory
// must already exist. If dst exists, it is overwritten.
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source %s: %w", src, err)
	}
	defer srcFile.Close()

	// Ensure destination directory exists.
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", dst, err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination %s: %w", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}
	return nil
}

// CleanDir removes the directory at dir and recreates it as an empty directory.
// If dir does not exist, it is simply created.
func CleanDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing %s: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	return nil
}

// DirSize calculates the total size in bytes of all files in dir, recursively.
// If dir does not exist, it returns 0.
func DirSize(dir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil && os.IsNotExist(err) {
		return 0, nil
	}
	return total, err
}
