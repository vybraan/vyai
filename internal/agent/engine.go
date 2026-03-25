package agent

import (
	"io"

	"github.com/grahms/promptweaver"
)

type AgentEngine struct {
	engine *promptweaver.Engine
	sink   *promptweaver.HandlerSink
}

func NewAgent(uiOut func(string), workspace string) *AgentEngine {
	reg := BuildRegistry()
	sink := BuildSink(uiOut, workspace)
	engine := promptweaver.NewEngine(reg)

	return &AgentEngine{engine, sink}
}

func (a *AgentEngine) Process(r io.Reader) error {
	return a.engine.ProcessStream(r, a.sink)
}
