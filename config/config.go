package config

import (
	"fmt"
	
	"github.com/spf13/viper"
	"github.com/santiagomed/boil/core"
)

func LoadConfig(configPath string) (*core.Request, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Start with default values
	req := core.DefaultRequest()

	// Unmarshal config into the request struct
	if err := v.Unmarshal(req); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Use NewRequest to create the final request object
	return core.NewRequest(
		req.ProjectDescription,
		req.ProjectName,
		req.APIKey,
		req.ModelName,
		req.GitRepo,
		req.GitIgnore,
		req.Readme,
		req.Dockerfile,
	), nil
}