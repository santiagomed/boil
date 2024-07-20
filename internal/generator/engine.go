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
}

// NewProjectEngine creates a new ProjectEngine
func NewProjectEngine(cfg *config.Config, llmClient *llm.Client) *ProjectEngine {
	return &ProjectEngine{
		config: cfg,
		llm:    llmClient,
	}
}

// Generate performs the project generation process
func (pg *ProjectEngine) Generate(projectDesc string) error {
	// Create temporary directory
	tmpDir := tempdir.NewManager(pg.config)
	tmpDirPath, err := tmpDir.CreateTempDir("boil")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer tmpDir.Cleanup()

	// Generate project details
	projectDetails, err := pg.llm.GenerateProjectDetails(projectDesc)
	if err != nil {
		return fmt.Errorf("failed to generate project details: %w", err)
	}

	// Generate file tree
	fileTree, err := pg.llm.GenerateFileTree(projectDetails)
	if err != nil {
		return fmt.Errorf("failed to generate file tree: %w", err)
	}

	// Determine file creation order
	fileOrder, err := pg.llm.DetermineFileOrder(fileTree)
	if err != nil {
		return fmt.Errorf("failed to determine file creation order: %w", err)
	}

	// Generate and execute file operations
	for _, filePath := range fileOrder {
		operations, err := pg.llm.GenerateFileOperations(filePath, projectDetails, fileTree)
		if err != nil {
			return fmt.Errorf("failed to generate file operations for %s: %w", filePath, err)
		}

		err = utils.ExecuteFileOperations(tmpDirPath, operations)
		if err != nil {
			return fmt.Errorf("failed to execute file operations for %s: %w", filePath, err)
		}
	}

	// Initialize git repository
	_, err = utils.ExecuteCmd(tmpDirPath, "git", "init")
	if err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Create .gitignore file
	err = utils.CreateGitIgnore(tmpDirPath)
	if err != nil {
		return fmt.Errorf("failed to create .gitignore file: %w", err)
	}

	// Allow user to review the generated project
	fmt.Printf("Project generated in temporary directory: %s\n", tmpDirPath)
	fmt.Print("Review the project and enter 'y' to finalize, or any other key to abort: ")
	var response string
	fmt.Scanln(&response)

	if response == "y" {
		// Finalize project
		projectName := utils.FormatProjectName(filepath.Base(projectDesc))
		finalDir := filepath.Join(pg.config.OutputDir, projectName)
		err = FinalizeProject(tmpDirPath, finalDir)
		if err != nil {
			return fmt.Errorf("failed to finalize project: %w", err)
		}
		fmt.Printf("Project successfully created in: %s\n", finalDir)
	} else {
		fmt.Println("Project creation aborted. Temporary files will be cleaned up.")
	}

	return nil
}

// FinalizeProject moves the generated project to the final output directory
func FinalizeProject(tmpDir, finalDir string) error {
	// Move the generated project to the final output directory
	err := utils.MoveDir(tmpDir, finalDir)
	if err != nil {
		return fmt.Errorf("failed to move project to final directory: %w", err)
	}

	return nil
}

// GenerateREADME generates a README.md file for the project
func (pg *ProjectEngine) GenerateREADME(projectDetails string) (string, error) {
	prompt := fmt.Sprintf("Generate a README.md file for the following project:\n\n%s\n\nInclude sections for project description, installation, usage, and any other relevant information.", projectDetails)
	
	readme, err := pg.llm.GenerateContent(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate README: %w", err)
	}

	return readme, nil
}

// GenerateLicense generates a LICENSE file for the project
func (pg *ProjectEngine) GenerateLicense(licenseType string) (string, error) {
	prompt := fmt.Sprintf("Generate the full text of a %s license.", licenseType)
	
	license, err := pg.llm.GenerateContent(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate LICENSE: %w", err)
	}

	return license, nil
}

// GenerateDockerfile generates a Dockerfile for the project
func (pg *ProjectEngine) GenerateDockerfile(projectDetails string) (string, error) {
	prompt := fmt.Sprintf("Generate a Dockerfile for the following project:\n\n%s\n\nEnsure it includes all necessary steps to build and run the application.", projectDetails)
	
	dockerfile, err := pg.llm.GenerateContent(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	return dockerfile, nil
}