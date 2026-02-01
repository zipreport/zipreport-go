// Package main demonstrates PagedJS pagination with ZipReport.
//
// This example shows how to:
//   - Use PagedJS for advanced pagination
//   - Generate charts dynamically using gonum/plot
//   - Use JS event waiting for PagedJS rendering
package main

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"os"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"

	"github.com/zipreport/zipreport-go/pkg/api"
	"github.com/zipreport/zipreport-go/pkg/report"
)

// generateGraphics1 creates a dual-line signal chart
func generateGraphics1(data interface{}) []byte {
	p := plot.New()
	p.Title.Text = "Signal Coherence"
	p.Title.TextStyle.Font.Size = vg.Points(8)
	p.X.Label.Text = "Time"
	p.X.Label.TextStyle.Font.Size = vg.Points(6)
	p.Y.Label.Text = "Amplitude"
	p.Y.Label.TextStyle.Font.Size = vg.Points(6)
	p.X.Tick.Label.Font.Size = vg.Points(5)
	p.Y.Tick.Label.Font.Size = vg.Points(5)
	p.Legend.TextStyle.Font.Size = vg.Points(6)
	p.Legend.Top = true

	// Generate signal data
	n := 100
	pts1 := make(plotter.XYs, n)
	pts2 := make(plotter.XYs, n)
	for i := 0; i < n; i++ {
		t := float64(i) * 0.02
		pts1[i].X = t
		pts1[i].Y = math.Sin(2*math.Pi*5*t) + 0.3*math.Sin(17*t)
		pts2[i].X = t
		pts2[i].Y = math.Sin(2*math.Pi*5*t) + 0.3*math.Cos(13*t)
	}

	line1, _ := plotter.NewLine(pts1)
	line1.Color = color.RGBA{R: 31, G: 119, B: 180, A: 255}
	line1.Width = vg.Points(1)

	line2, _ := plotter.NewLine(pts2)
	line2.Color = color.RGBA{R: 255, G: 127, B: 14, A: 255}
	line2.Width = vg.Points(1)

	p.Add(line1, line2)
	p.Legend.Add("Signal 1", line1)
	p.Legend.Add("Signal 2", line2)

	w, err := p.WriterTo(vg.Points(200), vg.Points(200), "png")
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	w.WriteTo(&buf)
	return buf.Bytes()
}

// generateGraphics2 creates a bar chart representing 3D surface projection
func generateGraphics2(data interface{}) []byte {
	p := plot.New()
	p.Title.Text = "3D Surface Projection"
	p.Title.TextStyle.Font.Size = vg.Points(8)
	p.X.Label.Text = "X"
	p.X.Label.TextStyle.Font.Size = vg.Points(6)
	p.Y.Label.Text = "Z (avg)"
	p.Y.Label.TextStyle.Font.Size = vg.Points(6)
	p.X.Tick.Label.Font.Size = vg.Points(5)
	p.Y.Tick.Label.Font.Size = vg.Points(5)

	// Calculate surface values
	gridSize := 10
	values := make(plotter.Values, gridSize)
	for i := 0; i < gridSize; i++ {
		x := float64(i-gridSize/2) * 1.0
		sum := 0.0
		for j := 0; j < gridSize; j++ {
			y := float64(j-gridSize/2) * 1.0
			r := math.Sqrt(x*x + y*y)
			z := math.Sin(r)
			sum += z
		}
		values[i] = (sum/float64(gridSize) + 1) * 50
	}

	bars, _ := plotter.NewBarChart(values, vg.Points(15))
	bars.Color = color.RGBA{R: 44, G: 160, B: 44, A: 255}

	p.Add(bars)
	p.NominalX("-5", "-4", "-3", "-2", "-1", "0", "1", "2", "3", "4")

	w, err := p.WriterTo(vg.Points(200), vg.Points(200), "png")
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	w.WriteTo(&buf)
	return buf.Bytes()
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

	// Build report from template directory (relative to project root)
	reportPath := "../zipreport/examples/reports/pagedjs/example_report"

	fmt.Println("Building PagedJS report template...")
	zfs, buildResult := report.BuildZipFs(reportPath, nil)
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

	// Template data with chart generators
	reportData := map[string]interface{}{
		"date":   time.Now().Format("2006-01-02"),
		"author": "ZipReport library",
		"graphics_1": map[string]interface{}{
			"x-axis":    "time",
			"g1-y-axis": "amplitude",
		},
		"graphics_2": map[string]interface{}{
			"title": "3D surface",
		},
		"gen_graphics_1": generateGraphics1,
		"gen_graphics_2": generateGraphics2,
	}

	// Render using zipreport-server processor with JS event enabled
	fmt.Println("Generating report with dynamic charts...")
	client := api.NewZipReport(serverURL, apiKey)
	job := report.NewReportJob(zpt)
	job.UseJSEvent(true)
	result := client.Render(job, reportData, nil)

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
