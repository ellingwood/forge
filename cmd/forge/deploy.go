package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the site",
	Long:  "Deploy the built site to the configured destination.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Deploying site...")
		return nil
	},
}

func init() {
	deployCmd.Flags().Bool("dry-run", false, "show what would be deployed without deploying")

	rootCmd.AddCommand(deployCmd)
}
