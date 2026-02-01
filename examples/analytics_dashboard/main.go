// Analytics Dashboard example demonstrates a dashboard with ZipReport.
//
// This example shows how to:
//   - Single-page dashboard layout with PagedJS
//   - Multiple chart types (line, bar)
//   - Dynamic chart generation using gonum/plot
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

// generateTrafficChart creates a line chart (520x160)
func generateTrafficChart(data interface{}) []byte {
	args, _ := data.(map[string]interface{})
	days, _ := args["days"].([]interface{})
	sessions, _ := args["sessions"].([]interface{})
	pageviews, _ := args["pageviews"].([]interface{})

	if len(days) == 0 || len(sessions) == 0 {
		return nil
	}

	p := plot.New()
	p.Title.TextStyle.Font.Size = vg.Points(7)
	p.X.Tick.Label.Font.Size = vg.Points(5)
	p.Y.Tick.Label.Font.Size = vg.Points(5)
	p.Legend.TextStyle.Font.Size = vg.Points(5)
	p.Legend.Top = true
	p.Legend.Left = false

	n := len(sessions)
	pts1 := make(plotter.XYs, n)
	pts2 := make(plotter.XYs, n)
	for i := 0; i < n; i++ {
		pts1[i].X = float64(i)
		pts1[i].Y = toFloat(sessions[i])
		if i < len(pageviews) {
			pts2[i].X = float64(i)
			pts2[i].Y = toFloat(pageviews[i])
		}
	}

	line1, _ := plotter.NewLine(pts1)
	line1.Color = color.RGBA{R: 59, G: 130, B: 246, A: 255}
	line1.Width = vg.Points(1.5)

	line2, _ := plotter.NewLine(pts2)
	line2.Color = color.RGBA{R: 139, G: 92, B: 246, A: 255}
	line2.Width = vg.Points(1.5)
	line2.Dashes = []vg.Length{vg.Points(3), vg.Points(2)}

	p.Add(line1, line2)
	p.Legend.Add("Sessions", line1)
	p.Legend.Add("Pageviews", line2)

	// Set X axis labels
	var labels []string
	for _, d := range days {
		if l, ok := d.(string); ok {
			labels = append(labels, l)
		}
	}
	p.NominalX(labels...)

	w, err := p.WriterTo(vg.Points(420), vg.Points(130), "png")
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	w.WriteTo(&buf)
	return buf.Bytes()
}

// generateDeviceChart creates a simple bar chart for devices (180x130)
func generateDeviceChart(data interface{}) []byte {
	args, _ := data.(map[string]interface{})
	values, _ := args["values"].([]interface{})
	labels, _ := args["labels"].([]interface{})

	if len(values) == 0 {
		return nil
	}

	p := plot.New()
	p.Title.TextStyle.Font.Size = vg.Points(6)
	p.X.Tick.Label.Font.Size = vg.Points(5)
	p.Y.Tick.Label.Font.Size = vg.Points(5)
	p.Y.Label.Text = "%"
	p.Y.Label.TextStyle.Font.Size = vg.Points(5)

	vals := make(plotter.Values, len(values))
	for i, v := range values {
		vals[i] = toFloat(v)
	}

	bars, _ := plotter.NewBarChart(vals, vg.Points(20))
	bars.Color = color.RGBA{R: 59, G: 130, B: 246, A: 255}

	p.Add(bars)

	var labelStrs []string
	for _, l := range labels {
		if s, ok := l.(string); ok {
			labelStrs = append(labelStrs, s)
		}
	}
	p.NominalX(labelStrs...)

	w, err := p.WriterTo(vg.Points(145), vg.Points(105), "png")
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	w.WriteTo(&buf)
	return buf.Bytes()
}

// generateHourlyChart creates a bar chart (240x110)
func generateHourlyChart(data interface{}) []byte {
	args, _ := data.(map[string]interface{})
	hours, _ := args["hours"].([]interface{})
	traffic, _ := args["traffic"].([]interface{})

	if len(hours) == 0 || len(traffic) == 0 {
		return nil
	}

	p := plot.New()
	p.Title.TextStyle.Font.Size = vg.Points(6)
	p.X.Tick.Label.Font.Size = vg.Points(4)
	p.Y.Tick.Label.Font.Size = vg.Points(4)

	vals := make(plotter.Values, len(traffic))
	for i, v := range traffic {
		vals[i] = toFloat(v)
	}

	bars, _ := plotter.NewBarChart(vals, vg.Points(12))
	bars.Color = color.RGBA{R: 59, G: 130, B: 246, A: 255}

	p.Add(bars)

	var labelStrs []string
	for _, h := range hours {
		if s, ok := h.(string); ok {
			labelStrs = append(labelStrs, s)
		}
	}
	p.NominalX(labelStrs...)

	w, err := p.WriterTo(vg.Points(195), vg.Points(90), "png")
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	w.WriteTo(&buf)
	return buf.Bytes()
}

// generateSourcesChart creates a horizontal bar chart (240x110)
func generateSourcesChart(data interface{}) []byte {
	args, _ := data.(map[string]interface{})
	sources, _ := args["sources"].([]interface{})
	values, _ := args["values"].([]interface{})

	if len(sources) == 0 || len(values) == 0 {
		return nil
	}

	p := plot.New()
	p.Title.TextStyle.Font.Size = vg.Points(6)
	p.X.Tick.Label.Font.Size = vg.Points(4)
	p.Y.Tick.Label.Font.Size = vg.Points(4)
	p.X.Label.Text = "%"
	p.X.Label.TextStyle.Font.Size = vg.Points(4)

	// For horizontal bars, we create a regular bar chart and rotate
	vals := make(plotter.Values, len(values))
	for i, v := range values {
		vals[i] = toFloat(v)
	}

	bars, _ := plotter.NewBarChart(vals, vg.Points(10))
	bars.Color = color.RGBA{R: 16, G: 185, B: 129, A: 255}
	bars.Horizontal = true

	p.Add(bars)

	var labelStrs []string
	for _, s := range sources {
		if l, ok := s.(string); ok {
			labelStrs = append(labelStrs, l)
		}
	}
	p.NominalY(labelStrs...)

	w, err := p.WriterTo(vg.Points(195), vg.Points(90), "png")
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
	fmt.Println("Building dashboard template...")
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
	reportData["gen_traffic_chart"] = generateTrafficChart
	reportData["gen_device_chart"] = generateDeviceChart
	reportData["gen_hourly_chart"] = generateHourlyChart
	reportData["gen_sources_chart"] = generateSourcesChart

	// Render using zipreport-server processor with JS event enabled
	fmt.Println("Generating analytics dashboard...")
	client := api.NewZipReport(serverURL, apiKey)
	job := report.NewReportJob(zpt)
	job.UseJSEvent(true)
	result := client.Render(job, reportData, nil)

	if !result.Success {
		fmt.Printf("An error occurred while generating the dashboard: %s\n", result.Error)
		os.Exit(1)
	}

	// Save PDF to file
	if err := os.WriteFile(pdfName, result.Report, 0644); err != nil {
		fmt.Printf("Failed to save PDF: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Dashboard generated: %s\n", pdfName)
}
