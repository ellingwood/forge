package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aellingwood/forge/internal/config"
)

// ---------- InjectLiveReload Tests ----------

func TestInjectLiveReload_BeforeBody(t *testing.T) {
	html := []byte("<html><body><p>Hello</p></body></html>")
	result := InjectLiveReload(html, 1313, "testnonce")

	if !bytes.Contains(result, []byte("ws://")) {
		t.Error("expected WebSocket script to be injected")
	}
	if !bytes.Contains(result, []byte(":1313/__forge/ws")) {
		t.Error("expected port 1313 in WebSocket URL")
	}

	// Script should appear before </body>.
	bodyIdx := bytes.Index(result, []byte("</body>"))
	scriptIdx := bytes.Index(result, []byte("<script nonce="))
	if scriptIdx == -1 || bodyIdx == -1 {
		t.Fatal("expected both <script nonce=...> and </body> in result")
	}
	if scriptIdx >= bodyIdx {
		t.Error("expected script to be injected before </body>")
	}
}

func TestInjectLiveReload_MissingBody(t *testing.T) {
	html := []byte("<html><p>No body tag</p></html>")
	result := InjectLiveReload(html, 8080, "testnonce")

	if !bytes.Contains(result, []byte("ws://")) {
		t.Error("expected WebSocket script to be appended")
	}
	if !bytes.Contains(result, []byte(":8080/__forge/ws")) {
		t.Error("expected port 8080 in WebSocket URL")
	}

	// Script should be appended at the end.
	if !bytes.HasSuffix(result, []byte("</script>")) {
		t.Error("expected script to be appended at end when no </body> tag")
	}
}

func TestInjectLiveReload_EmptyHTML(t *testing.T) {
	result := InjectLiveReload([]byte{}, 1313, "testnonce")
	if !bytes.Contains(result, []byte("<script nonce=")) {
		t.Error("expected script to be added even to empty HTML")
	}
}

func TestInjectLiveReload_CustomPort(t *testing.T) {
	html := []byte("<html><body></body></html>")
	result := InjectLiveReload(html, 9999, "testnonce")
	if !bytes.Contains(result, []byte(":9999/__forge/ws")) {
		t.Error("expected custom port 9999 in WebSocket URL")
	}
}

// ---------- InjectScriptNonces Tests ----------

func TestInjectScriptNonces_InlineScript(t *testing.T) {
	html := []byte(`<html><head></head><body><script>alert("hi")</script></body></html>`)
	result := InjectScriptNonces(html, "abc123")
	if !bytes.Contains(result, []byte(`<script nonce="abc123">`)) {
		t.Errorf("expected nonce in inline script, got: %s", result)
	}
}

func TestInjectScriptNonces_SkipsSrcScripts(t *testing.T) {
	html := []byte(`<script src="/js/app.js"></script>`)
	result := InjectScriptNonces(html, "abc123")
	if bytes.Contains(result, []byte("nonce=")) {
		t.Errorf("should not add nonce to script with src=, got: %s", result)
	}
}

func TestInjectScriptNonces_SkipsExistingNonce(t *testing.T) {
	html := []byte(`<script nonce="existing">code()</script>`)
	result := InjectScriptNonces(html, "abc123")
	// Should not add a second nonce.
	if bytes.Contains(result, []byte(`nonce="abc123"`)) {
		t.Errorf("should not add second nonce, got: %s", result)
	}
	// Original nonce should be preserved.
	if !bytes.Contains(result, []byte(`nonce="existing"`)) {
		t.Errorf("original nonce should be preserved, got: %s", result)
	}
}

func TestInjectScriptNonces_MultipleScripts(t *testing.T) {
	html := []byte(`<script>inline()</script><script src="/ext.js"></script><script>another()</script>`)
	result := InjectScriptNonces(html, "xyz")
	expected := `<script nonce="xyz">inline()</script><script src="/ext.js"></script><script nonce="xyz">another()</script>`
	if string(result) != expected {
		t.Errorf("unexpected result:\ngot:  %s\nwant: %s", result, expected)
	}
}

func TestInjectScriptNonces_NoScripts(t *testing.T) {
	html := []byte(`<html><body><p>Hello</p></body></html>`)
	result := InjectScriptNonces(html, "abc")
	if string(result) != string(html) {
		t.Error("should not modify HTML without script tags")
	}
}

