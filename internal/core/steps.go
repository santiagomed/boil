package core

import (
	"boil/internal/tempdir"
	"boil/internal/utils"
	"fmt"
	"os"
	"path/filepath"
)

var stepMap = map[StepType]Step{
	InitialStepType:              &InitialStep{},
	CreateTempDirType:            &CreateTempDir{},
	GenerateProjectDetailsType:   &GenerateProjectDetails{},
	GenerateFileTreeType:         &GenerateFileTree{},
	GenerateFileOperationsType:   &GenerateFileOperations{},
	ExecuteFileOperationsType:    &ExecuteFileOperations{},
	DetermineFileOrderType:       &DetermineFileOrder{},
	GenerateFileContentsType:     &GenerateFileContents{},
	CreateOptionalComponentsType: &CreateOptionalComponents{},
	FinalizeProjectType:          &FinalizeProject{},
}

func GetStep(stepType StepType) Step {
	return stepMap[stepType]
}

type InitialStep struct{}

func (s *InitialStep) Execute(state *State) error {
	state.Logger.Debug().Msg("Initializing pipeline execution.")
	return nil
}

type CreateTempDir struct{}

func (s *CreateTempDir) Execute(state *State) error {
	state.Logger.Debug().Msg("Creating temporary directory.")
	state.TempDir = tempdir.NewManager(state.Config)
	path, err := state.TempDir.CreateTempDir("boil")
	if err != nil {
		state.Logger.Error().Err(err).Msg("Failed to create temporary directory")
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	state.TempDirPath = path
	state.Logger.Debug().Msg("Temporary directory created successfully")
	return nil
}

type GenerateProjectDetails struct{}

func (s *GenerateProjectDetails) Execute(state *State) error {
	state.Logger.Debug().Msg("Generating project details.")
	details, err := state.Llm.GenerateProjectDetails(state.ProjectDesc)
	if err != nil {
		state.Logger.Error().Err(err).Msg("Failed to generate project details")
		return fmt.Errorf("failed to generate project details: %w", err)
	}
	state.ProjectDetails = details
	state.Logger.Debug().Msg("Project details generated successfully")
	return nil
}

type GenerateFileTree struct{}

func (s *GenerateFileTree) Execute(state *State) error {
	state.Logger.Debug().Msg("Generating file tree.")
	fileTree, err := state.Llm.GenerateFileTree(state.ProjectDetails)
	if err != nil {
		state.Logger.Error().Err(err).Msg("Failed to generate file tree")
		return fmt.Errorf("failed to generate file tree: %w", err)
	}
	state.FileTree = fileTree
	state.Logger.Debug().Msg("File tree generated successfully")
	return nil
}

type GenerateFileOperations struct{}

func (s *GenerateFileOperations) Execute(state *State) error {
	state.Logger.Debug().Msg("Generating file operations.")
	operations, err := state.Llm.GenerateFileOperations(state.ProjectDetails, state.FileTree)
	if err != nil {
		state.Logger.Error().Err(err).Msg("Failed to generate file operations")
		return fmt.Errorf("failed to generate file operations: %w", err)
	}
	state.FileOperations = operations
	state.Logger.Debug().Msg("File operations generated successfully")
	return nil
}

type ExecuteFileOperations struct{}

func (s *ExecuteFileOperations) Execute(state *State) error {
	state.Logger.Debug().Msg("Executing file operations.")
	err := utils.ExecuteFileOperations(state.TempDirPath, state.FileOperations)
	if err != nil {
		state.Logger.Error().Err(err).Msg("Failed to execute file operations")
		return fmt.Errorf("failed to execute file operations: %w", err)
	}
	state.Logger.Debug().Msg("File operations executed successfully")
	return nil
}

type DetermineFileOrder struct{}

func (s *DetermineFileOrder) Execute(state *State) error {
	state.Logger.Debug().Msg("Determining file creation order.")
	order, err := state.Llm.DetermineFileOrder(state.FileTree)
	if err != nil {
		state.Logger.Error().Err(err).Msg("Failed to determine file creation order")
		return fmt.Errorf("failed to determine file creation order: %w", err)
	}
	state.FileOrder = order
	state.Logger.Debug().Msg("File creation order determined successfully")
	return nil
}

type GenerateFileContents struct{}

func (s *GenerateFileContents) Execute(state *State) error {
	state.Logger.Debug().Msg("Generating file contents.")
	for _, file := range state.FileOrder {
		path := filepath.Join(state.TempDirPath, file)
		if utils.IsDir(path) || !utils.FileExists(path) {
			continue
		}
		state.Logger.Debug().Msgf("Generating content for file %s.", file)
		content, err := state.Llm.GenerateFileContent(file, state.ProjectDetails, state.FileTree, state.PreviousFiles)
		if err != nil {
			state.Logger.Error().Err(err).Msgf("Failed to generate content for file %s", file)
			return fmt.Errorf("failed to generate content for file %s: %w", file, err)
		}
		err = utils.WriteFile(path, content)
		if err != nil {
			state.Logger.Error().Err(err).Msgf("Failed to create file %s", file)
			return fmt.Errorf("failed to create file %s: %w", file, err)
		}
		state.PreviousFiles[file] = content
		state.Logger.Debug().Msgf("Content generated for file %s", file)
	}
	state.Logger.Debug().Msg("All file contents generated successfully")
	return nil
}

type CreateOptionalComponents struct{}

func (s *CreateOptionalComponents) Execute(state *State) error {
	state.Logger.Debug().Msg("Creating optional components.")

	if state.Config.GitRepo {
		state.Logger.Debug().Msg("Initializing Git repository.")
		if err := utils.InitializeGitRepo(state.TempDirPath); err != nil {
			state.Logger.Error().Err(err).Msg("Failed to initialize Git repository")
			return fmt.Errorf("failed to initialize git repository: %w", err)
		}
		state.Logger.Debug().Msg("Git repository initialized successfully")
	}

	if state.Config.GitIgnore {
		state.Logger.Debug().Msg("Creating .gitignore file.")
		gitignore, err := state.Llm.GenerateGitignoreContent(state.ProjectDetails)
		if err != nil {
			state.Logger.Error().Err(err).Msg("Failed to create .gitignore file")
			return fmt.Errorf("failed to create .gitignore file: %w", err)
		}
		if err := utils.WriteFile(filepath.Join(state.TempDirPath, ".gitignore"), gitignore); err != nil {
			state.Logger.Error().Err(err).Msg("Failed to create .gitignore file")
			return fmt.Errorf("failed to create .gitignore file: %w", err)
		}
		state.Logger.Debug().Msg(".gitignore file created successfully")
	}

	if state.Config.Readme {
		state.Logger.Debug().Msg("Generating README.md.")
		readme, err := state.Llm.GenerateReadmeContent(state.ProjectDetails)
		if err != nil {
			state.Logger.Error().Err(err).Msg("Failed to generate README")
			return fmt.Errorf("failed to generate README: %w", err)
		}
		if err := utils.WriteFile(filepath.Join(state.TempDirPath, "README.md"), readme); err != nil {
			state.Logger.Error().Err(err).Msg("Failed to create README file")
			return fmt.Errorf("failed to create README file: %w", err)
		}
		state.Logger.Debug().Msg("README.md created successfully")
	}

	if state.Config.Dockerfile {
		state.Logger.Debug().Msg("Generating Dockerfile.")
		dockerfile, err := state.Llm.GenerateDockerfileContent(state.ProjectDetails)
		if err != nil {
			state.Logger.Error().Err(err).Msg("Failed to generate Dockerfile")
			return fmt.Errorf("failed to generate Dockerfile: %w", err)
		}
		if err := utils.WriteFile(filepath.Join(state.TempDirPath, "Dockerfile"), dockerfile); err != nil {
			state.Logger.Error().Err(err).Msg("Failed to create Dockerfile")
			return fmt.Errorf("failed to create Dockerfile: %w", err)
		}
		state.Logger.Debug().Msg("Dockerfile created successfully")
	}

	state.Logger.Debug().Msg("Optional components created successfully")
	return nil
}

type FinalizeProject struct{}

func (s *FinalizeProject) Execute(state *State) error {
	state.Logger.Debug().Msg("Finalizing project.")
	projectName := utils.FormatProjectName(state.Config.ProjectName)

	outDir, err := os.Getwd()
	if err != nil {
		state.Logger.Error().Err(err).Msg("Failed to get current working directory")
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	finalPath := filepath.Join(outDir, projectName)
	state.Logger.Printf("Moving project from %s to %s\n", state.TempDirPath, finalPath)
	if err := utils.MoveDir(state.TempDirPath, finalPath); err != nil {
		state.Logger.Error().Err(err).Msg("Failed to move project to final directory")
		return fmt.Errorf("failed to move project to final directory: %w", err)
	}

	state.Logger.Debug().Msg("Project finalized successfully")
	return nil
}
