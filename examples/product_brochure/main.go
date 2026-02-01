// Product Brochure example demonstrates a brochure with ZipReport.
//
// This example shows how to:
//   - Software product brochure template with PagedJS
//   - Feature comparison charts using gonum/plot
//   - Pricing tiers and testimonials
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

// generateComparisonChart creates a grouped bar chart (700x400)
func generateComparisonChart(data interface{}) []byte {
	args, _ := data.(map[string]interface{})
	categories, _ := args["categories"].([]interface{})
	cloudflow, _ := args["cloudflow"].([]interface{})
	competitorA, _ := args["competitor_a"].([]interface{})
	competitorB, _ := args["competitor_b"].([]interface{})

	if len(categories) == 0 {
		return nil
	}

	p := plot.New()
	p.Title.Text = "Competitive Comparison"
	p.Title.TextStyle.Font.Size = vg.Points(12)
	p.X.Tick.Label.Font.Size = vg.Points(8)
	p.Y.Tick.Label.Font.Size = vg.Points(8)
	p.Y.Label.Text = "Score (%)"
	p.Y.Label.TextStyle.Font.Size = vg.Points(9)
	p.Legend.TextStyle.Font.Size = vg.Points(8)
	p.Legend.Top = true

	n := len(categories)
	w := vg.Points(15)

	// Create data for each series
	cf := make(plotter.Values, n)
	ca := make(plotter.Values, n)
	cb := make(plotter.Values, n)

	for i := 0; i < n; i++ {
		if i < len(cloudflow) {
			cf[i] = toFloat(cloudflow[i])
		}
		if i < len(competitorA) {
			ca[i] = toFloat(competitorA[i])
		}
		if i < len(competitorB) {
			cb[i] = toFloat(competitorB[i])
		}
	}

	bars1, _ := plotter.NewBarChart(cf, w)
	bars1.Color = color.RGBA{R: 99, G: 102, B: 241, A: 255}
	bars1.Offset = -w

	bars2, _ := plotter.NewBarChart(ca, w)
	bars2.Color = color.RGBA{R: 148, G: 163, B: 184, A: 255}

	bars3, _ := plotter.NewBarChart(cb, w)
	bars3.Color = color.RGBA{R: 203, G: 213, B: 225, A: 255}
	bars3.Offset = w

	p.Add(bars1, bars2, bars3)
	p.Legend.Add("CloudFlow", bars1)
	p.Legend.Add("Competitor A", bars2)
	p.Legend.Add("Competitor B", bars3)

	var labelStrs []string
	for _, c := range categories {
		if s, ok := c.(string); ok {
			labelStrs = append(labelStrs, s)
		}
	}
	p.NominalX(labelStrs...)

	w2, err := p.WriterTo(vg.Points(560), vg.Points(320), "png")
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	w2.WriteTo(&buf)
	return buf.Bytes()
}

// generateFeatureChart creates a horizontal bar chart (650x300)
func generateFeatureChart(data interface{}) []byte {
	args, _ := data.(map[string]interface{})
	metrics, _ := args["metrics"].([]interface{})
	scores, _ := args["scores"].([]interface{})

	if len(metrics) == 0 || len(scores) == 0 {
		return nil
	}

	p := plot.New()
	p.Title.Text = "Feature Scores"
	p.Title.TextStyle.Font.Size = vg.Points(11)
	p.X.Tick.Label.Font.Size = vg.Points(8)
	p.Y.Tick.Label.Font.Size = vg.Points(8)
	p.X.Label.Text = "Score (%)"
	p.X.Label.TextStyle.Font.Size = vg.Points(9)

	vals := make(plotter.Values, len(scores))

	for i, v := range scores {
		vals[i] = toFloat(v)
	}

	bars, _ := plotter.NewBarChart(vals, vg.Points(25))
	bars.Horizontal = true
	bars.Color = color.RGBA{R: 99, G: 102, B: 241, A: 255}

	p.Add(bars)

	var labelStrs []string
	for _, m := range metrics {
		if s, ok := m.(string); ok {
			labelStrs = append(labelStrs, s)
		}
	}
	p.NominalY(labelStrs...)

	w, err := p.WriterTo(vg.Points(520), vg.Points(240), "png")
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	w.WriteTo(&buf)
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
	fmt.Println("Building brochure template...")
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

	// Add chart generator callbacks
	reportData["gen_comparison_chart"] = generateComparisonChart
	reportData["gen_feature_chart"] = generateFeatureChart

	// Render using zipreport-server processor with JS event enabled
	fmt.Println("Generating product brochure...")
	client := api.NewZipReport(serverURL, apiKey)
	job := report.NewReportJob(zpt)
	job.UseJSEvent(true)
	result := client.Render(job, reportData, nil)

	if !result.Success {
		fmt.Printf("An error occurred while generating the brochure: %s\n", result.Error)
		os.Exit(1)
	}

	// Save PDF to file
	if err := os.WriteFile(pdfName, result.Report, 0644); err != nil {
		fmt.Printf("Failed to save PDF: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Brochure generated: %s\n", pdfName)
}
