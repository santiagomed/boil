package core

import (
	"boil/internal/config"
	"boil/internal/llm"

	"github.com/rs/zerolog"
)

type Engine struct {
	pipeline *Pipeline
}

func NewProjectEngine(config *config.Config, llm *llm.Client, logger *zerolog.Logger) *Engine {
	return &Engine{
		pipeline: NewPipeline(config, llm, logger),
	}
}

func (e *Engine) Generate(projectDesc string) error {
	e.pipeline.AddStep(&CreateTempDirStep{})
	e.pipeline.AddStep(&GenerateProjectDetailsStep{})
	e.pipeline.AddStep(&GenerateFileTreeStep{})
	e.pipeline.AddStep(&GenerateFileOperationsStep{})
	e.pipeline.AddStep(&ExecuteFileOperationsStep{})
	e.pipeline.AddStep(&DetermineFileOrderStep{})
	e.pipeline.AddStep(&GenerateFileContentsStep{})
	e.pipeline.AddStep(&CreateOptionalComponentsStep{})
	e.pipeline.AddStep(&FinalizeProjectStep{})

	return e.pipeline.Execute(projectDesc)
}

func (e *Engine) CleanupTempDir() error {
	return e.pipeline.state.TempDir.Cleanup()
}
