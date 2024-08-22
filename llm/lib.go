package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santiagomed/boil/fs"
)

type LlmConfig struct {
	APIKey    string
	ModelName string
	BatchID   string
	TellmURL  string
}

// GenerateProjectDetails generates detailed project information based on a description
func GenerateProjectDetails(client LlmClient, projectDesc string) (string, error) {
	prompt := getProjectDetailsPrompt(projectDesc)
	return client.GetCompletion(prompt, "text")
}

// GenerateFileTree generates a file tree structure based on project details
func GenerateFileTree(client LlmClient, projectDetails string) (string, error) {
	prompt := getFileTreePrompt(projectDetails)
	return client.GetCompletion(prompt, "text")
}

// DetermineFileOrder determines the order in which files should be created
func DetermineFileOrder(client LlmClient, fileTree string) ([]string, error) {
	prompt := getFileOrderPrompt(fileTree)
	response, err := client.GetCompletion(prompt, "json_object")
	if err != nil {
		return nil, fmt.Errorf("failed to determine file order: %w", err)
	}

	var fileOrder map[string][]string
	err = json.Unmarshal([]byte(response), &fileOrder)
	if err != nil {
		return nil, fmt.Errorf("error parsing file order: %w", err)
	}

	if len(fileOrder) == 0 {
		return nil, fmt.Errorf("no valid file paths found in the response")
	}

	return fileOrder["files"], nil
}

// GenerateFileOperations generates file operations for creating a specific file
func GenerateFileOperations(client LlmClient, projectDetails, fileTree string) ([]fs.FileOperation, error) {
	prompt := getFileOperationsPrompt(projectDetails, fileTree)
	response, err := client.GetCompletion(prompt, "json_object")
	if err != nil {
		return nil, fmt.Errorf("failed to generate file operations: %w", err)
	}

	var operations map[string][]fs.FileOperation
	err = json.Unmarshal([]byte(response), &operations)
	if err != nil {
		return nil, fmt.Errorf("error parsing file operations: %w", err)
	}

	if len(operations) == 0 {
		return nil, fmt.Errorf("no file operations generated")
	}

	return operations["operations"], nil
}

// GenerateFileContent generates content for a specific file
func GenerateFileContent(client LlmClient, fileName, projectDetails, fileTree string, previousFiles map[string]string) (string, error) {
	prompt := getFileContentPrompt(fileName, projectDetails, fileTree, previousFiles)
	var responseType string
	if strings.HasSuffix(fileName, ".json") {
		responseType = "json_object"
	} else {
		responseType = "text"
	}
	content, err := client.GetCompletion(prompt, responseType)
	if err != nil {
		return "", fmt.Errorf("failed to generate file content for %s: %w", fileName, err)
	}

	if content == "" {
		return "", fmt.Errorf("generated content for %s is empty", fileName)
	}

	return content, nil
}

// GenerateReadmeContent generates content for a README file
func GenerateReadmeContent(client LlmClient, projectDetails string) (string, error) {
	prompt := getReadmePrompt(projectDetails)
	return client.GetCompletion(prompt, "text")
}

// GenerateGitignoreContent generates gitignore content
func GenerateGitignoreContent(client LlmClient, projectDetails string) (string, error) {
	prompt := getGitignorePrompt(projectDetails)
	return client.GetCompletion(prompt, "text")
}

// GenerateDockerfileContent generates Dockerfile content
func GenerateDockerfileContent(client LlmClient, projectDetails string) (string, error) {
	prompt := getDockerfilePrompt(projectDetails)
	return client.GetCompletion(prompt, "text")
}
