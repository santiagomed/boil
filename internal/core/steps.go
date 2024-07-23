package core

import (
	"boil/internal/tempdir"
	"boil/internal/utils"
	"fmt"
	"path/filepath"
)

type CreateTempDirStep struct{}

func (s *CreateTempDirStep) Execute(state *State) error {
	state.Logger.Info().Msg("Creating temporary directory...")
	state.TempDir = tempdir.NewManager(state.Config)
	path, err := state.TempDir.CreateTempDir("boil")
	if err != nil {
		state.Logger.Error().Err(err).Msg("Failed to create temporary directory")
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	state.TempDirPath = path
	state.Logger.Info().Msg("Temporary directory created successfully")
	return nil
}

type GenerateProjectDetailsStep struct{}

func (s *GenerateProjectDetailsStep) Execute(state *State) error {
	state.Logger.Info().Msg("Generating project details...")
	details, err := state.LLM.GenerateProjectDetails(state.ProjectDesc)
	if err != nil {
		state.Logger.Error().Err(err).Msg("Failed to generate project details")
		return fmt.Errorf("failed to generate project details: %w", err)
	}
	state.ProjectDetails = details
	state.Logger.Info().Msg("Project details generated successfully")
	return nil
}

type GenerateFileTreeStep struct{}

func (s *GenerateFileTreeStep) Execute(state *State) error {
	state.Logger.Info().Msg("Generating file tree...")
	fileTree, err := state.LLM.GenerateFileTree(state.ProjectDetails)
	if err != nil {
		state.Logger.Error().Err(err).Msg("Failed to generate file tree")
		return fmt.Errorf("failed to generate file tree: %w", err)
	}
	state.FileTree = fileTree
	state.Logger.Info().Msg("File tree generated successfully")
	return nil
}

type GenerateFileOperationsStep struct{}

func (s *GenerateFileOperationsStep) Execute(state *State) error {
	state.Logger.Info().Msg("Generating file operations...")
	operations, err := state.LLM.GenerateFileOperations(state.ProjectDetails, state.FileTree)
	if err != nil {
		state.Logger.Error().Err(err).Msg("Failed to generate file operations")
		return fmt.Errorf("failed to generate file operations: %w", err)
	}
	state.FileOperations = operations
	state.Logger.Info().Msg("File operations generated successfully")
	return nil
}

type ExecuteFileOperationsStep struct{}

func (s *ExecuteFileOperationsStep) Execute(state *State) error {
	state.Logger.Info().Msg("Executing file operations...")
	err := utils.ExecuteFileOperations(state.TempDirPath, state.FileOperations)
	if err != nil {
		state.Logger.Error().Err(err).Msg("Failed to execute file operations")
		return fmt.Errorf("failed to execute file operations: %w", err)
	}
	state.Logger.Info().Msg("File operations executed successfully")
	return nil
}

type DetermineFileOrderStep struct{}

func (s *DetermineFileOrderStep) Execute(state *State) error {
	state.Logger.Info().Msg("Determining file creation order...")
	order, err := state.LLM.DetermineFileOrder(state.FileTree)
	if err != nil {
		state.Logger.Error().Err(err).Msg("Failed to determine file creation order")
		return fmt.Errorf("failed to determine file creation order: %w", err)
	}
	state.FileOrder = order
	state.Logger.Info().Msg("File creation order determined successfully")
	return nil
}

type GenerateFileContentsStep struct{}

func (s *GenerateFileContentsStep) Execute(state *State) error {
	state.Logger.Info().Msg("Generating file contents...")
	for _, file := range state.FileOrder {
		state.Logger.Info().Msgf("Generating content for file %s...", file)
		content, err := state.LLM.GenerateFileContent(file, state.ProjectDetails, state.FileTree, state.PreviousFiles)
		if err != nil {
			state.Logger.Error().Err(err).Msgf("Failed to generate content for file %s", file)
			return fmt.Errorf("failed to generate content for file %s: %w", file, err)
		}
		err = utils.WriteFile(filepath.Join(state.TempDirPath, file), content)
		if err != nil {
			state.Logger.Error().Err(err).Msgf("Failed to create file %s", file)
			return fmt.Errorf("failed to create file %s: %w", file, err)
		}
		state.PreviousFiles[file] = content
		state.Logger.Info().Msgf("Content generated for file %s", file)
	}
	state.Logger.Info().Msg("All file contents generated successfully")
	return nil
}

type CreateOptionalComponentsStep struct{}

func (s *CreateOptionalComponentsStep) Execute(state *State) error {
	state.Logger.Info().Msg("Creating optional components...")

	if state.Config.GitRepo {
		state.Logger.Info().Msg("Initializing Git repository...")
		if err := utils.InitializeGitRepo(state.TempDirPath); err != nil {
			state.Logger.Error().Err(err).Msg("Failed to initialize Git repository")
			return fmt.Errorf("failed to initialize git repository: %w", err)
		}
		state.Logger.Info().Msg("Git repository initialized successfully")
	}

	if state.Config.GitIgnore {
		state.Logger.Info().Msg("Creating .gitignore file...")
		if err := utils.CreateGitIgnore(state.TempDirPath); err != nil {
			state.Logger.Error().Err(err).Msg("Failed to create .gitignore file")
			return fmt.Errorf("failed to create .gitignore file: %w", err)
		}
		state.Logger.Info().Msg(".gitignore file created successfully")
	}

	if state.Config.Readme {
		state.Logger.Info().Msg("Generating README.md...")
		readme, err := state.LLM.GenerateReadmeContent(state.ProjectDetails)
		if err != nil {
			state.Logger.Error().Err(err).Msg("Failed to generate README")
			return fmt.Errorf("failed to generate README: %w", err)
		}
		if err := utils.WriteFile(filepath.Join(state.TempDirPath, "README.md"), readme); err != nil {
			state.Logger.Error().Err(err).Msg("Failed to create README file")
			return fmt.Errorf("failed to create README file: %w", err)
		}
		state.Logger.Info().Msg("README.md created successfully")
	}

	if state.Config.Dockerfile {
		state.Logger.Info().Msg("Generating Dockerfile...")
		dockerfile, err := state.LLM.GenerateDockerfileContent(state.ProjectDetails)
		if err != nil {
			state.Logger.Error().Err(err).Msg("Failed to generate Dockerfile")
			return fmt.Errorf("failed to generate Dockerfile: %w", err)
		}
		if err := utils.WriteFile(filepath.Join(state.TempDirPath, "Dockerfile"), dockerfile); err != nil {
			state.Logger.Error().Err(err).Msg("Failed to create Dockerfile")
			return fmt.Errorf("failed to create Dockerfile: %w", err)
		}
		state.Logger.Info().Msg("Dockerfile created successfully")
	}

	state.Logger.Info().Msg("Optional components created successfully")
	return nil
}

type FinalizeProjectStep struct{}

func (s *FinalizeProjectStep) Execute(state *State) error {
	state.Logger.Info().Msg("Finalizing project...")
	projectName := utils.FormatProjectName(filepath.Base(state.Config.OutputDir))
	finalPath := filepath.Join(state.Config.OutputDir, projectName)

	state.Logger.Printf("Moving project from %s to %s\n", state.TempDirPath, finalPath)
	if err := utils.MoveDir(state.TempDirPath, finalPath); err != nil {
		state.Logger.Error().Err(err).Msg("Failed to move project to final directory")
		return fmt.Errorf("failed to move project to final directory: %w", err)
	}

	state.Logger.Info().Msg("Project finalized successfully")
	return nil
}
