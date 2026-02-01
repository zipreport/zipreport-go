// Print CSS Chrome example demonstrates CSS Paged Media features with Chromium.
//
// This example shows how to:
//   - Use CSS @page rules with margin boxes
//   - Page counters (counter(page), counter(pages))
//   - Page breaks (break-before, break-after, break-inside)
//   - Widows and orphans control
//
// Usage:
//
//	go run main.go output.pdf
package main

import (
	"encoding/json"
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
	fmt.Println("Building print CSS template...")
	zfs, buildResult := report.BuildZipFs("template", nil)
	if !buildResult.Success() {
		fmt.Printf("Error building report: %s\n", buildResult.Error())
		os.Exit(1)
	}

	// Load ReportFile from in-memory zipfs
	zpt, err := report.LoadZipFs(zfs)
	if err != nil {
		fmt.Printf("Error loading report: %v\n", err)
		os.Exit(1)
	}

	// Load data from template's data.json
	dataBytes, err := zpt.GetBytes("data.json")
	if err != nil {
		fmt.Printf("Error loading data.json: %v\n", err)
		os.Exit(1)
	}

	var reportData map[string]interface{}
	if err := json.Unmarshal(dataBytes, &reportData); err != nil {
		fmt.Printf("Error parsing data.json: %v\n", err)
		os.Exit(1)
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
