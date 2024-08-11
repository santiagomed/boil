package core

import (
	"context"
	"fmt"
	"time"

	"github.com/santiagomed/boil/fs"
	"github.com/santiagomed/boil/llm"
	"github.com/santiagomed/boil/logger"
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
	Done
)

type State struct {
	ProjectDetails string
	FileTree       string
	FileOperations []fs.FileOperation
	FileOrder      []string
	PreviousFiles  map[string]string
	Request        *Request
	Logger         logger.Logger
}

type Pipeline struct {
	stepManager StepManager
	state       *State
	publisher   StepPublisher
}

func NewPipeline(r *Request, llm llm.LLMClient, sm StepManager, pub StepPublisher, logger logger.Logger) (*Pipeline, error) {
	return &Pipeline{
		state: &State{
			Request:       r,
			PreviousFiles: make(map[string]string),
			Logger:        logger,
		},
		publisher:   pub,
		stepManager: sm,
	}, nil
}

func (p *Pipeline) Execute(ctx context.Context) error {
	steps := p.stepManager.GetSteps()
	p.state.Logger.Info("Starting pipeline execution")
	for i, stepType := range steps {
		select {
		case <-ctx.Done():
			p.state.Logger.Info("Pipeline execution cancelled")
			return context.Canceled
		default:
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

			if i < len(steps)-1 {
				p.state.Logger.Info(fmt.Sprintf("Transitioning from step %v to step %v", stepType, steps[i+1]))
			}
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
