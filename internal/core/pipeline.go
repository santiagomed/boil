package core

import (
	"boil/internal/config"
	"boil/internal/llm"
	"boil/internal/tempdir"
	"boil/internal/utils"

	"github.com/rs/zerolog"
)

type Step interface {
	Execute(state *State) error
}

type State struct {
	ProjectDesc    string
	TempDir        *tempdir.Manager
	TempDirPath    string
	ProjectDetails string
	FileTree       string
	FileOperations []utils.FileOperation
	FileOrder      []string
	PreviousFiles  map[string]string
	Config         *config.Config
	LLM            *llm.Client
	Logger         *zerolog.Logger
}

type Pipeline struct {
	steps []Step
	state *State
}

func NewPipeline(config *config.Config, llm *llm.Client, logger *zerolog.Logger) *Pipeline {
	return &Pipeline{
		steps: []Step{},
		state: &State{
			Config:        config,
			LLM:           llm,
			PreviousFiles: make(map[string]string),
			Logger:        logger,
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
