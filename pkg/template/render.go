// Package template provides Jinja2-compatible template rendering using miya.
package template

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"

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
// A MiyaRender renders once: Render, RenderTemplate and RenderString may be called a single time in total; later calls return ErrRendererUsed (if filter registration failed, every call returns that error instead). Configure it with GetEnvironment/SetWrapper before the render.
type MiyaRender struct {
	report      *report.ReportFile
	env         *miya.Environment
	options     *RenderOptions
	wrapper     EnvironmentWrapper
	filterError error
	used        atomic.Bool
}

// claim marks the renderer as used; only the first call succeeds.
func (r *MiyaRender) claim() error {
	if !r.used.CompareAndSwap(false, true) {
		return ErrRendererUsed
	}
	return nil
}

// NewMiyaRender creates a new MiyaRender for a report.
// Rendering writes the output (and any generated images) into r; render r.Clone() to keep r unchanged.
// Create a new renderer for each render.
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
		// Reported by Render and RenderString
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
	if r.filterError != nil {
		return "", r.filterError
	}
	if err := r.claim(); err != nil {
		return "", err
	}
	// Work on a copy so the caller's map is never modified
	merged := make(map[string]interface{}, len(data))
	for k, v := range data {
		merged[k] = v
	}
	data = merged

	// Merge default data from data.json (provided data takes precedence)
	if r.report.Exists(report.FileData) {
		defaultData, err := r.loadDefaultData()
		if err != nil {
			return "", fmt.Errorf("invalid %s: %w", report.FileData, err)
		}
		for k, v := range defaultData {
			if _, exists := data[k]; !exists {
				data[k] = v
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
	if err := r.applyWrapper(); err != nil {
		return "", fmt.Errorf("environment wrapper: %w", err)
	}

	// Load and compile template
	tmpl, err := r.env.GetTemplate(templateName)
	if err != nil {
		return "", fmt.Errorf("failed to load template %s: %w", templateName, err)
	}

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

	// An empty file means no defaults; tolerate a UTF-8 byte-order mark
	content = bytes.TrimPrefix(content, []byte("\xef\xbb\xbf"))
	if len(bytes.TrimSpace(content)) == 0 {
		return nil, nil
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
	if r.filterError != nil {
		return "", r.filterError
	}
	if err := r.claim(); err != nil {
		return "", err
	}
	// Apply wrapper if present
	if err := r.applyWrapper(); err != nil {
		return "", fmt.Errorf("environment wrapper: %w", err)
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
	ErrRendererUsed     = errors.New("MiyaRender already used; create a new renderer for each render")
)

// applyWrapper configures the environment, passing the report to report-aware wrappers.
func (r *MiyaRender) applyWrapper() error {
	switch w := r.wrapper.(type) {
	case nil:
	case *ChainedWrapper:
		return w.configureReport(r.env, r.report)
	case ReportWrapper:
		return w.ConfigureReport(r.env, r.report)
	default:
		w.Configure(r.env)
	}
	return nil
}
