package main

import (
	"fmt"

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
		fmt.Println("Listing draft content...")
		return nil
	},
}

var listFutureCmd = &cobra.Command{
	Use:   "future",
	Short: "List future-dated content",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Listing future-dated content...")
		return nil
	},
}

var listExpiredCmd = &cobra.Command{
	Use:   "expired",
	Short: "List expired content",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Listing expired content...")
		return nil
	},
}

func init() {
	listCmd.AddCommand(listDraftsCmd)
	listCmd.AddCommand(listFutureCmd)
	listCmd.AddCommand(listExpiredCmd)

	rootCmd.AddCommand(listCmd)
}
