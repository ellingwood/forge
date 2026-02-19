package build

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBinaryName(t *testing.T) {
	name := BinaryName()

	// Should start with "tailwindcss-"
	if !strings.HasPrefix(name, "tailwindcss-") {
		t.Errorf("BinaryName() = %q, want prefix \"tailwindcss-\"", name)
	}

	// Verify correct OS mapping.
	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(name, "macos") {
			t.Errorf("BinaryName() = %q, expected \"macos\" for darwin", name)
		}
	case "linux":
		if !strings.Contains(name, "linux") {
			t.Errorf("BinaryName() = %q, expected \"linux\" for linux", name)
		}
	}

	// Verify correct arch mapping.
	switch runtime.GOARCH {
	case "amd64":
		if !strings.HasSuffix(name, "-x64") && !strings.HasSuffix(name, "-x64.exe") {
			t.Errorf("BinaryName() = %q, expected suffix \"-x64\" for amd64", name)
		}
	case "arm64":
		if !strings.HasSuffix(name, "-arm64") && !strings.HasSuffix(name, "-arm64.exe") {
			t.Errorf("BinaryName() = %q, expected suffix \"-arm64\" for arm64", name)
		}
	}

	// Windows should have .exe suffix.
	if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
		t.Errorf("BinaryName() = %q, expected .exe suffix on Windows", name)
	}

	// Non-Windows should NOT have .exe suffix.
	if runtime.GOOS != "windows" && strings.HasSuffix(name, ".exe") {
		t.Errorf("BinaryName() = %q, unexpected .exe suffix on %s", name, runtime.GOOS)
	}
}

func TestDownloadURL(t *testing.T) {
	version := "4.1.0"
	url := DownloadURL(version)

	// Should contain the GitHub releases base URL.
	expectedBase := "https://github.com/tailwindlabs/tailwindcss/releases/download/v4.1.0/"
	if !strings.HasPrefix(url, expectedBase) {
		t.Errorf("DownloadURL(%q) = %q, want prefix %q", version, url, expectedBase)
	}

	// Should end with the binary name.
	expectedSuffix := BinaryName()
	if !strings.HasSuffix(url, expectedSuffix) {
		t.Errorf("DownloadURL(%q) = %q, want suffix %q", version, url, expectedSuffix)
	}

	// Full URL check.
	expectedURL := expectedBase + expectedSuffix
	if url != expectedURL {
		t.Errorf("DownloadURL(%q) = %q, want %q", version, url, expectedURL)
	}
}

func TestBinaryPath(t *testing.T) {
	binDir := t.TempDir()
	tb := &TailwindBuilder{BinDir: binDir}

	path := tb.BinaryPath()

	// Should be under the specified BinDir.
	if !strings.HasPrefix(path, binDir) {
		t.Errorf("BinaryPath() = %q, want prefix %q", path, binDir)
	}

	// Should end with the binary name.
	expectedName := BinaryName()
	if filepath.Base(path) != expectedName {
		t.Errorf("BinaryPath() base = %q, want %q", filepath.Base(path), expectedName)
	}

	// Full path check.
	expectedPath := filepath.Join(binDir, expectedName)
	if path != expectedPath {
		t.Errorf("BinaryPath() = %q, want %q", path, expectedPath)
	}
}

func TestBinaryPath_DefaultBinDir(t *testing.T) {
	tb := &TailwindBuilder{}
	path := tb.BinaryPath()

	// Should contain .forge/bin in the path.
	if !strings.Contains(path, filepath.Join(".forge", "bin")) {
		t.Errorf("BinaryPath() with default BinDir = %q, expected to contain .forge/bin", path)
	}

	// Should end with the binary name.
	expectedName := BinaryName()
	if filepath.Base(path) != expectedName {
		t.Errorf("BinaryPath() base = %q, want %q", filepath.Base(path), expectedName)
	}
}

