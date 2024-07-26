package core

import (
	"boil/internal/config"
	"boil/internal/llm"

	"github.com/rs/zerolog"
)

type Engine struct {
	pipeline *Pipeline
}

func NewProjectEngine(config *config.Config, llm *llm.Client, pub StepPublisher, logger *zerolog.Logger) *Engine {
	pipeline := NewPipeline(config, llm, pub, logger)

	pipeline.AddStep(InitialStepType)
	pipeline.AddStep(CreateTempDirType)
	pipeline.AddStep(GenerateProjectDetailsType)
	pipeline.AddStep(GenerateFileTreeType)
	pipeline.AddStep(GenerateFileOperationsType)
	pipeline.AddStep(ExecuteFileOperationsType)
	pipeline.AddStep(DetermineFileOrderType)
	pipeline.AddStep(GenerateFileContentsType)
	pipeline.AddStep(CreateOptionalComponentsType)
	pipeline.AddStep(FinalizeProjectType)

	return &Engine{pipeline}
}

func (e *Engine) Generate(projectDesc string) {
	e.pipeline.Execute(projectDesc)
}

func (e *Engine) CleanupTempDir() error {
	return e.pipeline.state.TempDir.Cleanup()
}
