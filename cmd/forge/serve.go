package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
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
			"Build complete: %d pages in %s\n\n%s\n",
			result.PagesRendered,
			result.Duration.Round(time.Millisecond),
			renderTree(result.Pages),
		)

		// 3b. Start Tailwind CSS watch mode (if globals.css exists).
		themeName := cfg.Theme
		if themeName == "" {
			themeName = "default"
		}
		themePath := filepath.Join(projectRoot, "themes", themeName)
		cssInput := filepath.Join(themePath, "static", "css", "globals.css")
		var tailwindCancel func()
		if _, err := os.Stat(cssInput); err == nil {
			cssOutput := filepath.Join(outputDir, "css", "style.css")
			tb := &build.TailwindBuilder{}
			if _, binErr := tb.EnsureBinary(build.TailwindVersion); binErr != nil {
				log.Printf("warning: could not download Tailwind CSS binary: %v (skipping CSS watch mode)", binErr)
			} else {
				cancelFn, watchErr := tb.Watch(cssInput, cssOutput, projectRoot)
				if watchErr != nil {
					log.Printf("warning: could not start Tailwind CSS watcher: %v", watchErr)
				} else {
					tailwindCancel = cancelFn
				}
			}
		}

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
			log.Printf("Rebuild complete: %d pages in %s\n\n%s",
				rebuildResult.PagesRendered,
				rebuildResult.Duration.Round(time.Millisecond),
				renderTree(rebuildResult.Pages),
			)
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
			if tailwindCancel != nil {
				tailwindCancel()
			}
			cancel()
		}()

		// 7. Start the server (blocks until shutdown).
		if err := srv.Start(ctx); err != nil {
			return fmt.Errorf("server error: %w", err)
		}

		return nil
	},
}

// treeNode is a node in an ordered URL tree.
type treeNode struct {
	children   map[string]*treeNode
	childOrder []string
}

// renderTree produces a pretty-printed tree of URL paths.
// The root "/" is always the tree root; sub-paths branch from it.
func renderTree(pages []string) string {
	sorted := make([]string, len(pages))
	copy(sorted, pages)
	sort.Strings(sorted)

	root := &treeNode{children: make(map[string]*treeNode)}

	for _, url := range sorted {
		// Strip leading slash and split into segments.
		trimmed := strings.TrimPrefix(url, "/")
		if trimmed == "" {
			continue // root itself; always shown as "/"
		}
		// Remove trailing index.html if present (for directory URLs).
		trimmed = strings.TrimSuffix(trimmed, "index.html")
		trimmed = strings.TrimSuffix(trimmed, "/")
		if trimmed == "" {
			continue
		}
		segments := strings.Split(trimmed, "/")

		cur := root
		for _, seg := range segments {
			if _, exists := cur.children[seg]; !exists {
				cur.children[seg] = &treeNode{children: make(map[string]*treeNode)}
				cur.childOrder = append(cur.childOrder, seg)
			}
			cur = cur.children[seg]
		}
	}

	var sb strings.Builder
	sb.WriteString("/\n")
	writeTreeNodes(&sb, root, "")
	return strings.TrimRight(sb.String(), "\n")
}

func writeTreeNodes(sb *strings.Builder, node *treeNode, prefix string) {
	for i, name := range node.childOrder {
		child := node.children[name]
		isLast := i == len(node.childOrder)-1

		connector := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		label := name
		if len(child.children) > 0 || isDirectoryURL(name) {
			label = name + "/"
		}
		sb.WriteString(prefix + connector + label + "\n")
		writeTreeNodes(sb, child, childPrefix)
	}
}

// isDirectoryURL returns true if the segment looks like a directory (no extension).
func isDirectoryURL(seg string) bool {
	return !strings.Contains(seg, ".")
}

func init() {
	serveCmd.Flags().Int("port", 1313, "server port")
	serveCmd.Flags().String("bind", "localhost", "bind address")
	serveCmd.Flags().Bool("no-live-reload", false, "disable live reload")
	serveCmd.Flags().Bool("drafts", false, "include draft content")
	serveCmd.Flags().Bool("future", false, "include future-dated content")

	rootCmd.AddCommand(serveCmd)
}
