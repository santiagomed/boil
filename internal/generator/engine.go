package generator

import (
	"fmt"
	"path/filepath"

	"boil/internal/config"
	"boil/internal/llm"
	"boil/internal/tempdir"
	"boil/internal/utils"
)

// ProjectEngine handles the process of generating a project
type ProjectEngine struct {
	config *config.Config
	llm    *llm.Client
	tmpDir *tempdir.Manager
}

// NewProjectEngine creates a new ProjectEngine
func NewProjectEngine(cfg *config.Config, llmClient *llm.Client) *ProjectEngine {
	return &ProjectEngine{
		config: cfg,
		llm:    llmClient,
		tmpDir: tempdir.NewManager(cfg),
	}
}

// Generate performs the project generation process
func (pg *ProjectEngine) Generate(projectDesc string) (string, error) {
	// Create temporary directory
	tmpDirPath, err := pg.tmpDir.CreateTempDir("boil")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Generate project details
	projectDetails, err := pg.llm.GenerateProjectDetails(projectDesc)
	if err != nil {
		return "", fmt.Errorf("failed to generate project details: %w", err)
	}

	// Generate file tree
	fileTree, err := pg.llm.GenerateFileTree(projectDetails)
	if err != nil {
		return "", fmt.Errorf("failed to generate file tree: %w", err)
	}

	// Generate and execute file operations
	operations, err := pg.llm.GenerateFileOperations(projectDetails, fileTree)
	if err != nil {
		return "", fmt.Errorf("failed to generate file operations: %w", err)
	}
	err = utils.ExecuteFileOperations(tmpDirPath, operations)
	if err != nil {
		return "", fmt.Errorf("failed to execute file operations: %w", err)
	}

	// Determine file creation order
	fileOrder, err := pg.llm.DetermineFileOrder(fileTree)
	if err != nil {
		return "", fmt.Errorf("failed to determine file creation order: %w", err)
	}

	previousFiles := make(map[string]string)
	
	// Generate file content
	for _, file := range fileOrder {
		content, err := pg.llm.GenerateFileContent(file, projectDetails, fileTree, previousFiles)
		if err != nil {
			return "", fmt.Errorf("failed to generate content for file %s: %w", file, err)
		}
		err = utils.WriteFile(filepath.Join(tmpDirPath, file), content)
		if err != nil {
			return "", fmt.Errorf("failed to create file %s: %w", file, err)
		}
		previousFiles[file] = content
	}

	// Generate optional components based on config
	if pg.config.GitRepo {
		err = utils.InitializeGitRepo(tmpDirPath)
		if err != nil {
			return "", fmt.Errorf("failed to initialize git repository: %w", err)
		}
	}

	if pg.config.GitIgnore {
		err = utils.CreateGitIgnore(tmpDirPath)
		if err != nil {
			return "", fmt.Errorf("failed to create .gitignore file: %w", err)
		}
	}

	if pg.config.Readme {
		readme, err := pg.GenerateREADME(projectDetails)
		if err != nil {
			return "", fmt.Errorf("failed to generate README: %w", err)
		}
		err = utils.WriteFile(filepath.Join(tmpDirPath, "README.md"), readme)
		if err != nil {
			return "", fmt.Errorf("failed to create README file: %w", err)
		}
	}

	if pg.config.License {
		license, err := pg.GenerateLicense(pg.config.LicenseType)
		if err != nil {
			return "", fmt.Errorf("failed to generate LICENSE: %w", err)
		}
		err = utils.WriteFile(filepath.Join(tmpDirPath, "LICENSE"), license)
		if err != nil {
			return "", fmt.Errorf("failed to create LICENSE file: %w", err)
		}
	}

	if pg.config.Dockerfile {
		dockerfile, err := pg.GenerateDockerfile(projectDetails)
		if err != nil {
			return "", fmt.Errorf("failed to generate Dockerfile: %w", err)
		}
		err = utils.WriteFile(filepath.Join(tmpDirPath, "Dockerfile"), dockerfile)
		if err != nil {
			return "", fmt.Errorf("failed to create Dockerfile: %w", err)
		}
	}

	return tmpDirPath, nil
}

// FinalizeProject moves the generated project to the final output directory
func (pg *ProjectEngine) FinalizeProject(tmpDir, finalDir string) error {
	projectName := utils.FormatProjectName(filepath.Base(finalDir))
	finalPath := filepath.Join(pg.config.OutputDir, projectName)
	
	err := utils.MoveDir(tmpDir, finalPath)
	if err != nil {
		return fmt.Errorf("failed to move project to final directory: %w", err)
	}

	return nil
}

// CleanupTempDir removes the temporary directory
func (pg *ProjectEngine) CleanupTempDir() error {
	return pg.tmpDir.Cleanup()
}

// GenerateREADME generates a README.md file for the project
func (pg *ProjectEngine) GenerateREADME(projectDetails string) (string, error) {
	prompt := fmt.Sprintf("Generate a README.md file for the following project:\n\n%s\n\nInclude sections for project description, installation, usage, and any other relevant information.", projectDetails)
	return pg.llm.GenerateContent(prompt)
}

// GenerateLicense generates a LICENSE file for the project
func (pg *ProjectEngine) GenerateLicense(licenseType string) (string, error) {
	prompt := fmt.Sprintf("Generate the full text of a %s license.", licenseType)
	return pg.llm.GenerateContent(prompt)
}

// GenerateDockerfile generates a Dockerfile for the project
func (pg *ProjectEngine) GenerateDockerfile(projectDetails string) (string, error) {
	prompt := fmt.Sprintf("Generate a Dockerfile for the following project:\n\n%s\n\nEnsure it includes all necessary steps to build and run the application.", projectDetails)
	return pg.llm.GenerateContent(prompt)
}