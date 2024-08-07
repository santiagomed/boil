package core

import (
	"fmt"
	"time"

	"github.com/santiagomed/boil/pkg/config"
	"github.com/santiagomed/boil/pkg/fs"
	"github.com/santiagomed/boil/pkg/llm"
	"github.com/santiagomed/boil/pkg/logger"
)

type Step interface {
	Execute(state *State) error
}

type StepType int

const (
	GenerateProjectDetails StepType = iota
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
	ProjectDetails string
	FileTree       string
	FileOperations []fs.FileOperation
	FileOrder      []string
	PreviousFiles  map[string]string
	Config         *config.Config
	Logger         logger.Logger
}

type Pipeline struct {
	stepManager *StepManager
	state       *State
	publisher   StepPublisher
}

func NewPipeline(config *config.Config, pub StepPublisher, logger logger.Logger) (*Pipeline, error) {
	fs := fs.NewMemoryFileSystem()
	llmCfg := llm.LlmConfig{
		OpenAIAPIKey: config.OpenAIAPIKey,
		ModelName:    config.ModelName,
		ProjectName:  config.ProjectName,
	}
	llm, err := llm.NewClient(&llmCfg)
	if err != nil {
		return nil, err
	}
	stepManager := NewStepManager(llm, fs)
	return &Pipeline{
		state: &State{
			Config:        config,
			PreviousFiles: make(map[string]string),
			Logger:        logger,
		},
		publisher:   pub,
		stepManager: stepManager,
	}, nil
}

func (p *Pipeline) Execute(projectDesc string) error {
	p.state.ProjectDesc = projectDesc
	p.state.Logger.Info("Starting pipeline execution")
	for i, stepType := range p.stepManager.steps {
		p.state.Logger.Info(fmt.Sprintf("Attempting to execute step %d: %v", i, stepType))
		step := p.stepManager.GetStep(stepType)
		if step == nil {
			p.state.Logger.Error(fmt.Sprintf("Step %v not found", stepType))
			p.publisher.Error(stepType, fmt.Errorf("step %v not found", stepType))
			return fmt.Errorf("step %v not found", stepType)
		}

		startTime := time.Now()
		if err := step.Execute(p.state); err != nil {
			p.state.Logger.Error(fmt.Sprintf("Error executing step %v", stepType))
			p.publisher.Error(stepType, err)
			return err
		}
		duration := time.Since(startTime)
		p.state.Logger.Info(fmt.Sprintf("Step %v completed in %v", stepType, duration))
		p.publisher.PublishStep(stepType)

		if i < len(p.stepManager.steps)-1 {
			p.state.Logger.Info(fmt.Sprintf("Transitioning from step %v to step %v", stepType, p.stepManager.steps[i+1]))
		}
	}

	p.state.Logger.Info("Pipeline execution completed")
	return nil
}

type StepPublisher interface {
	PublishStep(step StepType)
	Error(step StepType, err error)
}

type DefaultStepPublisher struct{}

func (p *DefaultStepPublisher) PublishStep(step StepType) {}

func (p *DefaultStepPublisher) Error(step StepType, err error) {}
