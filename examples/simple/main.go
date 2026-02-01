// Package main demonstrates basic ZipReport usage.
//
// This example shows how to:
//   - Load a report from a .zpt file
//   - Provide template data
//   - Render to PDF using zipreport-server
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

	// Find the report file - try multiple locations
	reportPath := "simple.zpt"
	possiblePaths := []string{
		"simple.zpt",
		"examples/simple/simple.zpt",
		"./examples/simple/simple.zpt",
	}

	found := false
	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			reportPath = p
			found = true
			break
		}
	}

	if !found {
		fmt.Println("Missing report file. Build it with: zipreport build <template_dir> simple.zpt")
		os.Exit(1)
	}

	// Load report from file
	zpt, err := report.LoadFile(reportPath)
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
