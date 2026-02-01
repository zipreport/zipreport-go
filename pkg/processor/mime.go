package processor

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"net/mail"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zipreport/zipreport-go/pkg/report"
)

// MIMEProcessor generates MIME email messages from reports.
type MIMEProcessor struct {
	subject string
	from    string
	to      []string
}

// MIMEOption configures a MIMEProcessor.
type MIMEOption func(*MIMEProcessor)

// WithSubject sets the email subject.
func WithSubject(subject string) MIMEOption {
	return func(p *MIMEProcessor) {
		p.subject = subject
	}
}

// WithFrom sets the sender address.
func WithFrom(from string) MIMEOption {
	return func(p *MIMEProcessor) {
		p.from = from
	}
}

// WithTo sets the recipient addresses.
func WithTo(to ...string) MIMEOption {
	return func(p *MIMEProcessor) {
		p.to = to
	}
}

// NewMIMEProcessor creates a new MIMEProcessor.
func NewMIMEProcessor(opts ...MIMEOption) *MIMEProcessor {
	p := &MIMEProcessor{}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Process generates a MIME message from the report.
func (p *MIMEProcessor) Process(job *report.ReportJob) *report.JobResult {
	r := job.GetReport()

	// Get rendered HTML
	htmlContent, err := r.GetBytes(report.FileReport)
	if err != nil {
		return report.NewJobResultError("failed to get report.html: " + err.Error())
	}

	// Extract local resources from HTML
	resources := extractResources(string(htmlContent))

	// Build MIME message
	mimeData, err := p.buildMIMEMessage(r, htmlContent, resources)
	if err != nil {
		return report.NewJobResultError("failed to build MIME message: " + err.Error())
	}

	return report.NewJobResult(mimeData)
}

// buildMIMEMessage creates the MIME message with embedded resources.
func (p *MIMEProcessor) buildMIMEMessage(r *report.ReportFile, htmlContent []byte, resources []string) ([]byte, error) {
	var buf bytes.Buffer
	boundary := generateBoundary()

	// Write headers
	if p.from != "" {
		buf.WriteString(fmt.Sprintf("From: %s\r\n", p.from))
	}
	if len(p.to) > 0 {
		buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(p.to, ", ")))
	}
	if p.subject != "" {
		buf.WriteString(fmt.Sprintf("Subject: %s\r\n", p.subject))
	} else {
		buf.WriteString(fmt.Sprintf("Subject: %s\r\n", r.Title()))
	}
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/related; boundary=\"%s\"\r\n", boundary))
	buf.WriteString("\r\n")

	// Write HTML part
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	buf.WriteString("\r\n")

	// Replace local resource references with cid: references
	modifiedHTML := string(htmlContent)
	cidMap := make(map[string]string)
	for i, resource := range resources {
		cid := fmt.Sprintf("resource%d@zipreport", i)
		cidMap[resource] = cid
		modifiedHTML = strings.ReplaceAll(modifiedHTML, fmt.Sprintf(`"%s"`, resource), fmt.Sprintf(`"cid:%s"`, cid))
		modifiedHTML = strings.ReplaceAll(modifiedHTML, fmt.Sprintf(`'%s'`, resource), fmt.Sprintf(`'cid:%s'`, cid))
	}

	buf.WriteString(base64.StdEncoding.EncodeToString([]byte(modifiedHTML)))
	buf.WriteString("\r\n")

	// Write resource parts
	for _, resource := range resources {
		content, err := r.GetBytes(resource)
		if err != nil {
			continue // Skip missing resources
		}

		cid := cidMap[resource]
		mimeType := getMIMEType(resource)

		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString(fmt.Sprintf("Content-Type: %s\r\n", mimeType))
		buf.WriteString("Content-Transfer-Encoding: base64\r\n")
		buf.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", cid))
		buf.WriteString(fmt.Sprintf("Content-Disposition: inline; filename=\"%s\"\r\n", filepath.Base(resource)))
		buf.WriteString("\r\n")
		buf.WriteString(base64.StdEncoding.EncodeToString(content))
		buf.WriteString("\r\n")
	}

	// Write closing boundary
	buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return buf.Bytes(), nil
}

// extractResources finds local resource references in HTML.
func extractResources(html string) []string {
	// Match src and href attributes
	srcRegex := regexp.MustCompile(`(?:src|href)=["']([^"']+)["']`)
	matches := srcRegex.FindAllStringSubmatch(html, -1)

	seen := make(map[string]bool)
	var resources []string

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		resource := match[1]

		// Skip external URLs
		if isExternalURL(resource) {
			continue
		}

		// Skip data URIs
		if strings.HasPrefix(resource, "data:") {
			continue
		}

		// Skip cid references
		if strings.HasPrefix(resource, "cid:") {
			continue
		}

		// Skip anchors
		if strings.HasPrefix(resource, "#") {
			continue
		}

		// Deduplicate
		if seen[resource] {
			continue
		}
		seen[resource] = true

		resources = append(resources, resource)
	}

	return resources
}

// isExternalURL checks if a URL is external.
func isExternalURL(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "//")
}

// getMIMEType returns the MIME type for a file.
func getMIMEType(filename string) string {
	ext := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}

// generateBoundary creates a MIME boundary string.
func generateBoundary() string {
	return "----=_Part_0_ZipReport"
}

// ParseMIMEMessage parses a MIME message and validates it.
func ParseMIMEMessage(data []byte) (*mail.Message, error) {
	return mail.ReadMessage(bytes.NewReader(data))
}
