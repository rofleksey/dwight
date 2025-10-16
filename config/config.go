package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	BaseURL         string `mapstructure:"base_url" validate:"required"`
	Token           string `mapstructure:"token" validate:"required"`
	Model           string `mapstructure:"model" validate:"required"`
	SnippetMaxLines int    `mapstructure:"snippet_max_lines" validate:"required,min=1"`
}

func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	viper.SetConfigName(".dwight.conf")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(home)

	viper.SetDefault("snippet_max_lines", 50)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}
