package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  "Print the version, commit hash, and build date of Forge.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStdout(), "forge %s\n", version)
		fmt.Fprintf(cmd.OutOrStdout(), "  commit: %s\n", commit)
		fmt.Fprintf(cmd.OutOrStdout(), "  built:  %s\n", date)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
