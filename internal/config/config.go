package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config stores all configuration of the application.
type Config struct {
	ProjectName string `mapstructure:"project_name"`
	// OutputDir    string `mapstructure:"output_dir"`
	TempDir      string `mapstructure:"temp_dir"`
	OpenAIAPIKey string `mapstructure:"openai_api_key"`
	ModelName    string `mapstructure:"model_name"`
	GitRepo      bool   `mapstructure:"git_repo"`
	GitIgnore    bool   `mapstructure:"git_ignore"`
	Readme       bool   `mapstructure:"readme"`
	Dockerfile   bool   `mapstructure:"dockerfile"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	v.SetDefault("project_name", "my-project")
	v.SetDefault("temp_dir", os.TempDir())
	v.SetDefault("model_name", "gpt-4o-mini")
	v.SetDefault("git_repo", true)
	v.SetDefault("git_ignore", true)
	v.SetDefault("readme", true)
	v.SetDefault("license", false)
	v.SetDefault("license_type", "MIT")
	v.SetDefault("dockerfile", false)

	// Set config file name and path
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)
	v.AddConfigPath(".")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; ignore error if desired
		// fmt.Println("No config file found. Using defaults and environment variables.")
	}

	// Read from environment variables
	v.AutomaticEnv()

	// Set specific environment variable names
	v.BindEnv("openai_api_key", "OPENAI_API_KEY")
	v.BindEnv("output_dir", "BOIL_OUTPUT_DIR")
	v.BindEnv("temp_dir", "BOIL_TEMP_DIR")
	v.BindEnv("model_name", "BOILERPLATE_MODEL_NAME")

	var config Config
	err := v.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	return &config, nil
}
