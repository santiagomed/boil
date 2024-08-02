package core

import (
	"github.com/santiagomed/boil/internal/llm"
	"github.com/santiagomed/boil/pkg/config"

	"github.com/rs/zerolog"
)

type Engine struct {
	pipeline *Pipeline
}

func NewProjectEngine(config *config.Config, llm *llm.Client, pub StepPublisher, logger *zerolog.Logger) *Engine {
	pipeline := NewPipeline(config, llm, pub, logger)

	pipeline.AddStep(CreateTempDir)
	pipeline.AddStep(GenerateProjectDetails)
	pipeline.AddStep(GenerateFileTree)
	pipeline.AddStep(GenerateFileOperations)
	pipeline.AddStep(ExecuteFileOperations)
	pipeline.AddStep(DetermineFileOrder)
	pipeline.AddStep(GenerateFileContents)
	pipeline.AddStep(CreateOptionalComponents)
	pipeline.AddStep(FinalizeProject)

	return &Engine{pipeline}
}

func (e *Engine) Generate(projectDesc string) {
	e.pipeline.Execute(projectDesc)
}

func (e *Engine) CleanupTempDir() error {
	return e.pipeline.state.TempDir.Cleanup()
}
