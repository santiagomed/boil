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
