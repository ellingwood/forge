package main

import (
	"fmt"

	"github.com/aellingwood/forge/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print the resolved configuration",
	Long:  "Print the fully resolved configuration after merging all sources.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Load config.
		configPath, _ := cmd.Root().PersistentFlags().GetString("config")
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// 2. Marshal to YAML and print.
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("marshaling config: %w", err)
		}

		fmt.Fprint(cmd.OutOrStdout(), string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
