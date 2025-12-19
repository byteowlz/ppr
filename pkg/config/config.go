package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	ThemesPath         string   `toml:"themes_path"`
	TemplatesPath      string   `toml:"templates_path"`
	OutputPath         string   `toml:"output_path"`
	DefaultTheme       string   `toml:"default_theme"`
	DefaultTemplate    string   `toml:"default_template"`
	DefaultWidth       int      `toml:"default_width"`
	DefaultHeight      int      `toml:"default_height"`
	AutoSetWallpaper   bool     `toml:"auto_set_wallpaper"`
	CurrentTheme       string   `toml:"current_theme"`
	CurrentTemplate    string   `toml:"current_template"`
	LastOutputPath     string   `toml:"last_output_path"`
	PreferredTemplates []string `toml:"preferred_templates"`
	OnWallpaperSet     string   `toml:"on_wallpaper_set"`
}

func DefaultConfig() *Config {
	return &Config{
		ThemesPath:         "~/.config/ppr/themes",
		TemplatesPath:      "~/.config/ppr/templates",
		OutputPath:         "~/Pictures/ppr",
		DefaultTheme:       "nord",
		DefaultTemplate:    "geometric-simple.svg",
		DefaultWidth:       1920,
		DefaultHeight:      1080,
		AutoSetWallpaper:   false,
		CurrentTheme:       "",
		CurrentTemplate:    "",
		LastOutputPath:     "",
		PreferredTemplates: []string{"all"},
		OnWallpaperSet:     "",
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

	// Create a copy to modify paths for saving
	configCopy := *c
	homeDir, _ := os.UserHomeDir()

	// Replace homeDir with ~ in paths
	if strings.HasPrefix(configCopy.ThemesPath, homeDir) {
		configCopy.ThemesPath = "~" + configCopy.ThemesPath[len(homeDir):]
	}
	if strings.HasPrefix(configCopy.TemplatesPath, homeDir) {
		configCopy.TemplatesPath = "~" + configCopy.TemplatesPath[len(homeDir):]
	}
	if strings.HasPrefix(configCopy.OutputPath, homeDir) {
		configCopy.OutputPath = "~" + configCopy.OutputPath[len(homeDir):]
	}
	if strings.HasPrefix(configCopy.LastOutputPath, homeDir) {
		configCopy.LastOutputPath = "~" + configCopy.LastOutputPath[len(homeDir):]
	}

	configPath := GetConfigPath()
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(&configCopy); err != nil {
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
	if strings.HasPrefix(path, "~") {
		homeDir, _ := os.UserHomeDir()
		if len(path) == 1 {
			return homeDir
		} else if path[1] == '/' {
			return filepath.Join(homeDir, path[2:])
		} else {
			// ~ followed by other characters, treat as ~/
			return filepath.Join(homeDir, path[1:])
		}
	}
	return path
}

// ExecuteOnWallpaperSet runs the configured script when wallpaper is set
func (c *Config) ExecuteOnWallpaperSet() error {
	if c.OnWallpaperSet == "" {
		return nil
	}

	fmt.Printf("Executing custom script: %s\n", c.OnWallpaperSet)

	cmd := exec.Command("/bin/bash", "-c", c.OnWallpaperSet)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if cmd.ProcessState != nil {
			fmt.Printf("Script execution failed with exit code: %d\n", cmd.ProcessState.ExitCode())
		}
		return fmt.Errorf("failed to execute custom script: %w", err)
	}

	fmt.Printf("Custom script executed successfully\n")
	return nil
}
