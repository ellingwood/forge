package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the development server",
	Long:  "Start a local development server with live reload support.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Starting dev server...")
		return nil
	},
}

func init() {
	serveCmd.Flags().Int("port", 1313, "server port")
	serveCmd.Flags().String("bind", "localhost", "bind address")
	serveCmd.Flags().Bool("no-live-reload", false, "disable live reload")
	serveCmd.Flags().Bool("drafts", false, "include draft content")
	serveCmd.Flags().Bool("future", false, "include future-dated content")

	rootCmd.AddCommand(serveCmd)
}
