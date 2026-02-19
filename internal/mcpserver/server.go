package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ForgeServer is the MCP server for Forge.
type ForgeServer struct {
	server    *mcp.Server
	siteDir   string
	ctx       *SiteContext
	lastBuild *BuildResultDetail
	version   string
}

// New creates a new ForgeServer for the given site directory.
func New(siteDir, version string) *ForgeServer {
	fs := &ForgeServer{
		siteDir: siteDir,
		version: version,
	}
	fs.ctx = NewSiteContext(siteDir)

	fs.server = mcp.NewServer(
		&mcp.Implementation{
			Name:    "forge",
			Version: version,
		},
		nil,
	)

	fs.registerResources()
	fs.registerTools()
	fs.registerPrompts()

	return fs
}

// Run starts the MCP server on the given transport.
func (fs *ForgeServer) Run(ctx context.Context, transport mcp.Transport) error {
	if err := fs.startWatcher(ctx); err != nil {
		// Non-fatal: log but continue
		_ = err
	}
	return fs.server.Run(ctx, transport)
}

func ptr[T any](v T) *T {
	return &v
}
