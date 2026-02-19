package main

import (
	"os"

	"github.com/aellingwood/forge/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run MCP server over stdio",
	Long:  "Start an MCP (Model Context Protocol) server over stdio, enabling AI clients to query and interact with your Forge site.",
	RunE:  runMCP,
}

func runMCP(cmd *cobra.Command, args []string) error {
	siteDir, _ := cmd.Flags().GetString("source")
	if siteDir == "" {
		var err error
		siteDir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	srv := mcpserver.New(siteDir, version)
	return srv.Run(cmd.Context(), &mcp.StdioTransport{})
}

func init() {
	mcpCmd.Flags().String("source", "", "site root directory (default: current directory)")
	rootCmd.AddCommand(mcpCmd)
}
