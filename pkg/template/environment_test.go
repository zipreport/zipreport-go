package template

import (
	"testing"

	"github.com/zipreport/miya"
)

func TestFuncEnvironmentWrapper(t *testing.T) {
	called := false
	wrapper := FuncEnvironmentWrapper(func(env *miya.Environment) {
		called = true
	})

	env := miya.NewEnvironment()
	wrapper.Configure(env)

	if !called {
		t.Error("FuncEnvironmentWrapper.Configure did not call the function")
	}
}

func TestNewChainedWrapper(t *testing.T) {
	w1 := FuncEnvironmentWrapper(func(env *miya.Environment) {})
	w2 := FuncEnvironmentWrapper(func(env *miya.Environment) {})

	chain := NewChainedWrapper(w1, w2)

	if chain == nil {
		t.Fatal("NewChainedWrapper returned nil")
	}
	if len(chain.wrappers) != 2 {
		t.Errorf("wrappers length = %d, want 2", len(chain.wrappers))
	}
}

func TestChainedWrapper_Configure(t *testing.T) {
	callOrder := []int{}

	w1 := FuncEnvironmentWrapper(func(env *miya.Environment) {
		callOrder = append(callOrder, 1)
	})
	w2 := FuncEnvironmentWrapper(func(env *miya.Environment) {
		callOrder = append(callOrder, 2)
	})
	w3 := FuncEnvironmentWrapper(func(env *miya.Environment) {
		callOrder = append(callOrder, 3)
	})

	chain := NewChainedWrapper(w1, w2, w3)

	env := miya.NewEnvironment()
	chain.Configure(env)

	if len(callOrder) != 3 {
		t.Errorf("Expected 3 calls, got %d", len(callOrder))
	}
	if callOrder[0] != 1 || callOrder[1] != 2 || callOrder[2] != 3 {
		t.Errorf("Call order = %v, want [1, 2, 3]", callOrder)
	}
}

func TestChainedWrapper_Add(t *testing.T) {
	chain := NewChainedWrapper()

	if len(chain.wrappers) != 0 {
		t.Error("Empty chain should have no wrappers")
	}

	w1 := FuncEnvironmentWrapper(func(env *miya.Environment) {})
	chain.Add(w1)

	if len(chain.wrappers) != 1 {
		t.Errorf("wrappers length = %d, want 1", len(chain.wrappers))
	}

	w2 := FuncEnvironmentWrapper(func(env *miya.Environment) {})
	chain.Add(w2)

	if len(chain.wrappers) != 2 {
		t.Errorf("wrappers length = %d, want 2", len(chain.wrappers))
	}
}

func TestChainedWrapper_Empty(t *testing.T) {
	chain := NewChainedWrapper()
	env := miya.NewEnvironment()

	// Should not panic with empty chain
	chain.Configure(env)
}
