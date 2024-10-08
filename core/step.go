package core

import (
	"fmt"

	"github.com/santiagomed/boil/fs"
	"github.com/santiagomed/boil/llm"
)

type GenerateProjectDetailsStep struct {
	llm llm.LlmClient
}

func (s *GenerateProjectDetailsStep) Execute(state *State) error {
	state.Logger.Info("Generating project details.")
	details, err := llm.GenerateProjectDetails(s.llm, state.Request.ProjectDescription)
	if err != nil {
		state.Logger.Error(fmt.Sprintf("Failed to generate project details: %v", err))
		return fmt.Errorf("failed to generate project details: %w", err)
	}
	state.ProjectDetails = details
	state.Logger.Info("Project details generated successfully")
	return nil
}

type GenerateFileTreeStep struct {
	llm llm.LlmClient
}

func (s *GenerateFileTreeStep) Execute(state *State) error {
	state.Logger.Info("Generating file tree.")
	fileTree, err := llm.GenerateFileTree(s.llm, state.ProjectDetails)
	if err != nil {
		state.Logger.Error(fmt.Sprintf("Failed to generate file tree: %v", err))
		return fmt.Errorf("failed to generate file tree: %w", err)
	}
	state.FileTree = fileTree
	state.Logger.Info("File tree generated successfully")
	return nil
}

type GenerateFileOperationsStep struct {
	llm llm.LlmClient
}

func (s *GenerateFileOperationsStep) Execute(state *State) error {
	state.Logger.Info("Generating file operations.")
	operations, err := llm.GenerateFileOperations(s.llm, state.ProjectDetails, state.FileTree)
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
	llm llm.LlmClient
}

func (s *DetermineFileOrderStep) Execute(state *State) error {
	state.Logger.Info("Determining file creation order.")
	order, err := llm.DetermineFileOrder(s.llm, state.FileTree)
	if err != nil {
		state.Logger.Error(fmt.Sprintf("Failed to determine file creation order: %v", err))
		return fmt.Errorf("failed to determine file creation order: %w", err)
	}
	state.FileOrder = order
	state.Logger.Info("File creation order determined successfully")
	return nil
}

type GenerateFileContentsStep struct {
	llm llm.LlmClient
	fs  *fs.FileSystem
}

func (s *GenerateFileContentsStep) Execute(state *State) error {
	state.Logger.Info("Generating file contents.")
	for _, file := range state.FileOrder {
		if s.fs.IsDir(file) {
			continue
		}
		state.Logger.Info(fmt.Sprintf("Generating content for file %s.", file))
		content, err := llm.GenerateFileContent(s.llm, file, state.ProjectDetails, state.FileTree, state.PreviousFiles)
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
	llm llm.LlmClient
	fs  *fs.FileSystem
}

func (s *CreateOptionalComponentsStep) Execute(state *State) error {
	state.Logger.Info("Creating optional components.")

	if state.Request.GitRepo {
		state.Logger.Info("Initializing Git repository.")
		if err := s.fs.InitializeGitRepo(); err != nil {
			state.Logger.Error(fmt.Sprintf("Failed to initialize Git repository: %v", err))
			return fmt.Errorf("failed to initialize git repository: %w", err)
		}
		state.Logger.Info("Git repository initialized successfully")
	}

	if state.Request.GitIgnore {
		state.Logger.Info("Creating .gitignore file.")
		gitignore, err := llm.GenerateGitignoreContent(s.llm, state.ProjectDetails)
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

	if state.Request.Readme {
		state.Logger.Info("Generating README.md.")
		readme, err := llm.GenerateReadmeContent(s.llm, state.ProjectDetails)
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

	if state.Request.Dockerfile {
		state.Logger.Info("Generating Dockerfile.")
		dockerfile, err := llm.GenerateDockerfileContent(s.llm, state.ProjectDetails)
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

type DoneStep struct{}

func (s *DoneStep) Execute(state *State) error {
	state.Logger.Info("Project finalized successfully")
	return nil
}

type StepManager interface {
	GetStep(stepType StepType) Step
	GetSteps() []StepType
}

type DefaultStepManager struct {
	steps   []StepType
	stepMap map[StepType]Step
}

func NewDefaultStepManager(llm llm.LlmClient, fs *fs.FileSystem) *DefaultStepManager {
	return &DefaultStepManager{
		stepMap: map[StepType]Step{
			GenerateProjectDetails:   &GenerateProjectDetailsStep{llm: llm},
			GenerateFileTree:         &GenerateFileTreeStep{llm: llm},
			GenerateFileOperations:   &GenerateFileOperationsStep{llm: llm},
			ExecuteFileOperations:    &ExecuteFileOperationsStep{fs: fs},
			DetermineFileOrder:       &DetermineFileOrderStep{llm: llm},
			GenerateFileContents:     &GenerateFileContentsStep{llm: llm, fs: fs},
			CreateOptionalComponents: &CreateOptionalComponentsStep{llm: llm, fs: fs},
			Done:                     &DoneStep{},
		},
		steps: []StepType{
			GenerateProjectDetails,
			GenerateFileTree,
			GenerateFileOperations,
			ExecuteFileOperations,
			DetermineFileOrder,
			GenerateFileContents,
			CreateOptionalComponents,
			Done,
		},
	}
}

func (sm *DefaultStepManager) GetStep(stepType StepType) Step {
	return sm.stepMap[stepType]
}

func (sm *DefaultStepManager) GetSteps() []StepType {
	return sm.steps
}
