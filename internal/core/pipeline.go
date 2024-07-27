package core

import (
	"boil/internal/config"
	"boil/internal/llm"
	"boil/internal/tempdir"
	"boil/internal/utils"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

type Step interface {
	Execute(state *State) error
}

type StepType int

const (
	CreateTempDir StepType = iota
	GenerateProjectDetails
	GenerateFileTree
	GenerateFileOperations
	ExecuteFileOperations
	DetermineFileOrder
	GenerateFileContents
	CreateOptionalComponents
	FinalizeProject
)

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
	Llm            *llm.Client
	Logger         *zerolog.Logger
}

type Pipeline struct {
	steps     []StepType
	state     *State
	publisher StepPublisher
}

func NewPipeline(config *config.Config, llm *llm.Client, pub StepPublisher, logger *zerolog.Logger) *Pipeline {
	return &Pipeline{
		state: &State{
			Config:        config,
			Llm:           llm,
			PreviousFiles: make(map[string]string),
			Logger:        logger,
		},
		publisher: pub,
	}
}

func (p *Pipeline) AddStep(step StepType) {
	p.steps = append(p.steps, step)
}

func (p *Pipeline) Execute(projectDesc string) {
	p.state.ProjectDesc = projectDesc
	p.state.Logger.Debug().Msg("Starting pipeline execution")

	for i, stepType := range p.steps {
		p.state.Logger.Debug().Msgf("Attempting to execute step %d: %v", i, stepType)
		step := GetStep(stepType)
		if step == nil {
			p.state.Logger.Error().Msgf("Step %v not found", stepType)
			p.publisher.Error(stepType, fmt.Errorf("step %v not found", stepType))
			return
		}

		startTime := time.Now()
		if err := step.Execute(p.state); err != nil {
			p.state.Logger.Error().Err(err).Msgf("Error executing step %v", stepType)
			p.publisher.Error(stepType, err)
			return
		}
		duration := time.Since(startTime)
		p.state.Logger.Debug().Msgf("Step %v completed in %v", stepType, duration)
		p.publisher.PublishStep(stepType)

		if i < len(p.steps)-1 {
			p.state.Logger.Debug().Msgf("Transitioning from step %v to step %v", stepType, p.steps[i+1])
		}
	}

	p.state.Logger.Debug().Msg("Pipeline execution completed")
}

type StepPublisher interface {
	PublishStep(step StepType)
	Error(step StepType, err error)
}

type DefaultStepPublisher struct{}

func (p *DefaultStepPublisher) PublishStep(step StepType) {}

func (p *DefaultStepPublisher) Error(step StepType, err error) {}
