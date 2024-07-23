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

func (e *Engine) Generate(projectDesc string) (string, error) {
	e.pipeline.AddStep(&CreateTempDirStep{})
	e.pipeline.AddStep(&GenerateProjectDetailsStep{})
	e.pipeline.AddStep(&GenerateFileTreeStep{})
	e.pipeline.AddStep(&GenerateFileOperationsStep{})
	e.pipeline.AddStep(&ExecuteFileOperationsStep{})
	e.pipeline.AddStep(&DetermineFileOrderStep{})
	e.pipeline.AddStep(&GenerateFileContentsStep{})
	e.pipeline.AddStep(&CreateOptionalComponentsStep{})
	e.pipeline.AddStep(&FinalizeProjectStep{})

	if err := e.pipeline.Execute(projectDesc); err != nil {
		return "", err
	}

	return e.pipeline.state.Config.OutputDir, nil
}

func (e *Engine) CleanupTempDir() error {
	return e.pipeline.state.TempDir.Cleanup()
}
