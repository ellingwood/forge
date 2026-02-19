package mcpserver

import (
	"context"
	"path/filepath"
	"time"

	"github.com/aellingwood/forge/internal/server"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// startWatcher starts a file watcher that marks the site context dirty and
// sends resource update notifications when content files change.
func (fs *ForgeServer) startWatcher(ctx context.Context) error {
	watchPaths := []string{
		filepath.Join(fs.siteDir, "content"),
		filepath.Join(fs.siteDir, "forge.yaml"),
		filepath.Join(fs.siteDir, "layouts"),
		filepath.Join(fs.siteDir, "data"),
	}

	watcher := server.NewWatcher(watchPaths, 500*time.Millisecond, func() {
		fs.ctx.MarkDirty()
		// Notify clients that resources have changed
		_ = fs.server.ResourceUpdated(ctx, &mcp.ResourceUpdatedNotificationParams{
			URI: "forge://content/pages",
		})
	})

	go func() {
		if err := watcher.Start(); err != nil {
			// Non-fatal: file watching is best-effort
			return
		}
		<-ctx.Done()
		watcher.Stop()
	}()

	return nil
}