func TestInjectScriptNonces_SkipsNonJSTypes(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"application/json", `<script type="application/json">{"key":"val"}</script>`},
		{"application/ld+json", `<script type="application/ld+json">{"@type":"WebPage"}</script>`},
		{"text/template", `<script type="text/template"><div>{{.}}</div></script>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InjectScriptNonces([]byte(tt.input), "abc123")
			if bytes.Contains(result, []byte("nonce=")) {
				t.Errorf("should not add nonce to %s script, got: %s", tt.name, result)
			}
		})
	}
}

func TestInjectScriptNonces_AddsNonceToJSTypes(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"text/javascript", `<script type="text/javascript">code()</script>`},
		{"module", `<script type="module">import x from './x'</script>`},
		{"bare script", `<script>code()</script>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InjectScriptNonces([]byte(tt.input), "abc123")
			if !bytes.Contains(result, []byte(`nonce="abc123"`)) {
				t.Errorf("expected nonce for %s script, got: %s", tt.name, result)
			}
		})
	}
}

func TestHandleRequest_SecurityHeadersOnlyForHTML(t *testing.T) {
	outputDir := t.TempDir()
	writeTestFile(t, outputDir, "style.css", "body { color: red; }")
	writeTestFile(t, outputDir, "index.html", "<html><body>hi</body></html>")

	srv := NewServer(config.Default(), ServeOptions{
		Port:         1313,
		Bind:         "localhost",
		OutputDir:    outputDir,
		NoLiveReload: true,
	})

	// CSS request should NOT have CSP header.
	req := httptest.NewRequest("GET", "/style.css", nil)
	rr := httptest.NewRecorder()
	srv.handleRequest(rr, req)

	if rr.Header().Get("Content-Security-Policy") != "" {
		t.Error("CSS response should not have Content-Security-Policy header")
	}
	if rr.Header().Get("X-Frame-Options") != "" {
		t.Error("CSS response should not have X-Frame-Options header")
	}
	// X-Content-Type-Options should still be set for all responses.
	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("CSS response should still have X-Content-Type-Options: nosniff")
	}

	// HTML request SHOULD have CSP header.
	req2 := httptest.NewRequest("GET", "/", nil)
	rr2 := httptest.NewRecorder()
	srv.handleRequest(rr2, req2)

	if rr2.Header().Get("Content-Security-Policy") == "" {
		t.Error("HTML response should have Content-Security-Policy header")
	}
}

func TestInjectScriptNonces_CaseInsensitive(t *testing.T) {
	html := []byte(`<SCRIPT>code()</SCRIPT>`)
	result := InjectScriptNonces(html, "nonce1")
	if !bytes.Contains(result, []byte(`nonce="nonce1"`)) {
		t.Errorf("should handle uppercase SCRIPT tags, got: %s", result)
	}
}

// ---------- HTTP Handler Tests ----------

func TestHandleRequest_ServesFiles(t *testing.T) {
	// Set up a temp output directory with a file.
	outputDir := t.TempDir()
	writeTestFile(t, outputDir, "index.html", "<html><body><h1>Home</h1></body></html>")

	srv := NewServer(config.Default(), ServeOptions{
		Port:         1313,
		Bind:         "localhost",
		OutputDir:    outputDir,
		NoLiveReload: true,
	})

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	srv.handleRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("<h1>Home</h1>")) {
		t.Error("expected file content in response")
	}
}

func TestHandleRequest_CleanURLs(t *testing.T) {
	outputDir := t.TempDir()
	writeTestFile(t, outputDir, "blog/my-post/index.html", "<html><body><h1>Post</h1></body></html>")

	srv := NewServer(config.Default(), ServeOptions{
		Port:         1313,
		Bind:         "localhost",
		OutputDir:    outputDir,
		NoLiveReload: true,
	})

	tests := []struct {
		name string
		path string
	}{
		{"with trailing slash", "/blog/my-post/"},
		{"without trailing slash", "/blog/my-post"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()
			srv.handleRequest(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}
			if !bytes.Contains(rr.Body.Bytes(), []byte("<h1>Post</h1>")) {
				t.Error("expected post content in response")
			}
		})
	}
}

