package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/aellingwood/forge/internal/build"
	"github.com/aellingwood/forge/internal/config"
	"github.com/aellingwood/forge/internal/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the development server",
	Long:  "Start a local development server with live reload support.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Load config.
		configPath, _ := cmd.Root().PersistentFlags().GetString("config")
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// 2. Read CLI flags.
		port, _ := cmd.Flags().GetInt("port")
		bind, _ := cmd.Flags().GetString("bind")
		noLiveReload, _ := cmd.Flags().GetBool("no-live-reload")
		drafts, _ := cmd.Flags().GetBool("drafts")
		future, _ := cmd.Flags().GetBool("future")
		verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

		projectRoot, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("determining project root: %w", err)
		}

		outputDir := filepath.Join(projectRoot, "public")

		// 3. Run initial build.
		buildOpts := build.BuildOptions{
			IncludeDrafts: drafts,
			IncludeFuture: future,
			OutputDir:     outputDir,
			Verbose:       verbose,
			BaseURL:       cfg.BaseURL,
			ProjectRoot:   projectRoot,
		}

		builder := build.NewBuilder(cfg, buildOpts)
		result, err := builder.Build()
		if err != nil {
			return fmt.Errorf("initial build failed: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(),
			"Build complete: %d pages rendered, %d files written, %d files copied in %s\n",
			result.PagesRendered,
			result.FilesWritten,
			result.FilesCopied,
			result.Duration.Round(time.Millisecond),
		)

		// 4. Create the server.
		serveOpts := server.ServeOptions{
			Port:          port,
			Bind:          bind,
			OutputDir:     outputDir,
			ProjectRoot:   projectRoot,
			IncludeDrafts: drafts,
			IncludeFuture: future,
			NoLiveReload:  noLiveReload,
			Verbose:       verbose,
		}

		srv := server.NewServer(cfg, serveOpts)

		// 5. Set up file watcher for auto-rebuild.
		watchPaths := []string{
			filepath.Join(projectRoot, "content"),
			filepath.Join(projectRoot, "layouts"),
			filepath.Join(projectRoot, "static"),
			filepath.Join(projectRoot, "assets"),
			filepath.Join(projectRoot, "forge.yaml"),
		}

		watcher := server.NewWatcher(watchPaths, 100*time.Millisecond, func() {
			log.Println("Change detected, rebuilding...")
			rebuildResult, err := builder.Build()
			if err != nil {
				log.Printf("Rebuild failed: %v", err)
				return
			}
			if verbose {
				log.Printf("Rebuild complete: %d pages in %s",
					rebuildResult.PagesRendered,
					rebuildResult.Duration.Round(time.Millisecond),
				)
			}
			srv.NotifyReload()
		})
		srv.SetWatcher(watcher)

		// 6. Handle graceful shutdown.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\nShutting down...")
			cancel()
		}()

		// 7. Start the server (blocks until shutdown).
		if err := srv.Start(ctx); err != nil {
			return fmt.Errorf("server error: %w", err)
		}

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
