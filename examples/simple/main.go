// Simple example demonstrates basic ZipReport usage.
//
// This example shows how to:
//   - Load a report from a template directory
//   - Provide template data
//   - Render to PDF using zipreport-server
//
// Usage:
//
//	go run main.go output.pdf
package main

import (
	"fmt"
	"os"

	"github.com/zipreport/zipreport-go/pkg/api"
	"github.com/zipreport/zipreport-go/pkg/report"
)

func main() {
	// Get configuration from environment
	serverURL := os.Getenv("ZIPREPORT_URL")
	if serverURL == "" {
		serverURL = "https://127.0.0.1:6543"
	}
	apiKey := os.Getenv("ZIPREPORT_KEY")
	if apiKey == "" {
		apiKey = "somePassword"
	}

	// Check command line arguments
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <destination_file.pdf>")
		os.Exit(1)
	}
	pdfName := os.Args[1]

	// Check if output file already exists
	if _, err := os.Stat(pdfName); err == nil {
		fmt.Printf("%s already exists\n", pdfName)
		os.Exit(1)
	}

	// Build report from local template directory
	fmt.Println("Building report template...")
	zfs, buildResult := report.BuildZipFs("template", nil)
	if !buildResult.Success() {
		fmt.Printf("Error building report: %s\n", buildResult.Error())
		os.Exit(1)
	}

	// Load ReportFile from in-memory zipfs
	zpt, err := report.LoadZipFs(zfs)
	if err != nil {
		fmt.Printf("Failed to load report: %v\n", err)
		os.Exit(1)
	}

	// Template variables
	reportData := map[string]interface{}{
		"title":       "Example report using Jinja templating",
		"color_list":  []string{"red", "blue", "green"},
		"description": "a long text field with some filler description so the page isn't that empty",
	}

	// Render using zipreport-server processor
	fmt.Println("Generating PDF...")
	client := api.NewZipReport(serverURL, apiKey)
	result := client.RenderDefaults(zpt, reportData, nil)

	if !result.Success {
		fmt.Printf("An error occurred while generating the report: %s\n", result.Error)
		os.Exit(1)
	}

	// Save PDF to file
	if err := os.WriteFile(pdfName, result.Report, 0644); err != nil {
		fmt.Printf("Failed to save PDF: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Report generated to %s\n", pdfName)
}
