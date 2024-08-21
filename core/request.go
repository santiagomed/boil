package core

import "os"

// Request indicates the user's request for a new project.
type Request struct {
	ProjectDescription string `mapstructure:"project_description"`
	ProjectName        string `mapstructure:"project_name"`
	GitRepo            bool   `mapstructure:"git_repo"`
	GitIgnore          bool   `mapstructure:"git_ignore"`
	Readme             bool   `mapstructure:"readme"`
	Dockerfile         bool   `mapstructure:"dockerfile"`

	OpenAIAPIKey string `mapstructure:"openai_api_key"`
	ModelName    string `mapstructure:"model_name"`
}

// DefaultRequest returns a Request with default values.
func DefaultRequest() *Request {
	return &Request{
		ProjectDescription: "Simple go 'Hello World' web app",
		ProjectName:        "my-project",
		OpenAIAPIKey:       os.Getenv("OPENAI_API_KEY"),
		ModelName:          "gpt-4o-mini",
		GitRepo:            false,
		GitIgnore:          false,
		Readme:             false,
		Dockerfile:         false,
	}
}
