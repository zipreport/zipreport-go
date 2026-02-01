package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zipreport/zipreport-go/pkg/report"
)

var infoCmd = &cobra.Command{
	Use:   "info <report.zpt>",
	Short: "Display report metadata",
	Long: `Info displays the metadata from a .zpt file's manifest.

Example:
  zipreport info report.zpt`,
	Args: cobra.ExactArgs(1),
	RunE: runInfo,
}

func runInfo(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Load report
	r, err := report.Load(filename)
	if err != nil {
		return fmt.Errorf("failed to load report: %w", err)
	}

	// Display info
	manifest := r.GetManifest()

	fmt.Printf("Report: %s\n", filename)
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Title:       %s\n", manifest.Title)
	fmt.Printf("Author:      %s\n", manifest.Author)
	fmt.Printf("Version:     %s\n", manifest.Version)
	fmt.Printf("Description: %s\n", manifest.Description)

	if len(manifest.Params) > 0 {
		fmt.Printf("Parameters:  %s\n", strings.Join(manifest.Params, ", "))
	} else {
		fmt.Println("Parameters:  (none)")
	}

	fmt.Printf("Use JS Event: %v\n", manifest.UseJSEvent)

	if manifest.Report != "" {
		fmt.Printf("Output File: %s\n", manifest.Report)
	}

	return nil
}
