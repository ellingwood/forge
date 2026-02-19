package main

import (
	"fmt"

	"github.com/aellingwood/forge/embedded"
	"github.com/aellingwood/forge/internal/scaffold"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create new content",
	Long:  "Create a new site, post, page, or project.",
}

var newSiteCmd = &cobra.Command{
	Use:   "site <name>",
	Short: "Create a new site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := scaffold.NewSite(name, embedded.DefaultTheme); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Site created: %s/\n", name)
		return nil
	},
}

var newPostCmd = &cobra.Command{
	Use:   "post <name>",
	Short: "Create a new post",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]
		if err := scaffold.NewPost(title); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Post created: %s\n", scaffold.CreatedPostPath(title))
		return nil
	},
}

var newPageCmd = &cobra.Command{
	Use:   "page <name>",
	Short: "Create a new page",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]
		if err := scaffold.NewPage(title); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Page created: %s\n", scaffold.CreatedPagePath(title))
		return nil
	},
}

var newProjectCmd = &cobra.Command{
	Use:   "project <name>",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]
		if err := scaffold.NewProject(title); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Project created: %s\n", scaffold.CreatedProjectPath(title))
		return nil
	},
}

func init() {
	newCmd.AddCommand(newSiteCmd)
	newCmd.AddCommand(newPostCmd)
	newCmd.AddCommand(newPageCmd)
	newCmd.AddCommand(newProjectCmd)

	rootCmd.AddCommand(newCmd)
}
