// Package template provides Jinja2-compatible template rendering using miya.
package template

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zipreport/miya"
	"github.com/zipreport/zipreport-go/pkg/report"
)

// RenderOption configures the MiyaRender.
type RenderOption func(*MiyaRender)

// RenderOptions contains rendering configuration.
type RenderOptions struct {
	StrictParams bool
	AutoEscape   bool
	TrimBlocks   bool
	LstripBlocks bool
}

// DefaultRenderOptions returns the default rendering options.
func DefaultRenderOptions() *RenderOptions {
	return &RenderOptions{
		StrictParams: true,
		AutoEscape:   true,
		TrimBlocks:   true,
		LstripBlocks: true,
	}
}

// WithStrictParams enables or disables strict parameter validation.
func WithStrictParams(strict bool) RenderOption {
	return func(r *MiyaRender) {
		r.options.StrictParams = strict
	}
}

// WithAutoEscape enables or disables auto-escaping.
func WithAutoEscape(escape bool) RenderOption {
	return func(r *MiyaRender) {
		r.options.AutoEscape = escape
	}
}

// WithWrapper sets an EnvironmentWrapper.
func WithWrapper(wrapper EnvironmentWrapper) RenderOption {
	return func(r *MiyaRender) {
		r.wrapper = wrapper
	}
}

// MiyaRender handles template rendering using the miya engine.
type MiyaRender struct {
	report      *report.ReportFile
	env         *miya.Environment
	options     *RenderOptions
	wrapper     EnvironmentWrapper
	filterError error
}

// NewMiyaRender creates a new MiyaRender for a report.
// Rendering writes the output (and any generated images) into r; render r.Clone() to keep r unchanged.
func NewMiyaRender(r *report.ReportFile, opts ...RenderOption) *MiyaRender {
	options := DefaultRenderOptions()

	mr := &MiyaRender{
		report:  r,
		options: options,
	}

	// Apply options
	for _, opt := range opts {
		opt(mr)
	}

	// Create environment with configured options
	mr.env = miya.NewEnvironment(
		miya.WithAutoEscape(mr.options.AutoEscape),
		miya.WithTrimBlocks(mr.options.TrimBlocks),
		miya.WithLstripBlocks(mr.options.LstripBlocks),
	)

	// Set custom loader
	mr.env.SetLoader(NewReportFileLoader(r))

	// Register default filters
	if err := registerDefaultFilters(mr.env, r); err != nil {
		// Log the error but continue - filters may not be needed
		mr.filterError = err
	}

	return mr
}

// SetWrapper sets the EnvironmentWrapper.
func (r *MiyaRender) SetWrapper(wrapper EnvironmentWrapper) {
	r.wrapper = wrapper
}

// GetEnvironment returns the miya Environment.
func (r *MiyaRender) GetEnvironment() *miya.Environment {
	return r.env
}

// Render renders the template with the given data.
func (r *MiyaRender) Render(data map[string]interface{}) (string, error) {
	return r.RenderTemplate(report.FileIndex, data)
}

// RenderTemplate renders a specific template with the given data.
func (r *MiyaRender) RenderTemplate(templateName string, data map[string]interface{}) (string, error) {
	// Work on a copy so the caller's map is never modified
	merged := make(map[string]interface{}, len(data))
	for k, v := range data {
		merged[k] = v
	}
	data = merged

	// Try to load default data from data.json
	if r.report.Exists(report.FileData) {
		defaultData, err := r.loadDefaultData()
		if err == nil {
			// Merge default data with provided data (provided takes precedence)
			for k, v := range defaultData {
				if _, exists := data[k]; !exists {
					data[k] = v
				}
			}
		}
	}

	// Validate parameters if strict mode is enabled
	if r.options.StrictParams {
		if err := r.validateParams(data); err != nil {
			return "", err
		}
	}

	// Apply wrapper if present
	if r.wrapper != nil {
		r.wrapper.Configure(r.env)
	}

	// Load and compile template
	tmpl, err := r.env.GetTemplate(templateName)
	if err != nil {
		return "", fmt.Errorf("failed to load template %s: %w", templateName, err)
	}
	defer tmpl.Release()

	// Create context from data
	ctx := miya.NewContextFrom(data)
	defer miya.PutContext(ctx)

	// Render template
	output, err := tmpl.Render(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	// Write output to report
	outputFile := r.report.ReportFilename()
	if err := r.report.Add(outputFile, []byte(output), true); err != nil {
		return "", fmt.Errorf("failed to write output: %w", err)
	}

	return output, nil
}

// loadDefaultData loads default data from data.json.
func (r *MiyaRender) loadDefaultData() (map[string]interface{}, error) {
	content, err := r.report.GetBytes(report.FileData)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	return data, nil
}

// validateParams validates that all required parameters are provided.
func (r *MiyaRender) validateParams(data map[string]interface{}) error {
	requiredParams := r.report.Params()
	if len(requiredParams) == 0 {
		return nil
	}

	var missing []string
	for _, param := range requiredParams {
		if _, exists := data[param]; !exists {
			missing = append(missing, param)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required parameters: %v", missing)
	}

	return nil
}

// RenderString renders a template from a string.
func (r *MiyaRender) RenderString(source string, data map[string]interface{}) (string, error) {
	// Apply wrapper if present
	if r.wrapper != nil {
		r.wrapper.Configure(r.env)
	}

	// Create context from data
	ctx := miya.NewContextFrom(data)
	defer miya.PutContext(ctx)

	// Render directly
	return r.env.RenderString(source, ctx)
}

// Errors
var (
	ErrTemplateNotFound = errors.New("template not found")
	ErrRenderFailed     = errors.New("render failed")
)
