package template

import (
	"errors"
	"testing"

	"github.com/zipreport/miya"
	"github.com/zipreport/zipreport-go/pkg/report"
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

func TestChainedWrapper_ForwardsReport(t *testing.T) {
	r := createTestReportForRender()
	var got *report.ReportFile
	runs := 0
	plain := FuncEnvironmentWrapper(func(env *miya.Environment) { runs++ })
	_, err := NewMiyaRender(r, WithWrapper(NewChainedWrapper(plain, recordingReportWrapper(&got)))).RenderString("x", nil)
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	if runs != 1 {
		t.Errorf("plain wrapper ran %d times, want 1", runs)
	}
	if got != r {
		t.Errorf("report wrapper got %p, want %p", got, r)
	}
}

func TestChainedWrapper_NestedForwardsReport(t *testing.T) {
	r := createTestReportForRender()
	var got *report.ReportFile
	w := NewChainedWrapper(NewChainedWrapper(recordingReportWrapper(&got)))
	if _, err := NewMiyaRender(r, WithWrapper(w)).RenderString("x", nil); err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	if got != r {
		t.Errorf("nested report wrapper got %p, want %p", got, r)
	}
}

type embeddingChain struct{ *ChainedWrapper }

func (c embeddingChain) Configure(env *miya.Environment) {
	env.AddGlobal("e", "E")
	c.ChainedWrapper.Configure(env)
}

func TestChainedWrapper_EmbedderConfigureStillCalled(t *testing.T) {
	r := createTestReportForRender()
	out, err := NewMiyaRender(r, WithWrapper(embeddingChain{NewChainedWrapper()})).RenderString("{{ e }}", nil)
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	if out != "E" {
		t.Errorf("output %q, want E", out)
	}
}

func TestChainedWrapper_StopsAtFirstError(t *testing.T) {
	r := createTestReportForRender()
	laterRan := false
	later := FuncEnvironmentWrapper(func(env *miya.Environment) { laterRan = true })
	_, err := NewMiyaRender(r, WithWrapper(NewChainedWrapper(FuncReportWrapper(failingReportWrapper), later))).RenderString("x", nil)
	if !errors.Is(err, errTestWrapper) {
		t.Errorf("err = %v, want errTestWrapper", err)
	}
	if laterRan {
		t.Error("wrapper after the failing one ran")
	}
}

func TestFuncReportWrapper_ConfigureNilReport(t *testing.T) {
	got := &report.ReportFile{}
	called := false
	fn := func(env *miya.Environment, r *report.ReportFile) error {
		called = true
		got = r
		return errTestWrapper
	}
	FuncReportWrapper(fn).Configure(miya.NewEnvironment())
	if !called {
		t.Error("function not called")
	}
	if got != nil {
		t.Errorf("got %p, want nil report", got)
	}
}
