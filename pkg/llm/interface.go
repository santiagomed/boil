package llm

import "github.com/santiagomed/boil/pkg/fs"

type LLMClient interface {
	GenerateProjectDetails(projectDesc string) (string, error)
	GenerateFileTree(projectDetails string) (string, error)
	GenerateFileOperations(projectDetails, fileTree string) ([]fs.FileOperation, error)
	DetermineFileOrder(fileTree string) ([]string, error)
	GenerateFileContent(fileName, projectDetails, fileTree string, previousFiles map[string]string) (string, error)
	GenerateReadmeContent(projectDetails string) (string, error)
	GenerateGitignoreContent(projectDetails string) (string, error)
	GenerateDockerfileContent(projectDetails string) (string, error)
}
