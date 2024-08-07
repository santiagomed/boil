package core

import (
	"fmt"
	"path/filepath"

	"github.com/santiagomed/boil/pkg/fs"
	"github.com/santiagomed/boil/pkg/llm"
	"github.com/santiagomed/boil/pkg/utils"
)

type StepManager struct {
	steps   []StepType
	stepMap map[StepType]Step
}

func NewStepManager(llm llm.LLMClient, fs *fs.FileSystem) *StepManager {
	return &StepManager{
		stepMap: map[StepType]Step{
			GenerateProjectDetails:   &GenerateProjectDetailsStep{llm: llm},
			GenerateFileTree:         &GenerateFileTreeStep{llm: llm},
			GenerateFileOperations:   &GenerateFileOperationsStep{llm: llm},
			ExecuteFileOperations:    &ExecuteFileOperationsStep{fs: fs},
			DetermineFileOrder:       &DetermineFileOrderStep{llm: llm},
			GenerateFileContents:     &GenerateFileContentsStep{llm: llm, fs: fs},
			CreateOptionalComponents: &CreateOptionalComponentsStep{llm: llm, fs: fs},
			FinalizeProject:          &FinalizeProjectStep{fs: fs},
		},
		steps: []StepType{
			GenerateProjectDetails,
			GenerateFileTree,
			GenerateFileOperations,
			ExecuteFileOperations,
			DetermineFileOrder,
			GenerateFileContents,
			CreateOptionalComponents,
			FinalizeProject,
		},
	}
}

func (sm *StepManager) GetStep(stepType StepType) Step {
	return sm.stepMap[stepType]
}

type GenerateProjectDetailsStep struct {
	llm llm.LLMClient
}

func (s *GenerateProjectDetailsStep) Execute(state *State) error {
	state.Logger.Info("Generating project details.")
	details, err := s.llm.GenerateProjectDetails(state.ProjectDesc)
	if err != nil {
		state.Logger.Error(fmt.Sprintf("Failed to generate project details: %v", err))
		return fmt.Errorf("failed to generate project details: %w", err)
	}
	state.ProjectDetails = details
	state.Logger.Info("Project details generated successfully")
	return nil
}

type GenerateFileTreeStep struct {
	llm llm.LLMClient
}

func (s *GenerateFileTreeStep) Execute(state *State) error {
	state.Logger.Info("Generating file tree.")
	fileTree, err := s.llm.GenerateFileTree(state.ProjectDetails)
	if err != nil {
		state.Logger.Error(fmt.Sprintf("Failed to generate file tree: %v", err))
		return fmt.Errorf("failed to generate file tree: %w", err)
	}
	state.FileTree = fileTree
	state.Logger.Info("File tree generated successfully")
	return nil
}

type GenerateFileOperationsStep struct {
	llm llm.LLMClient
}

func (s *GenerateFileOperationsStep) Execute(state *State) error {
	state.Logger.Info("Generating file operations.")
	operations, err := s.llm.GenerateFileOperations(state.ProjectDetails, state.FileTree)
	if err != nil {
		state.Logger.Error(fmt.Sprintf("Failed to generate file operations: %v", err))
		return fmt.Errorf("failed to generate file operations: %w", err)
	}
	state.FileOperations = operations
	state.Logger.Info("File operations generated successfully")
	return nil
}

type ExecuteFileOperationsStep struct {
	fs *fs.FileSystem
}

func (s *ExecuteFileOperationsStep) Execute(state *State) error {
	state.Logger.Info("Executing file operations.")
	err := s.fs.ExecuteFileOperations(state.FileOperations)
	if err != nil {
		state.Logger.Error(fmt.Sprintf("Failed to execute file operations: %v", err))
		return fmt.Errorf("failed to execute file operations: %w", err)
	}
	state.Logger.Info("File operations executed successfully")
	return nil
}

type DetermineFileOrderStep struct {
	llm llm.LLMClient
}

func (s *DetermineFileOrderStep) Execute(state *State) error {
	state.Logger.Info("Determining file creation order.")
	order, err := s.llm.DetermineFileOrder(state.FileTree)
	if err != nil {
		state.Logger.Error(fmt.Sprintf("Failed to determine file creation order: %v", err))
		return fmt.Errorf("failed to determine file creation order: %w", err)
	}
	state.FileOrder = order
	state.Logger.Info("File creation order determined successfully")
	return nil
}

type GenerateFileContentsStep struct {
	llm llm.LLMClient
	fs  *fs.FileSystem
}

