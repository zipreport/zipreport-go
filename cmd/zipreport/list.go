package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [directory]",
	Short: "List .zpt files in a directory",
	Long: `List finds and displays all .zpt files in the specified directory.

If no directory is specified, the current directory is used.

Example:
  zipreport list
  zipreport list ./reports`,
	Args: cobra.MaximumNArgs(1),
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	// Verify directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("cannot access directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	// Find .zpt files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("cannot read directory: %w", err)
	}

	var zptFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".zpt" {
			zptFiles = append(zptFiles, entry.Name())
		}
	}

	if len(zptFiles) == 0 {
		fmt.Println("No .zpt files found")
		return nil
	}

	fmt.Printf("Found %d .zpt file(s) in %s:\n", len(zptFiles), dir)
	for _, f := range zptFiles {
		fmt.Printf("  %s\n", f)
	}

	return nil
}
