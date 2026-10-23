package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
var debugHost string

func init() {
	debugCmd.Flags().StringVar(&debugHost, "host", "127.0.0.1", "Interface to listen on (use 0.0.0.0 for all interfaces)")
	debugCmd.Flags().IntVarP(&debugPort, "port", "p", 8000, "Port to listen on")
}

func runDebug(cmd *cobra.Command, args []string) error {
	source := args[0]

	// Verify source exists
	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("source not found: %w", err)
	}

	// Start server
	srv := newDebugServer(debugHost, debugPort, source)
	displayHost := debugHost
	if displayHost == "" || displayHost == "0.0.0.0" || displayHost == "::" {
		displayHost = "localhost"
	}
	fmt.Printf("Starting debug server at http://%s\n", net.JoinHostPort(displayHost, strconv.Itoa(debugPort)))
	fmt.Printf("Serving template from: %s\n", source)
	fmt.Println("Press Ctrl+C to stop")

	return srv.ListenAndServe()
}

// newDebugServer creates the debug HTTP server for a template source.
// When bound to a loopback address it only answers requests addressed to
// localhost, which blocks DNS-rebinding access from web pages.
func newDebugServer(host string, port int, source string) *http.Server {
	host = strings.Trim(host, "[]") // accept --host [::1]
	var handler http.Handler = &debugHandler{source: source}
	if isLoopbackHost(host) {
		handler = localHostOnly(handler)
	}
	return &http.Server{
		Addr:              net.JoinHostPort(host, strconv.Itoa(port)),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

// isLoopbackHost reports whether host is localhost or a loopback IP.
func isLoopbackHost(host string) bool {
	if strings.EqualFold(strings.TrimSuffix(host, "."), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// localHostOnly rejects requests whose Host header is not a loopback name.
func localHostOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		// An empty Host (HTTP/1.0 clients) cannot come from a rebinding web page
		if host != "" && !isLoopbackHost(strings.Trim(host, "[]")) {
			http.Error(w, "Forbidden host (start with --host 0.0.0.0 to allow other hostnames)", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
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
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name != "" && name != "index.html" {
		// Try to serve as static file
		content, err := zpt.GetBytes(name)
		if err == nil {
			// Determine content type
			ext := filepath.Ext(name)
			contentType := getContentType(ext)
			w.Header().Set("Content-Type", contentType)
			_, _ = w.Write(content)
			return
		}
	}

	// Default data comes from data.json, merged by the renderer
	data := make(map[string]interface{})

	// Render into an in-memory copy so nothing is written to the template directory
	renderer := template.NewMiyaRender(zpt.Clone(), template.WithStrictParams(false))
	output, err := renderer.Render(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		log.Printf("Render error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(output))
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
