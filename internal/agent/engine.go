package agent

import (
	"io"

	"github.com/grahms/promptweaver"
)

type AgentEngine struct {
	engine *promptweaver.Engine
	sink   *promptweaver.HandlerSink
}

// NewAgent constructs and returns an initialized AgentEngine.
// It builds the promptweaver registry, creates a HandlerSink that routes UI output via uiOut and uses workspace for storage/paths, and instantiates the underlying promptweaver Engine wired to that registry.
func NewAgent(uiOut func(string), workspace string) *AgentEngine {
	reg := BuildRegistry()
	sink := BuildSink(uiOut, workspace)
	engine := promptweaver.NewEngine(reg)

	return &AgentEngine{engine, sink}
}

func (a *AgentEngine) Process(r io.Reader) error {
	return a.engine.ProcessStream(r, a.sink)
}