func TestHandleRequest_404(t *testing.T) {
	outputDir := t.TempDir()

	srv := NewServer(config.Default(), ServeOptions{
		Port:         1313,
		Bind:         "localhost",
		OutputDir:    outputDir,
		NoLiveReload: true,
	})

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rr := httptest.NewRecorder()
	srv.handleRequest(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestHandleRequest_Custom404(t *testing.T) {
	outputDir := t.TempDir()
	writeTestFile(t, outputDir, "404.html", "<html><body><h1>Custom Not Found</h1></body></html>")

	srv := NewServer(config.Default(), ServeOptions{
		Port:         1313,
		Bind:         "localhost",
		OutputDir:    outputDir,
		NoLiveReload: true,
	})

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rr := httptest.NewRecorder()
	srv.handleRequest(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("Custom Not Found")) {
		t.Error("expected custom 404 page content")
	}
}

func TestHandleRequest_LiveReloadInjection(t *testing.T) {
	outputDir := t.TempDir()
	writeTestFile(t, outputDir, "index.html", "<html><body><h1>Home</h1></body></html>")

	srv := NewServer(config.Default(), ServeOptions{
		Port:         1313,
		Bind:         "localhost",
		OutputDir:    outputDir,
		NoLiveReload: false,
	})

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	srv.handleRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("__forge/ws")) {
		t.Error("expected live reload script to be injected")
	}
}

func TestHandleRequest_NoLiveReloadForNonHTML(t *testing.T) {
	outputDir := t.TempDir()
	writeTestFile(t, outputDir, "style.css", "body { color: red; }")

	srv := NewServer(config.Default(), ServeOptions{
		Port:         1313,
		Bind:         "localhost",
		OutputDir:    outputDir,
		NoLiveReload: false,
	})

	req := httptest.NewRequest("GET", "/style.css", nil)
	rr := httptest.NewRecorder()
	srv.handleRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	if bytes.Contains(rr.Body.Bytes(), []byte("__forge/ws")) {
		t.Error("live reload script should not be injected into CSS files")
	}
}

func TestHandleRequest_MIMETypes(t *testing.T) {
	outputDir := t.TempDir()
	writeTestFile(t, outputDir, "style.css", "body{}")
	writeTestFile(t, outputDir, "app.js", "console.log('hello')")
	writeTestFile(t, outputDir, "index.html", "<html></html>")

	srv := NewServer(config.Default(), ServeOptions{
		Port:         1313,
		Bind:         "localhost",
		OutputDir:    outputDir,
		NoLiveReload: true,
	})

	tests := []struct {
		path        string
		contentType string
	}{
		{"/style.css", "text/css"},
		{"/app.js", "text/javascript"},
		{"/index.html", "text/html"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()
			srv.handleRequest(rr, req)

			ct := rr.Header().Get("Content-Type")
			if ct == "" {
				t.Error("expected Content-Type header")
			}
			if !bytes.Contains([]byte(ct), []byte(tt.contentType)) {
				t.Errorf("expected Content-Type containing %q, got %q", tt.contentType, ct)
			}
		})
	}
}

func TestHandleRequest_DirectoryTraversal(t *testing.T) {
	outputDir := t.TempDir()
	writeTestFile(t, outputDir, "index.html", "<html></html>")

	srv := NewServer(config.Default(), ServeOptions{
		Port:         1313,
		Bind:         "localhost",
		OutputDir:    outputDir,
		NoLiveReload: true,
	})

	req := httptest.NewRequest("GET", "/../../../etc/passwd", nil)
	rr := httptest.NewRecorder()
	srv.handleRequest(rr, req)

	// Should not serve files outside outputDir.
	if rr.Code == http.StatusOK && bytes.Contains(rr.Body.Bytes(), []byte("root:")) {
		t.Error("should not serve files outside the output directory")
	}
}

// ---------- Security Header Tests ----------

func TestHandleRequest_SecurityHeaders(t *testing.T) {
	outputDir := t.TempDir()
	writeTestFile(t, outputDir, "index.html", "<html><body><h1>Hi</h1></body></html>")

	srv := NewServer(config.Default(), ServeOptions{
		Port:         1313,
		Bind:         "localhost",
		OutputDir:    outputDir,
		NoLiveReload: true,
	})

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	srv.handleRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	// Check Content-Security-Policy header is present and contains key directives.
	csp := rr.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected Content-Security-Policy header")
	}
	if !bytes.Contains([]byte(csp), []byte("default-src")) {
		t.Error("expected default-src in CSP")
	}
	if !bytes.Contains([]byte(csp), []byte("script-src")) {
		t.Error("expected script-src in CSP")
	}
	if !bytes.Contains([]byte(csp), []byte("nonce-")) {
		t.Error("expected nonce in CSP script-src")
	}

	// Check other security headers.
	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected X-Content-Type-Options: nosniff")
	}
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("expected X-Frame-Options: DENY")
	}
	if rr.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Error("expected Referrer-Policy header")
	}
	if rr.Header().Get("Permissions-Policy") == "" {
		t.Error("expected Permissions-Policy header")
	}
}

