package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print the resolved configuration",
	Long:  "Print the fully resolved configuration after merging all sources.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Resolved configuration:")
		fmt.Println("  (placeholder — not yet implemented)")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
