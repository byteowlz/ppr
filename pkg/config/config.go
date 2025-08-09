package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	ThemesPath       string `toml:"themes_path"`
	TemplatesPath    string `toml:"templates_path"`
	OutputPath       string `toml:"output_path"`
	DefaultTheme     string `toml:"default_theme"`
	DefaultTemplate  string `toml:"default_template"`
	DefaultWidth     int    `toml:"default_width"`
	DefaultHeight    int    `toml:"default_height"`
	AutoSetWallpaper bool   `toml:"auto_set_wallpaper"`
}

func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		ThemesPath:       filepath.Join(homeDir, ".config", "ppr", "themes"),
		TemplatesPath:    filepath.Join(homeDir, ".config", "ppr", "templates"),
		OutputPath:       filepath.Join(homeDir, "Pictures", "ppr"),
		DefaultTheme:     "nord",
		DefaultTemplate:  "geometric-simple.svg",
		DefaultWidth:     1920,
		DefaultHeight:    1080,
		AutoSetWallpaper: false,
	}
}

func GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "ppr", "config.toml")
}

func GetConfigDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "ppr")
}

func Load() (*Config, error) {
	configPath := GetConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// Start with default config and override with file values
	config := *DefaultConfig()
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	// Expand tilde in paths
	config.ThemesPath = expandPath(config.ThemesPath)
	config.TemplatesPath = expandPath(config.TemplatesPath)
	config.OutputPath = expandPath(config.OutputPath)

	return &config, nil
}

func (c *Config) Save() error {
	configDir := GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := GetConfigPath()
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

func (c *Config) EnsureDirectories() error {
	dirs := []string{c.ThemesPath, c.TemplatesPath, c.OutputPath}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
