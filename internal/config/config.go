package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config stores all configuration of the application.
type Config struct {
	OpenAIAPIKey string `mapstructure:"openai_api_key"`
	OutputDir    string `mapstructure:"output_dir"`
	TempDir      string `mapstructure:"temp_dir"`
	ModelName    string `mapstructure:"model_name"`
	GitRepo      bool   `mapstructure:"git_repo"`
	GitIgnore    bool   `mapstructure:"git_ignore"`
	Readme       bool   `mapstructure:"readme"`
	License      bool   `mapstructure:"license"`
	LicenseType  string `mapstructure:"license_type"`
	Dockerfile   bool   `mapstructure:"dockerfile"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	v.SetDefault("output_dir", ".")
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
	v.BindEnv("output_dir", "BOILERPLATE_OUTPUT_DIR")
	v.BindEnv("temp_dir", "BOILERPLATE_TEMP_DIR")
	v.BindEnv("model_name", "BOILERPLATE_MODEL_NAME")

	var config Config
	err := v.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	return &config, nil
}

var globalConfig *Config

// InitConfig initializes the global configuration
func InitConfig(configPath string) error {
	config, err := LoadConfig(configPath)
	if err != nil {
		return err
	}
	globalConfig = config
	return nil
}

// GetConfig returns the global configuration
func GetConfig() *Config {
	if globalConfig == nil {
		panic("Config not initialized. Call InitConfig() before using GetConfig().")
	}
	return globalConfig
}

// GetOpenAIAPIKey returns the OpenAI API key
func GetOpenAIAPIKey() string {
	return GetConfig().OpenAIAPIKey
}

// GetOutputDir returns the output directory
func GetOutputDir() string {
	return GetConfig().OutputDir
}

// GetTempDir returns the temporary directory
func GetTempDir() string {
	return GetConfig().TempDir
}

// GetModelName returns the model name
func GetModelName() string {
	return GetConfig().ModelName
}

// CreateDefaultConfig creates a default configuration file
func CreateDefaultConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".boilerplate")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("unable to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	defaultConfig := `# Boilerplate CLI Configuration

# OpenAI API Key (required)
# openai_api_key: "your-api-key-here"

# Output directory for generated projects (optional, default: current directory)
# output_dir: "."

# Temporary directory for project generation (optional, default: system temp directory)
# temp_dir: "/tmp"

# Model name to use for generation (optional, default: gpt-3.5-turbo)
# model_name: "gpt-3.5-turbo"
`

	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("unable to write default config file: %w", err)
	}

	fmt.Printf("Default configuration file created at: %s\n", configPath)
	return nil
}
