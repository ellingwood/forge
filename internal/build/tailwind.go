package build

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// TailwindBuilder manages the Tailwind CSS standalone CLI binary.
type TailwindBuilder struct {
	BinDir string // directory where the binary is cached (default: ~/.forge/bin/)
}

// BinaryName returns the filename of the tailwindcss binary for the current
// OS and architecture. Format: "tailwindcss-<os>-<arch>" with ".exe" suffix
// on Windows. Supported: linux/darwin + amd64/arm64.
func BinaryName() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go OS names to Tailwind binary naming convention.
	switch osName {
	case "darwin":
		osName = "macos"
	case "linux":
		osName = "linux"
	default:
		osName = runtime.GOOS
	}

	// Map Go architecture names to Tailwind binary naming convention.
	switch arch {
	case "amd64":
		arch = "x64"
	case "arm64":
		arch = "arm64"
	default:
		arch = runtime.GOARCH
	}

	name := fmt.Sprintf("tailwindcss-%s-%s", osName, arch)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// DownloadURL returns the GitHub release download URL for the latest Tailwind
// CSS standalone CLI binary.
func DownloadURL(version string) string {
	return fmt.Sprintf(
		"https://github.com/tailwindlabs/tailwindcss/releases/download/v%s/%s",
		version, BinaryName(),
	)
}

// BinaryPath returns the expected path to the tailwindcss binary for the
// current OS and architecture.
func (tb *TailwindBuilder) BinaryPath() string {
	binDir := tb.BinDir
	if binDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			binDir = filepath.Join(".", ".forge", "bin")
		} else {
			binDir = filepath.Join(home, ".forge", "bin")
		}
	}
	return filepath.Join(binDir, BinaryName())
}

// EnsureBinary checks if the Tailwind binary exists and is executable.
// If not, it downloads it from GitHub releases. Returns the path to the binary.
func (tb *TailwindBuilder) EnsureBinary(version string) (string, error) {
	binPath := tb.BinaryPath()

	// Check if binary already exists and is executable.
	info, err := os.Stat(binPath)
	if err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
		return binPath, nil
	}

	// Create BinDir if needed.
	binDir := filepath.Dir(binPath)
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", fmt.Errorf("creating bin directory %s: %w", binDir, err)
	}

	// Download the binary.
	url := DownloadURL(version)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("downloading tailwindcss from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading tailwindcss: HTTP %d from %s", resp.StatusCode, url)
	}

	// Write to file.
	f, err := os.Create(binPath)
	if err != nil {
		return "", fmt.Errorf("creating binary file %s: %w", binPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("writing binary to %s: %w", binPath, err)
	}

	// Close the file before chmod so all data is flushed.
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("closing binary file %s: %w", binPath, err)
	}

	// Make executable.
	if err := os.Chmod(binPath, 0o755); err != nil {
		return "", fmt.Errorf("making binary executable: %w", err)
	}

	return binPath, nil
}

// Build runs the Tailwind CLI in production mode with --minify.
// input is the path to the input CSS file (e.g., themes/default/static/css/globals.css)
// output is the path to write the compiled CSS (e.g., public/css/style.css)
// contentPaths are glob patterns for content files to scan for class usage.
func (tb *TailwindBuilder) Build(input, output string, contentPaths []string) error {
	binPath := tb.BinaryPath()

	args := []string{
		"-i", input,
		"-o", output,
		"--content", strings.Join(contentPaths, ","),
		"--minify",
	}

	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running tailwindcss build: %w", err)
	}
	return nil
}

// Watch runs the Tailwind CLI in watch mode for development.
// It returns a cancel function to stop the watcher.
func (tb *TailwindBuilder) Watch(input, output string, contentPaths []string) (cancel func(), err error) {
	binPath := tb.BinaryPath()

	args := []string{
		"-i", input,
		"-o", output,
		"--content", strings.Join(contentPaths, ","),
		"--watch",
	}

	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting tailwindcss watch: %w", err)
	}

	cancel = func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}

	return cancel, nil
}
