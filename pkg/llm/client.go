package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/santiagomed/boil/pkg/fs"

	tellm "github.com/santiagomed/tellm/sdk"
	"github.com/sashabaranov/go-openai"
)

type LlmConfig struct {
	OpenAIAPIKey string
	ModelName    string
	ProjectName  string
}

// Client represents an LLM client
type Client struct {
	openAIClient *openai.Client
	config       *LlmConfig
	tellmClient  *tellm.Client
}

// NewClient creates a new LLM client
func NewClient(cfg *LlmConfig) (*Client, error) {
	if cfg.OpenAIAPIKey == "" {
		return nil, errors.New("OpenAI API key is required")
	}
	openAIClient := openai.NewClient(cfg.OpenAIAPIKey)
	tellmClient := tellm.NewClient("http://localhost:8080")
	return &Client{
		openAIClient: openAIClient,
		config:       cfg,
		tellmClient:  tellmClient,
	}, nil
}

// GenerateProjectDetails generates detailed project information based on a description
func (c *Client) GenerateProjectDetails(projectDesc string) (string, error) {
	prompt := getProjectDetailsPrompt(projectDesc)
	return c.getCompletion(prompt, "text")
}

// GenerateFileTree generates a file tree structure based on project details
func (c *Client) GenerateFileTree(projectDetails string) (string, error) {
	prompt := getFileTreePrompt(projectDetails)
	return c.getCompletion(prompt, "text")
}

// DetermineFileOrder determines the order in which files should be created
func (c *Client) DetermineFileOrder(fileTree string) ([]string, error) {
	prompt := getFileOrderPrompt(fileTree)
	response, err := c.getCompletion(prompt, "json_object")
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
func (c *Client) GenerateFileOperations(projectDetails, fileTree string) ([]fs.FileOperation, error) {
	prompt := getFileOperationsPrompt(projectDetails, fileTree)
	response, err := c.getCompletion(prompt, "json_object")
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
func (c *Client) GenerateFileContent(fileName, projectDetails, fileTree string, previousFiles map[string]string) (string, error) {
	prompt := getFileContentPrompt(fileName, projectDetails, fileTree, previousFiles)
	var responseType openai.ChatCompletionResponseFormatType
	if strings.HasSuffix(fileName, ".json") {
		responseType = "json_object"
	} else {
		responseType = "text"
	}
	content, err := c.getCompletion(prompt, responseType)
	if err != nil {
		return "", fmt.Errorf("failed to generate file content for %s: %w", fileName, err)
	}

	if content == "" {
		return "", fmt.Errorf("generated content for %s is empty", fileName)
	}

	return content, nil
}

// GenerateReadmeContent generates content for a README file
func (c *Client) GenerateReadmeContent(projectDetails string) (string, error) {
	prompt := getReadmePrompt(projectDetails)
	return c.getCompletion(prompt, "text")
}

// Generate gitignore content
func (c *Client) GenerateGitignoreContent(projectDetails string) (string, error) {
	prompt := getGitignorePrompt(projectDetails)
	return c.getCompletion(prompt, "text")
}

// Generate Dockerfile content
func (c *Client) GenerateDockerfileContent(projectDetails string) (string, error) {
	prompt := getDockerfilePrompt(projectDetails)
	return c.getCompletion(prompt, "text")
}

// getCompletion sends a request to the OpenAI API and returns the generated text
func (c *Client) getCompletion(prompt string, responseType openai.ChatCompletionResponseFormatType) (string, error) {
	resp, err := c.openAIClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: c.config.ModelName,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: getSystemPrompt(),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: responseType},
		},
	)

	e := &openai.APIError{}
	if errors.As(err, &e) {
		switch e.HTTPStatusCode {
		case 401:
			// unauthorized
			return "", fmt.Errorf("unauthorized: invalid OpenAI API key")
		case 429:
			// rate limiting or engine overload (wait and retry)
			return "", fmt.Errorf("rate limited by OpenAI API")
		case 500:
			// openai server error (retry)
			return "", fmt.Errorf("OpenAI server error")
		default:
			// unhandled
			return "", fmt.Errorf("OpenAI API error: %v", e)
		}
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from OpenAI")
	}

	res := resp.Choices[0].Message.Content
	c.tellmClient.Log(c.config.ProjectName, prompt, res)

	return res, nil
}
