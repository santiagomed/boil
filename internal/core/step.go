package core

import (
	"boil/internal/config"
	"boil/internal/llm"
	"boil/internal/tempdir"
	"boil/internal/utils"
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
	FinalDir       string
	Config         *config.Config
	LLM            *llm.Client
}
