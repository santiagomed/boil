package core

import (
	"boil/internal/config"
	"boil/internal/llm"
)

type Engine struct {
	pipeline *Pipeline
}

func NewProjectEngine(config *config.Config, llm *llm.Client) *Engine {
	return &Engine{
		pipeline: NewPipeline(config, llm),
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

	return e.pipeline.state.FinalDir, nil
}

func (e *Engine) CleanupTempDir() error {
	return e.pipeline.state.TempDir.Cleanup()
}
