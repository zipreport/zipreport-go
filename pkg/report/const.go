package report

// File name constants
const (
	FileManifest = "manifest.json"
	FileIndex    = "index.html"
	FileReport   = "report.html"
	FileData     = "data.json"
)

// PageSize represents supported page sizes for PDF generation.
type PageSize string

const (
	PageA3      PageSize = "A3"
	PageA4      PageSize = "A4"
	PageA5      PageSize = "A5"
	PageLegal   PageSize = "Legal"
	PageLetter  PageSize = "Letter"
	PageTabloid PageSize = "Tabloid"
)

// MarginStyle represents predefined margin styles.
type MarginStyle string

const (
	MarginStandard MarginStyle = "standard"
	MarginNone     MarginStyle = "none"
	MarginMinimum  MarginStyle = "minimum"
)

// Required manifest fields
var ManifestRequiredFields = []string{
	"title",
	"author",
	"version",
	"description",
	"params",
}

// Default job settings
const (
	DefaultPageSize     = PageA4
	DefaultMarginStyle  = MarginStandard
	DefaultSettlingTime = 200 // milliseconds
	DefaultJSTimeout    = 60  // seconds
	DefaultJobTimeout   = 120 // seconds
)