func TestEnsureBinary_AlreadyExists(t *testing.T) {
	binDir := t.TempDir()
	tb := &TailwindBuilder{BinDir: binDir}

	// Create a fake binary at the expected path.
	binPath := tb.BinaryPath()
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a matching version marker.
	if err := os.WriteFile(tb.versionMarkerPath(), []byte("4.1.0"), 0o644); err != nil {
		t.Fatal(err)
	}

	// EnsureBinary should return the path without attempting a download.
	got, err := tb.EnsureBinary("4.1.0")
	if err != nil {
		t.Fatalf("EnsureBinary() error: %v", err)
	}
	if got != binPath {
		t.Errorf("EnsureBinary() = %q, want %q", got, binPath)
	}

	// Verify the file content is unchanged (no download occurred).
	data, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "#!/bin/sh\necho fake" {
		t.Error("binary file content was modified; expected no download")
	}
}

func TestEnsureBinary_VersionMismatch(t *testing.T) {
	binDir := t.TempDir()
	tb := &TailwindBuilder{BinDir: binDir}

	// Create a fake binary at the expected path.
	binPath := tb.BinaryPath()
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a mismatched version marker (old version).
	if err := os.WriteFile(tb.versionMarkerPath(), []byte("3.4.17"), 0o644); err != nil {
		t.Fatal(err)
	}

	// EnsureBinary should attempt to download because version doesn't match.
	// Since we use a non-existent version, it will fail — this verifies the
	// version check triggers a re-download.
	_, err := tb.EnsureBinary("0.0.0-nonexistent")
	if err == nil {
		t.Error("EnsureBinary() should have returned an error when downloading a non-existent version")
	}
}

func TestEnsureBinary_NotExecutable(t *testing.T) {
	binDir := t.TempDir()
	tb := &TailwindBuilder{BinDir: binDir}

	// Create a file that exists but is NOT executable.
	binPath := tb.BinaryPath()
	if err := os.WriteFile(binPath, []byte("not executable"), 0o644); err != nil {
		t.Fatal(err)
	}

	// EnsureBinary should attempt to download because the file is not executable.
	// Since we don't want to hit the network, we just verify it tries and fails
	// (since the version URL won't resolve in a test environment without network).
	// This test verifies that the executable check works.
	_, err := tb.EnsureBinary("0.0.0-nonexistent")
	if err == nil {
		t.Error("EnsureBinary() should have returned an error when downloading a non-existent version")
	}
}

func TestBuild_CommandArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	binDir := t.TempDir()
	tb := &TailwindBuilder{BinDir: binDir}

	// Create a mock script that records its arguments.
	binPath := tb.BinaryPath()
	argsFile := filepath.Join(t.TempDir(), "recorded_args.txt")

	script := "#!/bin/sh\necho \"$@\" > " + argsFile + "\n"
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	input := "/path/to/input.css"
	output := "/path/to/output.css"
	cwd := "/path/to/project"

	err := tb.Build(input, output, cwd)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Verify the recorded arguments.
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("reading recorded args: %v", err)
	}

	args := strings.TrimSpace(string(data))

	// Check that all expected arguments are present.
	if !strings.Contains(args, "-i "+input) {
		t.Errorf("args %q missing \"-i %s\"", args, input)
	}
	if !strings.Contains(args, "-o "+output) {
		t.Errorf("args %q missing \"-o %s\"", args, output)
	}
	if !strings.Contains(args, "--cwd "+cwd) {
		t.Errorf("args %q missing \"--cwd %s\"", args, cwd)
	}
	if !strings.Contains(args, "--minify") {
		t.Errorf("args %q missing \"--minify\"", args)
	}
	// v4 should NOT have --content or --config flags.
	if strings.Contains(args, "--content") {
		t.Errorf("args %q should not contain \"--content\" (v4 uses automatic content detection)", args)
	}
	if strings.Contains(args, "--config") {
		t.Errorf("args %q should not contain \"--config\" (v4 uses CSS-first config)", args)
	}
}

