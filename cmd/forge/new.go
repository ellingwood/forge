package main

import (
	"fmt"

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
		fmt.Printf("Creating new site: %s\n", args[0])
		return nil
	},
}

var newPostCmd = &cobra.Command{
	Use:   "post <name>",
	Short: "Create a new post",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Creating new post: %s\n", args[0])
		return nil
	},
}

var newPageCmd = &cobra.Command{
	Use:   "page <name>",
	Short: "Create a new page",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Creating new page: %s\n", args[0])
		return nil
	},
}

var newProjectCmd = &cobra.Command{
	Use:   "project <name>",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Creating new project: %s\n", args[0])
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
