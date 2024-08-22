package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/santiagomed/boil/logger"
	tellm "github.com/santiagomed/tellm/sdk"
)

var url = "https://api.anthropic.com/v1/messages"

type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	ID           string  `json:"id"`
	Model        string  `json:"model"`
	Role         string  `json:"role"`
	StopReason   string  `json:"stop_reason"`
	StopSequence *string `json:"stop_sequence"`
	Type         string  `json:"type"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type AnthropicErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type AnthropicRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicClient struct {
	config      *LlmConfig
	tellmClient *tellm.Client
	logger      logger.Logger
	httpClient  *http.Client
}

func NewAnthropicClient(cfg *LlmConfig, logger logger.Logger) (LlmClient, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("anthropic API key is required")
	}
	tellmClient := tellm.NewClient(cfg.TellmURL)
	return &AnthropicClient{
		config:      cfg,
		tellmClient: tellmClient,
		logger:      logger,
		httpClient:  &http.Client{},
	}, nil
}

func (a *AnthropicClient) GetCompletion(prompt, responseType string) (string, error) {
	req := AnthropicRequest{
		Model:     a.config.ModelName,
		MaxTokens: 2048,
		System:    getSystemPrompt(),
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	httpReq.Header.Set("x-api-key", a.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp AnthropicErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return "", fmt.Errorf("error unmarshaling error response: %v", err)
		}
		return "", fmt.Errorf("anthropic API error: %s - %s", errResp.Error.Type, errResp.Error.Message)
	}

	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("no content returned from Anthropic")
	}

	res := anthropicResp.Content[0].Text
	err = a.tellmClient.Log(a.config.BatchID, prompt, res, a.config.ModelName, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens)
	if err != nil {
		a.logger.WithField("warning", err).Warn("failed to log to tellm")
	}

	return res, nil
}
