package processor

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"net/mail"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zipreport/zipreport-go/pkg/fileutils"
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
	reportName := r.ReportFilename()
	htmlContent, err := r.GetBytes(reportName)
	if err != nil {
		return report.NewJobResultError("failed to get " + reportName + ": " + err.Error())
	}

	// Extract local resources from HTML
	resources := extractResources(string(htmlContent))

	// Build MIME message
	mimeData, err := p.buildMIMEMessage(r, reportName, htmlContent, resources)
	if err != nil {
		return report.NewJobResultError("failed to build MIME message: " + err.Error())
	}

	return report.NewJobResult(mimeData)
}

// buildMIMEMessage creates the MIME message with embedded resources.
func (p *MIMEProcessor) buildMIMEMessage(r *report.ReportFile, reportName string, htmlContent []byte, resources []string) ([]byte, error) {
	subject := strings.TrimSpace(p.subject)
	if subject == "" {
		// The manifest title is not caller input: collapse any line breaks instead of failing
		subject = strings.Join(strings.FieldsFunc(r.Title(), isHeaderSpace), " ")
	}
	from := strings.TrimSpace(p.from)
	to := make([]string, 0, len(p.to))
	for _, addr := range p.to {
		to = append(to, strings.TrimSpace(addr))
	}
	for _, v := range append([]string{subject, from}, to...) {
		if hasControlChars(v) {
			return nil, fmt.Errorf("header value contains a line break or control character: %q", v)
		}
	}
	encodedSubject := mime.QEncoding.Encode("utf-8", subject)
	if encodedSubject == subject && len("Subject: ")+len(subject) > maxHeaderLineLength {
		// ASCII subjects are not encoded, so they cannot be folded at encoded-word boundaries
		return nil, fmt.Errorf("subject too long: %d characters", len(subject))
	}

	var buf bytes.Buffer
	boundary := generateBoundary()

	// Write headers
	if from != "" {
		fmt.Fprintf(&buf, "From: %s\r\n", from)
	}
	if len(to) > 0 {
		fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(to, ",\r\n "))
	}
	fmt.Fprintf(&buf, "Subject: %s\r\n", foldEncodedWords(encodedSubject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/related; boundary=\"%s\"\r\n", boundary)
	buf.WriteString("\r\n")

	// Embed each referenced file once; leave references that are not files untouched
	type embedded struct {
		name    string
		cid     string
		content []byte
	}
	var parts []embedded
	cidByPath := make(map[string]string)
	cidByRef := make(map[string]string)
	baseDir := path.Dir(reportName)
	for _, ref := range resources {
		resolved := resolveResource(baseDir, ref)
		if fileutils.IsHiddenPath(resolved) {
			continue // never embed hidden files
		}
		if cid, ok := cidByPath[resolved]; ok {
			cidByRef[ref] = cid
			continue
		}
		content, err := r.GetBytes(resolved)
		if err != nil {
			continue
		}
		cid := fmt.Sprintf("resource%d@zipreport", len(parts))
		cidByPath[resolved] = cid
		cidByRef[ref] = cid
		parts = append(parts, embedded{name: resolved, cid: cid, content: content})
	}

	// Replace embedded references with cid: references, only inside src and <link href> values
	modifiedHTML := rewriteRefs(string(htmlContent), srcRegex, cidByRef)
	modifiedHTML = rewriteRefs(modifiedHTML, linkHrefRegex, cidByRef)

	// Write HTML part
	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	buf.WriteString("\r\n")
	writeBase64Lines(&buf, []byte(modifiedHTML))

	// Write resource parts
	for _, part := range parts {
		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		fmt.Fprintf(&buf, "Content-Type: %s\r\n", getMIMEType(part.name))
		buf.WriteString("Content-Transfer-Encoding: base64\r\n")
		fmt.Fprintf(&buf, "Content-ID: <%s>\r\n", part.cid)
		fmt.Fprintf(&buf, "Content-Disposition: %s\r\n", mime.FormatMediaType("inline", map[string]string{"filename": filepath.Base(part.name)}))
		buf.WriteString("\r\n")
		writeBase64Lines(&buf, part.content)
	}

	// Write closing boundary
	fmt.Fprintf(&buf, "--%s--\r\n", boundary)

	return buf.Bytes(), nil
}

// resolveResource resolves a reference from the HTML file in baseDir to a path in the report.
// A leading "/" refers to the report root.
func resolveResource(baseDir, ref string) string {
	if strings.HasPrefix(ref, "/") {
		return path.Clean(strings.TrimPrefix(ref, "/"))
	}
	return path.Join(baseDir, ref)
}

// rewriteRefs replaces the captured reference in each match of re with its cid: form.
// References without an embedded part are left unchanged.
func rewriteRefs(html string, re *regexp.Regexp, cidByRef map[string]string) string {
	return re.ReplaceAllStringFunc(html, func(m string) string {
		loc := re.FindStringSubmatchIndex(m)
		cid, ok := cidByRef[m[loc[2]:loc[3]]]
		if !ok {
			return m
		}
		return m[:loc[2]] + "cid:" + cid + m[loc[3]:]
	})
}

// maxHeaderLineLength is the maximum header line length allowed by RFC 5322.
const maxHeaderLineLength = 998

// isHeaderSpace reports whether c is a space, tab or line break.
func isHeaderSpace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

// hasControlChars reports whether s contains ASCII control characters other than tab (including CR and LF).
func hasControlChars(s string) bool {
	for _, c := range s {
		if (c < 0x20 && c != '\t') || c == 0x7f {
			return true
		}
	}
	return false
}

// foldEncodedWords puts each RFC 2047 encoded word on its own folded header line.
func foldEncodedWords(s string) string {
	return strings.ReplaceAll(s, "?= =?", "?=\r\n =?")
}

// base64LineLength is the maximum encoded line length allowed by RFC 2045.
const base64LineLength = 76

// writeBase64Lines writes data base64-encoded in CRLF-terminated lines of at most 76 characters.
func writeBase64Lines(buf *bytes.Buffer, data []byte) {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > base64LineLength {
		buf.WriteString(encoded[:base64LineLength])
		buf.WriteString("\r\n")
		encoded = encoded[base64LineLength:]
	}
	buf.WriteString(encoded)
	buf.WriteString("\r\n")
}

// srcRegex matches src attributes (images, scripts, media).
var srcRegex = regexp.MustCompile(`\bsrc=["']([^"']+)["']`)

// linkHrefRegex matches href attributes of <link> elements (stylesheets, icons). Anchor hrefs are links, not resources.
var linkHrefRegex = regexp.MustCompile(`<link\b[^>]*\bhref=["']([^"']+)["']`)

// schemeRegex matches references that start with a URL scheme (http:, mailto:, data:, cid:, ...).
var schemeRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// extractResources finds local resource references in HTML.
func extractResources(html string) []string {
	matches := srcRegex.FindAllStringSubmatch(html, -1)
	matches = append(matches, linkHrefRegex.FindAllStringSubmatch(html, -1)...)

	seen := make(map[string]bool)
	var resources []string

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		resource := match[1]

		// Skip external URLs and anything with a scheme (mailto:, tel:, data:, cid:, ...)
		if isExternalURL(resource) || schemeRegex.MatchString(resource) {
			continue
		}

		// Skip anchors
		if strings.HasPrefix(resource, "#") {
			continue
		}

		// Never embed HTML pages (e.g. <link rel="canonical" href="report.html">)
		if ext := strings.ToLower(path.Ext(resource)); ext == ".html" || ext == ".htm" {
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
