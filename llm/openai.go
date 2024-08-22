package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/santiagomed/boil/logger"
	tellm "github.com/santiagomed/tellm/sdk"
	"github.com/sashabaranov/go-openai"
)

// Client represents an LLM client implementation
type OpenAIClient struct {
	openAIClient *openai.Client
	config       *LlmConfig
	tellmClient  *tellm.Client
	logger       logger.Logger
}

// NewClient creates a new LLM client
func NewOpenAIClient(cfg *LlmConfig, logger logger.Logger) (LlmClient, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("OpenAI API key is required")
	}
	openAIClient := openai.NewClient(cfg.APIKey)
	tellmClient := tellm.NewClient(cfg.TellmURL)
	return &OpenAIClient{
		openAIClient: openAIClient,
		config:       cfg,
		tellmClient:  tellmClient,
		logger:       logger,
	}, nil
}

// getCompletion sends a request to the OpenAI API and returns the generated text
func (c *OpenAIClient) GetCompletion(prompt, responseType string) (string, error) {
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
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatType(responseType)},
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
	usage := resp.Usage
	res := resp.Choices[0].Message.Content
	err = c.tellmClient.Log(c.config.BatchID, prompt, res, c.config.ModelName, usage.PromptTokens, usage.CompletionTokens)
	if err != nil {
		c.logger.WithField("warning", err).Warn("failed to log to tellm")
	}

	return res, nil
}
