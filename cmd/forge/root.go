package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "forge",
	Short: "A fast, opinionated static site generator",
	Long:  "Forge transforms Markdown content into a beautiful, deployable static website.",
}

func init() {
	rootCmd.PersistentFlags().String("config", "forge.yaml", "path to config file")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose output")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
