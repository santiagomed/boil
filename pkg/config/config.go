package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config stores all configuration of the application.
type Config struct {
	ProjectName  string `mapstructure:"project_name"`
	OpenAIAPIKey string `mapstructure:"openai_api_key"`
	ModelName    string `mapstructure:"model_name"`
	GitRepo      bool   `mapstructure:"git_repo"`
	GitIgnore    bool   `mapstructure:"git_ignore"`
	Readme       bool   `mapstructure:"readme"`
	Dockerfile   bool   `mapstructure:"dockerfile"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		ProjectName:  "my-project",
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
		ModelName:    "gpt-4o-mini",
		GitRepo:      false,
		GitIgnore:    false,
		Readme:       false,
		Dockerfile:   false,
	}
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)
	v.AddConfigPath(".")
	v.AddConfigPath(filepath.Join(os.Getenv("HOME"), ".boil"))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; ignore error if desired
	}

	// Environment variables
	v.SetEnvPrefix("BOIL")
	v.AutomaticEnv()
	v.BindEnv("openai_api_key", "OPENAI_API_KEY")

	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	// Validate config
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

func validateConfig(config *Config) error {
	if config.OpenAIAPIKey == "" {
		return fmt.Errorf("OpenAI API key is required")
	}

	return nil
}
