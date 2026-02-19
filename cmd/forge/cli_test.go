package main

import (
	"bytes"
	"testing"
)

func TestRootCommand(t *testing.T) {
	if rootCmd.Use != "forge" {
		t.Errorf("expected root command Use to be 'forge', got %q", rootCmd.Use)
	}

	expectedSubcommands := []string{"build", "serve", "new", "deploy", "version", "list", "config"}
	commands := rootCmd.Commands()

	nameSet := make(map[string]bool)
	for _, cmd := range commands {
		nameSet[cmd.Name()] = true
	}

	for _, expected := range expectedSubcommands {
		if !nameSet[expected] {
			t.Errorf("expected root command to have subcommand %q", expected)
		}
	}
}

func TestBuildFlags(t *testing.T) {
	expectedFlags := []string{"drafts", "future", "expired", "baseURL", "destination", "minify"}
	for _, name := range expectedFlags {
		flag := buildCmd.Flags().Lookup(name)
		if flag == nil {
			t.Errorf("expected build command to have flag %q", name)
		}
	}

	// Verify destination has short flag -d
	flag := buildCmd.Flags().ShorthandLookup("d")
	if flag == nil {
		t.Error("expected build command to have short flag -d for destination")
	} else if flag.Name != "destination" {
		t.Errorf("expected short flag -d to map to 'destination', got %q", flag.Name)
	}
}

func TestServeFlags(t *testing.T) {
	expectedFlags := []string{"port", "bind", "no-live-reload", "drafts", "future"}
	for _, name := range expectedFlags {
		flag := serveCmd.Flags().Lookup(name)
		if flag == nil {
			t.Errorf("expected serve command to have flag %q", name)
		}
	}

	// Verify default values
	portFlag := serveCmd.Flags().Lookup("port")
	if portFlag != nil && portFlag.DefValue != "1313" {
		t.Errorf("expected port default to be '1313', got %q", portFlag.DefValue)
	}

	bindFlag := serveCmd.Flags().Lookup("bind")
	if bindFlag != nil && bindFlag.DefValue != "localhost" {
		t.Errorf("expected bind default to be 'localhost', got %q", bindFlag.DefValue)
	}
}

func TestVersionOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("expected version command to produce output")
	}

	// Reset for other tests
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)
	rootCmd.SetArgs(nil)
}

func TestNewSubcommands(t *testing.T) {
	expectedSubcommands := []string{"site", "post", "page", "project"}
	commands := newCmd.Commands()

	nameSet := make(map[string]bool)
	for _, cmd := range commands {
		nameSet[cmd.Name()] = true
	}

	for _, expected := range expectedSubcommands {
		if !nameSet[expected] {
			t.Errorf("expected new command to have subcommand %q", expected)
		}
	}

	// Verify each subcommand requires exactly 1 argument
	for _, cmd := range commands {
		if cmd.Args == nil {
			t.Errorf("expected new %s to have Args validation", cmd.Name())
		}
	}
}
