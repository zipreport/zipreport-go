// Newsletter example demonstrates MIME email generation with ZipReport.
//
// This example shows how to:
//   - Generate MIME email messages from templates
//   - Embed resources in the email
//   - Use the MIMEReport processor
//
// Usage:
//
//	go run main.go output.eml
package main

import (
	"fmt"
	"os"

	"github.com/zipreport/zipreport-go/pkg/api"
	"github.com/zipreport/zipreport-go/pkg/processor"
	"github.com/zipreport/zipreport-go/pkg/report"
)

func main() {
	// Check command line arguments
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <destination_file.eml>")
		os.Exit(1)
	}
	emlName := os.Args[1]

	// Check if output file already exists
	if _, err := os.Stat(emlName); err == nil {
		fmt.Printf("%s already exists\n", emlName)
		os.Exit(1)
	}

	// Build report from local template directory
	fmt.Println("Building newsletter template...")
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

	// Template data
	reportData := map[string]interface{}{
		"recipient_name": "John Doe",
		"articles": []map[string]string{
			{
				"title":   "Welcome to Our Newsletter",
				"summary": "Thank you for subscribing to our newsletter. We're excited to share updates with you!",
			},
			{
				"title":   "New Features Released",
				"summary": "We've added several new features that will help streamline your workflow.",
			},
			{
				"title":   "Upcoming Events",
				"summary": "Join us for our upcoming webinar series starting next month.",
			},
		},
	}

	// Render using MIME processor
	fmt.Println("Generating MIME email...")
	client := api.NewMIMEReport(
		processor.WithSubject("Monthly Newsletter"),
		processor.WithFrom("newsletter@example.com"),
		processor.WithTo("john.doe@example.com"),
	)
	result := client.RenderDefaults(zpt, reportData, nil)

	if !result.Success {
		fmt.Printf("An error occurred while generating the email: %s\n", result.Error)
		os.Exit(1)
	}

	// Save MIME message to file
	if err := os.WriteFile(emlName, result.Report, 0644); err != nil {
		fmt.Printf("Failed to save email: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Email message generated to %s\n", emlName)
}