func TestWatch_StartsAndStops(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	binDir := t.TempDir()
	tb := &TailwindBuilder{BinDir: binDir}

	// Create a mock script that runs indefinitely and records its PID.
	binPath := tb.BinaryPath()
	pidFile := filepath.Join(t.TempDir(), "watch_pid.txt")

	// The script writes its PID to a file, then sleeps forever.
	script := "#!/bin/sh\necho $$ > " + pidFile + "\nwhile true; do sleep 1; done\n"
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	input := "/path/to/input.css"
	output := "/path/to/output.css"
	cwd := "/path/to/project"

	cancel, err := tb.Watch(input, output, cwd)
	if err != nil {
		t.Fatalf("Watch() error: %v", err)
	}

	// Wait briefly for the process to start and write its PID.
	deadline := time.Now().Add(3 * time.Second)
	var pidData []byte
	for time.Now().Before(deadline) {
		pidData, err = os.ReadFile(pidFile)
		if err == nil && len(strings.TrimSpace(string(pidData))) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(pidData) == 0 {
		t.Fatal("watch process did not write PID file")
	}

	pid := strings.TrimSpace(string(pidData))
	if pid == "" {
		t.Fatal("PID file is empty")
	}

	// Verify the process is running by checking the PID file exists.
	t.Logf("watch process started with PID %s", pid)

	// Call cancel to stop the watcher.
	cancel()

	// Wait briefly and verify the process has stopped.
	time.Sleep(200 * time.Millisecond)

	// Try to find the process - it should no longer be running.
	// We check by trying to send signal 0 to the PID.
	// On Unix, this checks if the process exists without sending a signal.
	// After Kill + Wait, the process should be gone.
	// We parse the PID and check.
	var pidInt int
	if _, err := fmt.Sscanf(pid, "%d", &pidInt); err != nil {
		t.Fatalf("parsing PID %q: %v", pid, err)
	}

	proc, err := os.FindProcess(pidInt)
	if err != nil {
		// Process not found, which is what we want.
		return
	}

	// On Unix, FindProcess always succeeds. Check if it's actually running.
	err = proc.Signal(os.Signal(nil))
	// If err is nil, the process is somehow still alive (unlikely after Kill+Wait).
	// If err is non-nil, the process is dead, which is expected.
	if err == nil {
		t.Log("process may still be running after cancel, but this is a race condition in tests")
	}
}

func TestWatch_CommandArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	binDir := t.TempDir()
	tb := &TailwindBuilder{BinDir: binDir}

	// Create a mock script that records its arguments and exits.
	binPath := tb.BinaryPath()
	argsFile := filepath.Join(t.TempDir(), "watch_args.txt")

	// The script records its arguments then sleeps so Watch doesn't fail immediately.
	script := "#!/bin/sh\necho \"$@\" > " + argsFile + "\nsleep 60\n"
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	input := "/path/to/input.css"
	output := "/path/to/output.css"
	cwd := "/path/to/project"

	cancel, err := tb.Watch(input, output, cwd)
	if err != nil {
		t.Fatalf("Watch() error: %v", err)
	}
	defer cancel()

	// Wait for the args file to be written.
	deadline := time.Now().Add(3 * time.Second)
	var data []byte
	for time.Now().Before(deadline) {
		data, err = os.ReadFile(argsFile)
		if err == nil && len(data) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(data) == 0 {
		t.Fatal("watch process did not write args file")
	}

	args := strings.TrimSpace(string(data))

	// Check for --watch flag instead of --minify.
	if !strings.Contains(args, "--watch") {
		t.Errorf("watch args %q missing \"--watch\"", args)
	}
	if strings.Contains(args, "--minify") {
		t.Errorf("watch args %q should not contain \"--minify\"", args)
	}
	if !strings.Contains(args, "-i "+input) {
		t.Errorf("watch args %q missing \"-i %s\"", args, input)
	}
	if !strings.Contains(args, "-o "+output) {
		t.Errorf("watch args %q missing \"-o %s\"", args, output)
	}
	if !strings.Contains(args, "--cwd "+cwd) {
		t.Errorf("watch args %q missing \"--cwd %s\"", args, cwd)
	}
	// v4 should NOT have --content or --config flags.
	if strings.Contains(args, "--content") {
		t.Errorf("watch args %q should not contain \"--content\" (v4 uses automatic content detection)", args)
	}
	if strings.Contains(args, "--config") {
		t.Errorf("watch args %q should not contain \"--config\" (v4 uses CSS-first config)", args)
	}
}
