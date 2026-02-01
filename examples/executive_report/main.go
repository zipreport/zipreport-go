// Executive Report example demonstrates chart generation with ZipReport.
//
// This example shows how to:
//   - Generate charts dynamically at render time using gonum/plot
//   - Use ReportJob for custom configuration
//   - Enable JavaScript events for PagedJS
//
// Usage:
//
//	go run main.go output.pdf
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"os"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"

	"github.com/zipreport/zipreport-go/pkg/api"
	"github.com/zipreport/zipreport-go/pkg/report"
)

// generateRevenueChart creates a bar chart for revenue data using gonum/plot.
func generateRevenueChart(data interface{}) []byte {
	chartData, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	quarters, _ := chartData["quarters"].([]interface{})
	revenue, _ := chartData["revenue"].([]interface{})

	if len(quarters) == 0 || len(revenue) == 0 {
		return nil
	}

	p := plot.New()
	p.Title.Text = "Quarterly Revenue"
	p.Title.TextStyle.Font.Size = vg.Points(12)
	p.X.Tick.Label.Font.Size = vg.Points(9)
	p.Y.Tick.Label.Font.Size = vg.Points(9)
	p.Y.Label.Text = "Revenue ($M)"
	p.Y.Label.TextStyle.Font.Size = vg.Points(10)

	vals := make(plotter.Values, len(revenue))
	for i, v := range revenue {
		vals[i] = toFloat(v)
	}

	bars, _ := plotter.NewBarChart(vals, vg.Points(40))
	bars.Color = color.RGBA{R: 14, G: 165, B: 233, A: 255} // sky-500

	p.Add(bars)

	var labels []string
	for _, q := range quarters {
		if s, ok := q.(string); ok {
			labels = append(labels, s)
		}
	}
	p.NominalX(labels...)

	w, err := p.WriterTo(vg.Points(560), vg.Points(280), "png")
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	w.WriteTo(&buf)
	return buf.Bytes()
}

// generatePerformanceChart creates a grouped bar chart using gonum/plot.
func generatePerformanceChart(data interface{}) []byte {
	chartData, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	categories, _ := chartData["categories"].([]interface{})
	current, _ := chartData["current"].([]interface{})
	target, _ := chartData["target"].([]interface{})

	if len(categories) == 0 {
		return nil
	}

	p := plot.New()
	p.Title.Text = "Performance vs Target"
	p.Title.TextStyle.Font.Size = vg.Points(12)
	p.X.Tick.Label.Font.Size = vg.Points(8)
	p.Y.Tick.Label.Font.Size = vg.Points(9)
	p.Y.Label.Text = "Score (%)"
	p.Y.Label.TextStyle.Font.Size = vg.Points(10)
	p.Legend.TextStyle.Font.Size = vg.Points(9)
	p.Legend.Top = true

	n := len(categories)
	w := vg.Points(20)

	currentVals := make(plotter.Values, n)
	targetVals := make(plotter.Values, n)

	for i := 0; i < n; i++ {
		if i < len(current) {
			currentVals[i] = toFloat(current[i])
		}
		if i < len(target) {
			targetVals[i] = toFloat(target[i])
		}
	}

	bars1, _ := plotter.NewBarChart(currentVals, w)
	bars1.Color = color.RGBA{R: 14, G: 165, B: 233, A: 255} // sky-500
	bars1.Offset = -w / 2

	bars2, _ := plotter.NewBarChart(targetVals, w)
	bars2.Color = color.RGBA{R: 16, G: 185, B: 129, A: 200} // emerald-500
	bars2.Offset = w / 2

	p.Add(bars1, bars2)
	p.Legend.Add("Current", bars1)
	p.Legend.Add("Target", bars2)

	var labels []string
	for _, c := range categories {
		if s, ok := c.(string); ok {
			labels = append(labels, s)
		}
	}
	p.NominalX(labels...)

	wr, err := p.WriterTo(vg.Points(560), vg.Points(320), "png")
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	wr.WriteTo(&buf)
	return buf.Bytes()
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func main() {
	// Get configuration from environment
	serverURL := os.Getenv("ZIPREPORT_URL")
	if serverURL == "" {
		serverURL = "http://127.0.0.1:6543"
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

	// Add chart generation callbacks
	reportData["gen_revenue_chart"] = generateRevenueChart
	reportData["gen_performance_chart"] = generatePerformanceChart

	// Create rendering job with JS event support for PagedJS
	job := report.NewReportJob(zpt)
	job.UseJSEvent(true)

	// Render using zipreport-server
	fmt.Println("Generating executive report...")
	client := api.NewZipReport(serverURL, apiKey)
	result := client.Render(job, reportData, nil)

	if !result.Success {
		fmt.Printf("Error generating report: %s\n", result.Error)
		os.Exit(1)
	}

	// Save PDF to file
	if err := os.WriteFile(pdfName, result.Report, 0644); err != nil {
		fmt.Printf("Failed to save PDF: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Report generated: %s\n", pdfName)
}
