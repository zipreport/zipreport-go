package template

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/zipreport/miya"
	"github.com/zipreport/miya/runtime"
	"github.com/zipreport/zipreport-go/pkg/report"
)

// uidCounter is used to generate unique IDs for generated files.
var uidCounter uint64

// generateUID generates a unique ID.
func generateUID() uint64 {
	return atomic.AddUint64(&uidCounter, 1)
}

// ImageFilterArgs contains arguments for image filters.
type ImageFilterArgs struct {
	Data   interface{}
	Alt    string
	Width  int
	Height int
	Class  string
}

// registerDefaultFilters registers the default filters on the environment.
func registerDefaultFilters(env *miya.Environment, r *report.ReportFile) error {
	// PNG filter
	if err := env.AddFilter("png", createImageFilter(r, "png", "image/png")); err != nil {
		return fmt.Errorf("failed to register png filter: %w", err)
	}

	// GIF filter
	if err := env.AddFilter("gif", createImageFilter(r, "gif", "image/gif")); err != nil {
		return fmt.Errorf("failed to register gif filter: %w", err)
	}

	// JPG filter
	if err := env.AddFilter("jpg", createImageFilter(r, "jpg", "image/jpeg")); err != nil {
		return fmt.Errorf("failed to register jpg filter: %w", err)
	}

	// JPEG filter (alias)
	if err := env.AddFilter("jpeg", createImageFilter(r, "jpg", "image/jpeg")); err != nil {
		return fmt.Errorf("failed to register jpeg filter: %w", err)
	}

	// SVG filter
	if err := env.AddFilter("svg", createImageFilter(r, "svg", "image/svg+xml")); err != nil {
		return fmt.Errorf("failed to register svg filter: %w", err)
	}

	// JSON filter
	if err := env.AddFilter("json", jsonFilter); err != nil {
		return fmt.Errorf("failed to register json filter: %w", err)
	}

	return nil
}

// createImageFilter creates an image filter for the given format.
// The returned function matches miya's FilterFunc signature.
func createImageFilter(r *report.ReportFile, ext, mimeType string) miya.FilterFunc {
	return func(callable interface{}, args ...interface{}) (interface{}, error) {
		// Parse filter arguments
		filterArgs := parseImageArgs(args)

		// Call the generator function
		var imageData []byte
		var genErr error

		switch fn := callable.(type) {
		case func(interface{}) []byte:
			imageData = fn(filterArgs.Data)
		case func(interface{}) ([]byte, error):
			imageData, genErr = fn(filterArgs.Data)
			if genErr != nil {
				return nil, fmt.Errorf("error generating image: %w", genErr)
			}
		case func() []byte:
			imageData = fn()
		case func() ([]byte, error):
			imageData, genErr = fn()
			if genErr != nil {
				return nil, fmt.Errorf("error generating image: %w", genErr)
			}
		case []byte:
			imageData = fn
		default:
			return nil, fmt.Errorf("invalid image generator type: %T", callable)
		}

		if len(imageData) == 0 {
			return nil, fmt.Errorf("empty image data")
		}

		// Generate unique filename and add to report
		filename := fmt.Sprintf("_gen_%d.%s", generateUID(), ext)
		if err := r.Add(filename, imageData, true); err != nil {
			return nil, fmt.Errorf("error adding image: %w", err)
		}

		// Build img tag and return as safe HTML (won't be escaped)
		return runtime.SafeValue{Value: buildImgTag(filename, filterArgs)}, nil
	}
}

// parseImageArgs parses arguments passed to image filters.
func parseImageArgs(args []interface{}) ImageFilterArgs {
	result := ImageFilterArgs{}

	for i, arg := range args {
		switch i {
		case 0:
			result.Data = arg
		case 1:
			if s, ok := arg.(string); ok {
				result.Alt = s
			}
		case 2:
			if n, ok := toInt(arg); ok {
				result.Width = n
			}
		case 3:
			if n, ok := toInt(arg); ok {
				result.Height = n
			}
		case 4:
			if s, ok := arg.(string); ok {
				result.Class = s
			}
		}
	}

	// Handle named arguments if passed as a map
	if len(args) == 1 {
		if m, ok := args[0].(map[string]interface{}); ok {
			if v, exists := m["data"]; exists {
				result.Data = v
			}
			if v, exists := m["alt"]; exists {
				if s, ok := v.(string); ok {
					result.Alt = s
				}
			}
			if v, exists := m["width"]; exists {
				if n, ok := toInt(v); ok {
					result.Width = n
				}
			}
			if v, exists := m["height"]; exists {
				if n, ok := toInt(v); ok {
					result.Height = n
				}
			}
			if v, exists := m["class"]; exists {
				if s, ok := v.(string); ok {
					result.Class = s
				}
			}
		}
	}

	return result
}

// buildImgTag builds an HTML img tag.
func buildImgTag(src string, args ImageFilterArgs) string {
	tag := fmt.Sprintf(`<img src="%s"`, src)

	if args.Alt != "" {
		tag += fmt.Sprintf(` alt="%s"`, escapeHTML(args.Alt))
	}
	if args.Width > 0 {
		tag += fmt.Sprintf(` width="%d"`, args.Width)
	}
	if args.Height > 0 {
		tag += fmt.Sprintf(` height="%d"`, args.Height)
	}
	if args.Class != "" {
		tag += fmt.Sprintf(` class="%s"`, escapeHTML(args.Class))
	}

	tag += " />"
	return tag
}

// jsonFilter converts a value to JSON.
// Matches miya's FilterFunc signature.
func jsonFilter(value interface{}, args ...interface{}) (interface{}, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "null", nil
	}
	return string(data), nil
}

// toInt converts a value to int.
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	case string:
		var i int
		_, err := fmt.Sscanf(n, "%d", &i)
		if err == nil {
			return i, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// escapeHTML escapes special HTML characters.
func escapeHTML(s string) string {
	replacer := map[rune]string{
		'&':  "&amp;",
		'<':  "&lt;",
		'>':  "&gt;",
		'"':  "&quot;",
		'\'': "&#39;",
	}

	result := make([]rune, 0, len(s))
	for _, r := range s {
		if replacement, ok := replacer[r]; ok {
			result = append(result, []rune(replacement)...)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
