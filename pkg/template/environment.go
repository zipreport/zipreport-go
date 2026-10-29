package template

import (
	"github.com/zipreport/miya"
	"github.com/zipreport/zipreport-go/pkg/report"
)

// EnvironmentWrapper allows customization of the miya Environment.
// Implement this interface to add custom filters, globals, or tests.
type EnvironmentWrapper interface {
	// Configure is called to configure the miya Environment.
	Configure(env *miya.Environment)
}

// FuncEnvironmentWrapper wraps a function as an EnvironmentWrapper.
type FuncEnvironmentWrapper func(env *miya.Environment)

// Configure calls the wrapped function.
func (f FuncEnvironmentWrapper) Configure(env *miya.Environment) {
	f(env)
}

// ReportWrapper is an EnvironmentWrapper that also receives the ReportFile being rendered.
// MiyaRender calls ConfigureReport instead of Configure, passing the report it renders into;
// for api.ZipReport and api.MIMEReport that is the per-render copy.
// A non-nil error from ConfigureReport fails the render with that error, wrapped.
// One wrapper value may serve concurrent renders: do not store r anywhere shared.
type ReportWrapper interface {
	EnvironmentWrapper
	ConfigureReport(env *miya.Environment, r *report.ReportFile) error
}

// FuncReportWrapper wraps a function as a ReportWrapper.
type FuncReportWrapper func(env *miya.Environment, r *report.ReportFile) error

// ConfigureReport calls the wrapped function.
func (f FuncReportWrapper) ConfigureReport(env *miya.Environment, r *report.ReportFile) error {
	return f(env, r)
}

// Configure calls the wrapped function with a nil ReportFile and discards its error.
// MiyaRender and ChainedWrapper never call it, but a custom composite wrapper that only
// forwards Configure does, so the function must handle r == nil.
func (f FuncReportWrapper) Configure(env *miya.Environment) {
	_ = f(env, nil)
}

var _ ReportWrapper = FuncReportWrapper(nil)

// ChainedWrapper combines multiple EnvironmentWrappers.
type ChainedWrapper struct {
	wrappers []EnvironmentWrapper
}

// NewChainedWrapper creates a ChainedWrapper from multiple wrappers.
func NewChainedWrapper(wrappers ...EnvironmentWrapper) *ChainedWrapper {
	return &ChainedWrapper{wrappers: wrappers}
}

// Configure calls Configure on all wrapped wrappers in order.
func (c *ChainedWrapper) Configure(env *miya.Environment) {
	for _, w := range c.wrappers {
		w.Configure(env)
	}
}

// Add adds a wrapper to the chain.
func (c *ChainedWrapper) Add(wrapper EnvironmentWrapper) {
	c.wrappers = append(c.wrappers, wrapper)
}

// configureReport applies each wrapper in order, passing r to nested chains and ReportWrappers,
// and stops at the first error. It is unexported so types embedding *ChainedWrapper
// do not become ReportWrappers.
func (c *ChainedWrapper) configureReport(env *miya.Environment, r *report.ReportFile) error {
	for _, w := range c.wrappers {
		switch w := w.(type) {
		case *ChainedWrapper:
			if err := w.configureReport(env, r); err != nil {
				return err
			}
		case ReportWrapper:
			if err := w.ConfigureReport(env, r); err != nil {
				return err
			}
		default:
			w.Configure(env)
		}
	}
	return nil
}
