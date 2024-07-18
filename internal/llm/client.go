package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/santiagomed/boil/internal/config"
	openai "github.com/sashabaranov/go-openai"
)

var client *openai.Client

func init() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = config.GetOpenAIAPIKey()
	}
	if apiKey == "" {
		fmt.Println("OpenAI API key not found. Please set the OPENAI_API_KEY environment variable or configure it in the config file.")
		os.Exit(1)
	}
	client = openai.NewClient(apiKey)
}

func GenerateProjectDetails(projectDesc string) (string, error) {
	prompt := getProjectDetailsPrompt(projectDesc)
	return getCompletion(prompt)
}

func GenerateFileTree(projectDetails string) (string, error) {
	prompt := getFileTreePrompt(projectDetails)
	return getCompletion(prompt)
}

func DetermineFileOrder(fileTree string) ([]string, error) {
	prompt := getFileOrderPrompt(fileTree)
	response, err := getCompletion(prompt)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(response, "\n")
	var fileOrder []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			parts := strings.SplitN(strings.TrimSpace(line), ".", 2)
			if len(parts) > 1 {
				fileOrder = append(fileOrder, strings.TrimSpace(parts[1]))
			}
		}
	}

	return fileOrder, nil
}

func GenerateFileContent(filePath, projectDetails, fileTree string) (string, error) {
	prompt := getFileContentPrompt(filePath, projectDetails, fileTree)
	return getCompletion(prompt)
}

func getCompletion(prompt string) (string, error) {
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("ChatCompletion error: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}

type FileOperation struct {
	Operation string `json:"operation"`
	Path      string `json:"path"`
	Content   string `json:"content,omitempty"`
}

func GenerateFileOperations(filePath, projectDetails, fileTree string) ([]FileOperation, error) {
	prompt := getFileOperationsPrompt(filePath, projectDetails, fileTree)
	response, err := getCompletion(prompt)
	if err != nil {
		return nil, err
	}

	var operations []FileOperation
	err = json.Unmarshal([]byte(response), &operations)
	if err != nil {
		return nil, fmt.Errorf("error parsing file operations: %v", err)
	}

	return operations, nil
}