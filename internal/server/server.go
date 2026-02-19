package server

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aellingwood/forge/internal/config"
)

// ServeOptions contains the configurable settings for the development server.
type ServeOptions struct {
	Port          int
	Bind          string
	OutputDir     string
	ProjectRoot   string
	IncludeDrafts bool
	IncludeFuture bool
	NoLiveReload  bool
	Verbose       bool
}

// Server is the development HTTP server that serves static files, handles
// clean URLs, and provides WebSocket-based live reloading.
type Server struct {
	config  *config.SiteConfig
	options ServeOptions
	hub     *Hub
	watcher *Watcher
	server  *http.Server
}

// NewServer creates a new Server with the given site configuration and options.
func NewServer(cfg *config.SiteConfig, opts ServeOptions) *Server {
	return &Server{
		config:  cfg,
		options: opts,
		hub:     NewHub(),
	}
}

// Start starts the HTTP server, WebSocket hub, and file watcher. It blocks
// until the provided context is cancelled or the server is stopped.
func (s *Server) Start(ctx context.Context) error {
	// Start the WebSocket hub.
	go s.hub.Run()

	// Build HTTP handler.
	mux := http.NewServeMux()
	mux.HandleFunc("/__forge/ws", s.hub.HandleWS)
	mux.HandleFunc("/", s.handleRequest)

	addr := fmt.Sprintf("%s:%d", s.options.Bind, s.options.Port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start the file watcher if paths are provided.
	if s.watcher != nil {
		go func() {
			if err := s.watcher.Start(); err != nil {
				log.Printf("watcher error: %v", err)
			}
		}()
	}

	fmt.Printf("Serving at http://%s:%d\n", s.options.Bind, s.options.Port)

	// Listen for context cancellation to trigger graceful shutdown.
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}

	if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the server, watcher, and hub.
func (s *Server) Stop() error {
	if s.watcher != nil {
		s.watcher.Stop()
	}
	s.hub.Stop()
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// SetWatcher configures the file watcher for the server.
func (s *Server) SetWatcher(w *Watcher) {
	s.watcher = w
}

// NotifyReload sends a reload message to all connected WebSocket clients.
func (s *Server) NotifyReload() {
	s.hub.Broadcast([]byte("reload"))
}

// handleRequest serves static files from the output directory with support
// for clean URLs and optional live reload script injection.
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path

	// Try to resolve the file path.
	filePath := s.resolveFilePath(urlPath)
	if filePath == "" {
		s.handle404(w, r)
		return
	}

	// Read the file.
	data, err := os.ReadFile(filePath)
	if err != nil {
		s.handle404(w, r)
		return
	}

	// Determine content type.
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Inject live reload script for HTML files.
	if !s.options.NoLiveReload && isHTML(ext, contentType) {
		data = InjectLiveReload(data, s.options.Port)
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// resolveFilePath maps a URL path to an actual file in the output directory.
// It handles clean URLs by checking for index.html in directories.
func (s *Server) resolveFilePath(urlPath string) string {
	outputDir := s.options.OutputDir

	// Clean the path to prevent directory traversal.
	cleaned := filepath.Clean(urlPath)
	if strings.Contains(cleaned, "..") {
		return ""
	}

	// Build the full path.
	fullPath := filepath.Join(outputDir, filepath.FromSlash(cleaned))

	// Check if the file exists directly.
	if info, err := os.Stat(fullPath); err == nil {
		if !info.IsDir() {
			return fullPath
		}
		// If it's a directory, try index.html.
		indexPath := filepath.Join(fullPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			return indexPath
		}
		return ""
	}

	// Try appending .html for extensionless URLs.
	htmlPath := fullPath + ".html"
	if _, err := os.Stat(htmlPath); err == nil {
		return htmlPath
	}

	// Try treating the path as a directory with index.html (clean URL).
	indexPath := filepath.Join(fullPath, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		return indexPath
	}

	return ""
}

// handle404 serves a 404 page. If a custom 404.html exists in the output
// directory, it is served; otherwise a plain text message is returned.
func (s *Server) handle404(w http.ResponseWriter, r *http.Request) {
	// Try to serve custom 404 page.
	notFoundPath := filepath.Join(s.options.OutputDir, "404.html")
	data, err := os.ReadFile(notFoundPath)
	if err == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(data)
		return
	}

	http.Error(w, "404 page not found", http.StatusNotFound)
}

// isHTML returns true if the file extension or content type indicates HTML.
func isHTML(ext, contentType string) bool {
	if ext == ".html" || ext == ".htm" {
		return true
	}
	return bytes.Contains([]byte(contentType), []byte("text/html"))
}
