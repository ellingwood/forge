package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aellingwood/forge/internal/config"
	"github.com/aellingwood/forge/internal/content"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List content",
	Long:  "List content by type: drafts, future, or expired.",
}

var listDraftsCmd = &cobra.Command{
	Use:   "drafts",
	Short: "List draft content",
	RunE: func(cmd *cobra.Command, args []string) error {
		pages, err := discoverAllContent(cmd)
		if err != nil {
			return err
		}

		// Filter to only draft pages.
		var drafts []*content.Page
		for _, p := range pages {
			if p.Draft {
				drafts = append(drafts, p)
			}
		}

		if len(drafts) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No draft content found.")
			return nil
		}

		printPageList(cmd, drafts)
		return nil
	},
}

var listFutureCmd = &cobra.Command{
	Use:   "future",
	Short: "List future-dated content",
	RunE: func(cmd *cobra.Command, args []string) error {
		pages, err := discoverAllContent(cmd)
		if err != nil {
			return err
		}

		// Filter to only future-dated pages.
		now := time.Now()
		var future []*content.Page
		for _, p := range pages {
			if p.Date.After(now) {
				future = append(future, p)
			}
		}

		if len(future) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No future-dated content found.")
			return nil
		}

		printPageList(cmd, future)
		return nil
	},
}

var listExpiredCmd = &cobra.Command{
	Use:   "expired",
	Short: "List expired content",
	RunE: func(cmd *cobra.Command, args []string) error {
		pages, err := discoverAllContent(cmd)
		if err != nil {
			return err
		}

		// Filter to only expired pages.
		now := time.Now()
		var expired []*content.Page
		for _, p := range pages {
			if !p.ExpiryDate.IsZero() && p.ExpiryDate.Before(now) {
				expired = append(expired, p)
			}
		}

		if len(expired) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No expired content found.")
			return nil
		}

		printPageList(cmd, expired)
		return nil
	},
}

// discoverAllContent loads config and discovers all content pages.
func discoverAllContent(cmd *cobra.Command) ([]*content.Page, error) {
	configPath, _ := cmd.Root().PersistentFlags().GetString("config")
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	projectRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("determining project root: %w", err)
	}

	contentDir := filepath.Join(projectRoot, "content")
	pages, err := content.Discover(contentDir, cfg)
	if err != nil {
		return nil, fmt.Errorf("discovering content: %w", err)
	}

	return pages, nil
}

// printPageList prints a formatted table of pages with title, date, and URL.
func printPageList(cmd *cobra.Command, pages []*content.Page) {
	out := cmd.OutOrStdout()
	for _, p := range pages {
		dateStr := ""
		if !p.Date.IsZero() {
			dateStr = p.Date.Format("2006-01-02")
		}
		title := p.Title
		if title == "" {
			title = "(untitled)"
		}
		fmt.Fprintf(out, "%s  %s  %s\n", dateStr, title, p.URL)
	}
}

func init() {
	listCmd.AddCommand(listDraftsCmd)
	listCmd.AddCommand(listFutureCmd)
	listCmd.AddCommand(listExpiredCmd)

	rootCmd.AddCommand(listCmd)
}
