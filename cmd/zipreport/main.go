// Package main provides the CLI entry point for ZipReport.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set via ldflags at build time; empty means use the module version
var version = ""

var rootCmd = &cobra.Command{
	Use:   "zipreport",
	Short: "ZipReport CLI - HTML to PDF report generation",
	Long: `ZipReport is a tool for generating PDF reports from Jinja2 HTML templates.

Templates are packaged as .zpt files (ZIP archives) containing:
  - manifest.json: Report metadata and configuration
  - index.html: Main Jinja2 template
  - Assets: CSS, JS, images, fonts, etc.`,
	SilenceErrors: true, // main prints errors once, escaped; cobra would print them raw
}

func main() {
	rootCmd.AddCommand(
		buildCmd,
		listCmd,
		infoCmd,
		debugCmd,
		versionCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, printableError(err.Error()))
		os.Exit(1)
	}
}
