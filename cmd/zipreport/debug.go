package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zipreport/zipreport-go/pkg/report"
	"github.com/zipreport/zipreport-go/pkg/template"
)

var debugCmd = &cobra.Command{
	Use:   "debug <source>",
	Short: "Start debug HTTP server for template development",
	Long: `Debug starts an HTTP server that renders the template on each request.

This is useful for template development as changes are reflected immediately.

The source can be a .zpt file or a template directory.

Example:
  zipreport debug ./my-template
  zipreport debug report.zpt
  zipreport debug -p 8080 ./my-template`,
	Args: cobra.ExactArgs(1),
	RunE: runDebug,
}

var debugPort int

func init() {
	debugCmd.Flags().IntVarP(&debugPort, "port", "p", 8000, "Port to listen on")
}

func runDebug(cmd *cobra.Command, args []string) error {
	source := args[0]

	// Verify source exists
	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("source not found: %w", err)
	}

	// Create handler
	handler := &debugHandler{source: source}

	// Start server
	addr := fmt.Sprintf(":%d", debugPort)
	fmt.Printf("Starting debug server at http://localhost%s\n", addr)
	fmt.Printf("Serving template from: %s\n", source)
	fmt.Println("Press Ctrl+C to stop")

	return http.ListenAndServe(addr, handler)
}

type debugHandler struct {
	source string
}

func (h *debugHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Load report fresh on each request
	zpt, err := report.Load(h.source)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load report: %v", err), http.StatusInternalServerError)
		return
	}

	// Handle static assets
	path := r.URL.Path
	if path != "/" && path != "/index.html" {
		// Try to serve as static file
		content, err := zpt.GetBytes(path[1:]) // Remove leading /
		if err == nil {
			// Determine content type
			ext := filepath.Ext(path)
			contentType := getContentType(ext)
			w.Header().Set("Content-Type", contentType)
			w.Write(content)
			return
		}
	}

	// Load data from data.json if exists
	var data map[string]interface{}
	if zpt.Exists(report.FileData) {
		dataBytes, err := zpt.GetBytes(report.FileData)
		if err == nil {
			json.Unmarshal(dataBytes, &data)
		}
	}
	if data == nil {
		data = make(map[string]interface{})
	}

	// Render template
	renderer := template.NewMiyaRender(zpt, template.WithStrictParams(false))
	output, err := renderer.Render(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		log.Printf("Render error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(output))
}

func getContentType(ext string) string {
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".otf":
		return "font/otf"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
