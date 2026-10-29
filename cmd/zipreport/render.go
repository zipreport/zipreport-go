package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zipreport/zipreport-go/pkg/api"
	"github.com/zipreport/zipreport-go/pkg/processor"
	"github.com/zipreport/zipreport-go/pkg/report"
	"github.com/zipreport/zipreport-go/pkg/template"
)

var renderCmd = &cobra.Command{
	Use:   "render <source>",
	Short: "Render a report to HTML or a MIME email",
	Long: `Render renders a report with the given data and writes the result to a file or stdout.

The source can be a .zpt file or a template directory.

Formats:
  html  the rendered index only; relative asset links are not copied
  mime  a MIME email with the report's assets embedded

Without -o the rendered document is written to stdout unmodified, like cat.
Manifest params are required: pass them with --data.

Example:
  zipreport render report.zpt -d data.json -o report.html
  zipreport render ./my-template -d data.json -f mime --from a@example.com --to b@example.com -o mail.eml
  zipreport render report.zpt -d data.json > report.html`,
	Args: cobra.ExactArgs(1),
	RunE: runRender,
}

var (
	renderData    string
	renderFormat  string
	renderOutput  string
	renderSubject string
	renderFrom    string
	renderTo      []string
)

func init() {
	renderCmd.Flags().StringVarP(&renderData, "data", "d", "", "JSON file with template data (a JSON object)")
	renderCmd.Flags().StringVarP(&renderFormat, "format", "f", "html", "Output format: html or mime")
	renderCmd.Flags().StringVarP(&renderOutput, "output", "o", "", "Output file (default: stdout)")
	renderCmd.Flags().StringVar(&renderSubject, "subject", "", "MIME Subject header (default: manifest title)")
	renderCmd.Flags().StringVar(&renderFrom, "from", "", "MIME From header")
	renderCmd.Flags().StringArrayVar(&renderTo, "to", nil, "MIME To header (repeatable)")
}

// loadRenderData reads --data with the same rules as data.json
func loadRenderData(path string) (map[string]interface{}, error) {
	data := map[string]interface{}{}
	if path == "" {
		return data, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read data file: %w", err)
	}
	// Same rules as data.json: a byte-order mark is ignored and an empty file means no data
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	if len(bytes.TrimSpace(raw)) == 0 {
		return data, nil
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("invalid data file %s: %w", path, err)
	}
	return data, nil
}

func runRender(cmd *cobra.Command, args []string) error {
	if renderFormat != "html" && renderFormat != "mime" {
		return fmt.Errorf("unknown format %q (use html or mime)", renderFormat)
	}
	data, err := loadRenderData(renderData)
	if err != nil {
		return err
	}
	zpt, err := report.Load(args[0])
	if err != nil {
		return fmt.Errorf("failed to load report: %w", err)
	}

	var out []byte
	switch renderFormat {
	case "html":
		// Render a copy so nothing is written to the template directory
		s, err := template.NewMiyaRender(zpt.Clone()).Render(data)
		if err != nil {
			return fmt.Errorf("render failed: %w", err)
		}
		out = []byte(s)
	case "mime":
		res := api.NewMIMEReport(
			processor.WithSubject(renderSubject),
			processor.WithFrom(renderFrom),
			processor.WithTo(renderTo...),
		).RenderDefaults(zpt, data, nil)
		if !res.Success {
			return fmt.Errorf("render failed: %s", res.Error)
		}
		out = res.Report
	}

	// Nothing is written unless rendering succeeded
	if renderOutput != "" {
		return os.WriteFile(renderOutput, out, 0644)
	}
	_, err = os.Stdout.Write(out)
	return err
}
