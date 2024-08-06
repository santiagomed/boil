package core

import (
	"github.com/santiagomed/boil/pkg/config"
	"github.com/santiagomed/boil/pkg/fs"
	"github.com/santiagomed/boil/pkg/llm"

	"github.com/rs/zerolog"
)

type Engine struct {
	pipeline *Pipeline
}

func NewProjectEngine(config *config.Config, llm *llm.Client, pub StepPublisher, logger *zerolog.Logger) *Engine {
	pipeline := NewPipeline(config, llm, pub, logger)
	pipeline.state.FileSystem = fs.NewMemoryFileSystem()

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
