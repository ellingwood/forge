package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the static site",
	Long:  "Build transforms your content into a complete static website.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Building site...")
		return nil
	},
}

func init() {
	buildCmd.Flags().Bool("drafts", false, "include draft content")
	buildCmd.Flags().Bool("future", false, "include future-dated content")
	buildCmd.Flags().Bool("expired", false, "include expired content")
	buildCmd.Flags().String("baseURL", "", "override base URL")
	buildCmd.Flags().StringP("destination", "d", "public", "output directory")
	buildCmd.Flags().Bool("minify", false, "minify output")

	rootCmd.AddCommand(buildCmd)
}
