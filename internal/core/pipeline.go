package core

import (
	"boil/internal/config"
	"boil/internal/llm"
)

type Pipeline struct {
	steps []Step
	state *State
}

func NewPipeline(config *config.Config, llm *llm.Client) *Pipeline {
	return &Pipeline{
		steps: []Step{},
		state: &State{
			Config:        config,
			LLM:           llm,
			PreviousFiles: make(map[string]string),
		},
	}
}

func (p *Pipeline) AddStep(step Step) {
	p.steps = append(p.steps, step)
}

func (p *Pipeline) Execute(projectDesc string) error {
	p.state.ProjectDesc = projectDesc

	for _, step := range p.steps {
		if err := step.Execute(p.state); err != nil {
			return err
		}
	}

	return nil
}
