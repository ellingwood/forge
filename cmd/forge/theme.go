package main

import (
	"fmt"

	"github.com/aellingwood/forge/embedded"
	"github.com/aellingwood/forge/internal/scaffold"
	"github.com/spf13/cobra"
)

var themeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Manage themes",
	Long:  "Commands for managing site themes.",
}

var themeUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the default theme to the latest embedded version",
	Long: `Re-extract the default theme from the Forge binary into themes/default/,
overwriting existing files. This brings your on-disk theme in sync with
the version embedded in the current forge binary.

Run this from the site root (the directory containing forge.yaml).`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := scaffold.RefreshTheme(".", embedded.DefaultTheme); err != nil {
			return fmt.Errorf("updating theme: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Default theme updated successfully.")
		return nil
	},
}

func init() {
	themeCmd.AddCommand(themeUpdateCmd)
	rootCmd.AddCommand(themeCmd)
}
