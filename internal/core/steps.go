package core

import (
	"boil/internal/tempdir"
	"boil/internal/utils"
	"fmt"
	"path/filepath"
)

type CreateTempDirStep struct{}

func (s *CreateTempDirStep) Execute(state *State) error {
	state.TempDir = tempdir.NewManager(state.Config)
	path, err := state.TempDir.CreateTempDir("boil")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	state.TempDirPath = path
	return nil
}

type GenerateProjectDetailsStep struct{}

func (s *GenerateProjectDetailsStep) Execute(state *State) error {
	details, err := state.LLM.GenerateProjectDetails(state.ProjectDesc)
	if err != nil {
		return fmt.Errorf("failed to generate project details: %w", err)
	}
	state.ProjectDetails = details
	return nil
}

type GenerateFileTreeStep struct{}

func (s *GenerateFileTreeStep) Execute(state *State) error {
	fileTree, err := state.LLM.GenerateFileTree(state.ProjectDetails)
	if err != nil {
		return fmt.Errorf("failed to generate file tree: %w", err)
	}
	state.FileTree = fileTree
	return nil
}

type GenerateFileOperationsStep struct{}

func (s *GenerateFileOperationsStep) Execute(state *State) error {
	operations, err := state.LLM.GenerateFileOperations(state.ProjectDetails, state.FileTree)
	if err != nil {
		return fmt.Errorf("failed to generate file operations: %w", err)
	}
	state.FileOperations = operations
	return nil
}

type ExecuteFileOperationsStep struct{}

func (s *ExecuteFileOperationsStep) Execute(state *State) error {
	err := utils.ExecuteFileOperations(state.TempDirPath, state.FileOperations)
	if err != nil {
		return fmt.Errorf("failed to execute file operations: %w", err)
	}
	return nil
}

type DetermineFileOrderStep struct{}

func (s *DetermineFileOrderStep) Execute(state *State) error {
	order, err := state.LLM.DetermineFileOrder(state.FileTree)
	if err != nil {
		return fmt.Errorf("failed to determine file creation order: %w", err)
	}
	state.FileOrder = order
	return nil
}

type GenerateFileContentsStep struct{}

func (s *GenerateFileContentsStep) Execute(state *State) error {
	for _, file := range state.FileOrder {
		content, err := state.LLM.GenerateFileContent(file, state.ProjectDetails, state.FileTree, state.PreviousFiles)
		if err != nil {
			return fmt.Errorf("failed to generate content for file %s: %w", file, err)
		}
		err = utils.WriteFile(filepath.Join(state.TempDirPath, file), content)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", file, err)
		}
		state.PreviousFiles[file] = content
	}
	return nil
}

type CreateOptionalComponentsStep struct{}

func (s *CreateOptionalComponentsStep) Execute(state *State) error {
	if state.Config.GitRepo {
		if err := utils.InitializeGitRepo(state.TempDirPath); err != nil {
			return fmt.Errorf("failed to initialize git repository: %w", err)
		}
	}

	if state.Config.GitIgnore {
		if err := utils.CreateGitIgnore(state.TempDirPath); err != nil {
			return fmt.Errorf("failed to create .gitignore file: %w", err)
		}
	}

	if state.Config.Readme {
		readme, err := state.LLM.GenerateReadmeContent(state.ProjectDetails)
		if err != nil {
			return fmt.Errorf("failed to generate README: %w", err)
		}
		if err := utils.WriteFile(filepath.Join(state.TempDirPath, "README.md"), readme); err != nil {
			return fmt.Errorf("failed to create README file: %w", err)
		}
	}

	if state.Config.Dockerfile {
		dockerfile, err := state.LLM.GenerateDockerfileContent(state.ProjectDetails)
		if err != nil {
			return fmt.Errorf("failed to generate Dockerfile: %w", err)
		}
		if err := utils.WriteFile(filepath.Join(state.TempDirPath, "Dockerfile"), dockerfile); err != nil {
			return fmt.Errorf("failed to create Dockerfile: %w", err)
		}
	}

	return nil
}

type FinalizeProjectStep struct{}

func (s *FinalizeProjectStep) Execute(state *State) error {
	projectName := utils.FormatProjectName(filepath.Base(state.FinalDir))
	finalPath := filepath.Join(state.Config.OutputDir, projectName)

	if err := utils.MoveDir(state.TempDirPath, finalPath); err != nil {
		return fmt.Errorf("failed to move project to final directory: %w", err)
	}

	state.FinalDir = finalPath
	return nil
}