func TestHandleRequest_UniqueNoncePerRequest(t *testing.T) {
	outputDir := t.TempDir()
	writeTestFile(t, outputDir, "index.html", "<html><body><h1>Hi</h1></body></html>")

	srv := NewServer(config.Default(), ServeOptions{
		Port:         1313,
		Bind:         "localhost",
		OutputDir:    outputDir,
		NoLiveReload: false,
	})

	req1 := httptest.NewRequest("GET", "/", nil)
	rr1 := httptest.NewRecorder()
	srv.handleRequest(rr1, req1)

	req2 := httptest.NewRequest("GET", "/", nil)
	rr2 := httptest.NewRecorder()
	srv.handleRequest(rr2, req2)

	csp1 := rr1.Header().Get("Content-Security-Policy")
	csp2 := rr2.Header().Get("Content-Security-Policy")

	if csp1 == csp2 {
		t.Error("expected different nonces (and thus different CSP headers) for different requests")
	}
}

// ---------- WebSocket Hub Tests ----------

func TestHub_RegisterUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	// Give hub time to start.
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", hub.ClientCount())
	}
}

func TestHub_BroadcastDoesNotBlock(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	// Broadcasting with no clients should not panic or block.
	done := make(chan struct{})
	go func() {
		hub.Broadcast([]byte("reload"))
		close(done)
	}()

	select {
	case <-done:
		// Success.
	case <-time.After(1 * time.Second):
		t.Error("Broadcast blocked with no clients")
	}
}

func TestHub_StopClosesClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Give hub time to start.
	time.Sleep(10 * time.Millisecond)

	// Stop should not panic.
	hub.Stop()

	// Give time for goroutine to exit.
	time.Sleep(10 * time.Millisecond)
}

// ---------- Watcher Debouncing Tests ----------

func TestWatcher_Debouncing(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial"), 0o644); err != nil {
		t.Fatal(err)
	}

	var callCount atomic.Int32
	var mu sync.Mutex
	var lastCall time.Time

	w := NewWatcher([]string{dir}, 100*time.Millisecond, func() {
		mu.Lock()
		lastCall = time.Now()
		mu.Unlock()
		callCount.Add(1)
	})

	go func() {
		if err := w.Start(); err != nil {
			t.Logf("watcher start error: %v", err)
		}
	}()

	// Give watcher time to start.
	time.Sleep(50 * time.Millisecond)

	// Make several rapid changes.
	for i := range 5 {
		if err := os.WriteFile(testFile, fmt.Appendf(nil, "change %d", i), 0o644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce to settle.
	time.Sleep(300 * time.Millisecond)

	w.Stop()

	// Due to debouncing, we should have significantly fewer callbacks than
	// the number of changes. The exact count depends on timing, but it
	// should be much less than 5.
	count := callCount.Load()
	if count == 0 {
		t.Error("expected at least one onChange callback")
	}
	if count >= 5 {
		t.Errorf("expected debouncing to reduce callbacks, got %d for 5 changes", count)
	}

	mu.Lock()
	_ = lastCall
	mu.Unlock()
}

func TestWatcher_NonexistentPaths(t *testing.T) {
	// Watcher should gracefully handle nonexistent paths.
	w := NewWatcher([]string{"/nonexistent/path/that/does/not/exist"}, 100*time.Millisecond, func() {})

	go func() {
		_ = w.Start()
	}()

	time.Sleep(50 * time.Millisecond)
	w.Stop()
}

func TestWatcher_StopIsIdempotent(t *testing.T) {
	w := NewWatcher([]string{}, 100*time.Millisecond, func() {})

	go func() {
		_ = w.Start()
	}()

	time.Sleep(50 * time.Millisecond)

	// Calling Stop multiple times should not panic.
	w.Stop()
	w.Stop()
}

// ---------- Server Construction Tests ----------

func TestNewServer(t *testing.T) {
	cfg := config.Default()
	opts := ServeOptions{
		Port:         1313,
		Bind:         "localhost",
		OutputDir:    "/tmp/test",
		NoLiveReload: false,
	}

	srv := NewServer(cfg, opts)
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.hub == nil {
		t.Error("expected hub to be initialized")
	}
	if srv.options.Port != 1313 {
		t.Errorf("expected port 1313, got %d", srv.options.Port)
	}
}

// ---------- Helper ----------

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	fullPath := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
