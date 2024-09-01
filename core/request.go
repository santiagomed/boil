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

	APIKey    string `mapstructure:"openai_api_key"`
	ModelName string `mapstructure:"model_name"`
}

// DefaultRequest returns a Request with default values.
func DefaultRequest() *Request {
	return &Request{
		ProjectDescription: "Simple go 'Hello World' web app",
		ProjectName:        "my-project",
		APIKey:             os.Getenv("OPENAI_API_KEY"),
		ModelName:          "gpt-4o-mini",
		GitRepo:            false,
		GitIgnore:          false,
		Readme:             false,
		Dockerfile:         false,
	}
}

func NewRequest(projectDescription, projectName, apiKey, modelName string, gitRepo, gitIgnore, readme, dockerfile bool) *Request {
	return &Request{
		ProjectDescription: projectDescription,
		ProjectName:        projectName,
		APIKey:             apiKey,
		ModelName:          modelName,
		GitRepo:            gitRepo,
		GitIgnore:          gitIgnore,
		Readme:             readme,
		Dockerfile:         dockerfile,
	}
}
