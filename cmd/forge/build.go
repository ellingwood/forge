package main

import (
	"fmt"
	"os"

	"github.com/aellingwood/forge/internal/build"
	"github.com/aellingwood/forge/internal/config"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the static site",
	Long:  "Build transforms your content into a complete static website.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Load config.
		configPath, _ := cmd.Root().PersistentFlags().GetString("config")
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// 2. Apply CLI flag overrides.
		overrides := make(map[string]any)
		if baseURL, _ := cmd.Flags().GetString("baseURL"); baseURL != "" {
			overrides["baseURL"] = baseURL
		}
		if minify, _ := cmd.Flags().GetBool("minify"); minify {
			overrides["minify"] = minify
		}
		cfg.WithOverrides(overrides)

		// 3. Build options from flags.
		drafts, _ := cmd.Flags().GetBool("drafts")
		future, _ := cmd.Flags().GetBool("future")
		expired, _ := cmd.Flags().GetBool("expired")
		destination, _ := cmd.Flags().GetString("destination")
		verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
		minify, _ := cmd.Flags().GetBool("minify")

		projectRoot, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("determining project root: %w", err)
		}

		opts := build.BuildOptions{
			IncludeDrafts:  drafts,
			IncludeFuture:  future,
			IncludeExpired: expired,
			OutputDir:      destination,
			Verbose:        verbose,
			Minify:         minify,
			BaseURL:        cfg.BaseURL,
			ProjectRoot:    projectRoot,
		}

		// 4. Create builder and run the build.
		builder := build.NewBuilder(cfg, opts)
		result, err := builder.Build()
		if err != nil {
			return fmt.Errorf("build failed: %w", err)
		}

		// 5. Print build result summary.
		fmt.Fprintf(cmd.OutOrStdout(),
			"Build complete: %d pages rendered, %d files written, %d files copied in %s\n",
			result.PagesRendered,
			result.FilesWritten,
			result.FilesCopied,
			result.Duration.Round(1_000_000), // round to milliseconds
		)

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
