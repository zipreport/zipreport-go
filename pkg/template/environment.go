package template

import "github.com/zipreport/miya"

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
