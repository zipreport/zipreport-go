// Package main demonstrates dynamic image generation with ZipReport filters.
//
// This example shows how to:
//   - Use image filters (png, gif, jpg, svg)
//   - Generate images dynamically at render time
//   - Pass callable functions to templates
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"

	"github.com/zipreport/zipreport-go/pkg/api"
	"github.com/zipreport/zipreport-go/pkg/report"
)

// colorMap maps color names to RGBA values
var colorMap = map[string]color.RGBA{
	"red":    {255, 0, 0, 255},
	"green":  {0, 255, 0, 255},
	"blue":   {0, 0, 255, 255},
	"yellow": {255, 255, 0, 255},
	"pink":   {255, 192, 203, 255},
	"cyan":   {0, 255, 255, 255},
	"white":  {255, 255, 255, 255},
	"black":  {0, 0, 0, 255},
}

// parseColor converts a color name to color.RGBA
func parseColor(name string) color.RGBA {
	if c, ok := colorMap[name]; ok {
		return c
	}
	return colorMap["white"]
}

// renderImage creates a solid color image
func renderImage(colorName string) *image.RGBA {
	c := parseColor(colorName)
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// renderPNG generates a PNG image with the specified color
func renderPNG(data interface{}) []byte {
	colorName, _ := data.(string)
	img := renderImage(colorName)

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// renderGIF generates a GIF image with the specified color
func renderGIF(data interface{}) []byte {
	colorName, _ := data.(string)
	img := renderImage(colorName)

	var buf bytes.Buffer
	gif.Encode(&buf, img, nil)
	return buf.Bytes()
}

// renderJPG generates a JPEG image with the specified color
func renderJPG(data interface{}) []byte {
	colorName, _ := data.(string)
	img := renderImage(colorName)

	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes()
}

// renderSVG generates an SVG with the specified color
func renderSVG(data interface{}) []byte {
	colorName, _ := data.(string)
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
  <rect x="0" y="0" width="100" height="100" fill="%s"/>
</svg>`, colorName)
	return []byte(svg)
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
	reportPath := "../zipreport/examples/reports/filter_example"
	zfs, result := report.BuildZipFs(reportPath, nil)
	if !result.Success() {
		fmt.Printf("Error building report: %s\n", result.Error())
		os.Exit(1)
	}

	// Load ReportFile from in-memory zipfs
	zpt, err := report.LoadZipFs(zfs)
	if err != nil {
		fmt.Printf("Error loading report: %v\n", err)
		os.Exit(1)
	}

	// Template variables with image generators
	reportData := map[string]interface{}{
		"png_graphic":      renderPNG,
		"png_graphic_data": "pink",
		"gif_graphic":      renderGIF,
		"gif_graphic_data": "blue",
		"jpg_graphic":      renderJPG,
		"jpg_graphic_data": "green",
		"svg_graphic":      renderSVG,
		"svg_graphic_data": "yellow",
	}

	// Render using zipreport-server processor
	client := api.NewZipReport(serverURL, apiKey)
	renderResult := client.RenderDefaults(zpt, reportData, nil)

	if !renderResult.Success {
		fmt.Printf("An error occurred while generating the PDF: %s\n", renderResult.Error)
		os.Exit(1)
	}

	// Save PDF to file
	if err := os.WriteFile(pdfName, renderResult.Report, 0644); err != nil {
		fmt.Printf("Failed to save PDF: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Report generated to %s\n", pdfName)
}