func (s *GenerateFileContentsStep) Execute(state *State) error {
	state.Logger.Info("Generating file contents.")
	for _, file := range state.FileOrder {
		if s.fs.IsDir(file) {
			continue
		}
		state.Logger.Info(fmt.Sprintf("Generating content for file %s.", file))
		content, err := s.llm.GenerateFileContent(file, state.ProjectDetails, state.FileTree, state.PreviousFiles)
		if err != nil {
			state.Logger.Error(fmt.Sprintf("Failed to generate content for file %s: %v", file, err))
			return fmt.Errorf("failed to generate content for file %s: %w", file, err)
		}
		err = s.fs.WriteFile(file, content)
		if err != nil {
			state.Logger.Error(fmt.Sprintf("Failed to create file %s: %v", file, err))
			return fmt.Errorf("failed to create file %s: %w", file, err)
		}
		state.PreviousFiles[file] = content
		state.Logger.Info(fmt.Sprintf("Content generated for file %s", file))
	}
	state.Logger.Info("All file contents generated successfully")
	return nil
}

type CreateOptionalComponentsStep struct {
	llm llm.LLMClient
	fs  *fs.FileSystem
}

func (s *CreateOptionalComponentsStep) Execute(state *State) error {
	state.Logger.Info("Creating optional components.")

	if state.Config.GitRepo {
		state.Logger.Info("Initializing Git repository.")
		if err := s.fs.InitializeGitRepo(); err != nil {
			state.Logger.Error(fmt.Sprintf("Failed to initialize Git repository: %v", err))
			return fmt.Errorf("failed to initialize git repository: %w", err)
		}
		state.Logger.Info("Git repository initialized successfully")
	}

	if state.Config.GitIgnore {
		state.Logger.Info("Creating .gitignore file.")
		gitignore, err := s.llm.GenerateGitignoreContent(state.ProjectDetails)
		if err != nil {
			state.Logger.Error(fmt.Sprintf("Failed to create .gitignore file: %v", err))
			return fmt.Errorf("failed to create .gitignore file: %w", err)
		}
		if err := s.fs.WriteFile(".gitignore", gitignore); err != nil {
			state.Logger.Error(fmt.Sprintf("Failed to create .gitignore file: %v", err))
			return fmt.Errorf("failed to create .gitignore file: %w", err)
		}
		state.Logger.Info(".gitignore file created successfully")
	}

	if state.Config.Readme {
		state.Logger.Info("Generating README.md.")
		readme, err := s.llm.GenerateReadmeContent(state.ProjectDetails)
		if err != nil {
			state.Logger.Error(fmt.Sprintf("Failed to generate README: %v", err))
			return fmt.Errorf("failed to generate README: %w", err)
		}
		if err := s.fs.WriteFile("README.md", readme); err != nil {
			state.Logger.Error(fmt.Sprintf("Failed to create README file: %v", err))
			return fmt.Errorf("failed to create README file: %w", err)
		}
		state.Logger.Info("README.md created successfully")
	}

	if state.Config.Dockerfile {
		state.Logger.Info("Generating Dockerfile.")
		dockerfile, err := s.llm.GenerateDockerfileContent(state.ProjectDetails)
		if err != nil {
			state.Logger.Error(fmt.Sprintf("Failed to generate Dockerfile: %v", err))
			return fmt.Errorf("failed to generate Dockerfile: %w", err)
		}
		if err := s.fs.WriteFile("Dockerfile", dockerfile); err != nil {
			state.Logger.Error(fmt.Sprintf("Failed to create Dockerfile: %v", err))
			return fmt.Errorf("failed to create Dockerfile: %w", err)
		}
		state.Logger.Info("Dockerfile created successfully")
	}

	state.Logger.Info("Optional components created successfully")
	return nil
}

type FinalizeProjectStep struct {
	fs *fs.FileSystem
}

func (s *FinalizeProjectStep) Execute(state *State) error {
	state.Logger.Info("Finalizing project.")
	projectName := utils.FormatProjectName(state.Config.ProjectName)

	zipPath := filepath.Join(".", projectName+".zip")
	state.Logger.Info(fmt.Sprintf("Writing project to zip file: %s", zipPath))
	if err := s.fs.WriteToZip(zipPath); err != nil {
		state.Logger.Error(fmt.Sprintf("Failed to write project to zip file: %v", err))
		return fmt.Errorf("failed to write project to zip file: %w", err)
	}

	state.Logger.Info("Project finalized successfully")
	return nil
}
