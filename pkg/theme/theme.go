package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Theme struct {
	System  string            `yaml:"system"`
	Name    string            `yaml:"name"`
	Author  string            `yaml:"author"`
	Variant string            `yaml:"variant"`
	Palette map[string]string `yaml:"palette"`
}

type ThemeManager struct {
	themesPath string
	themes     map[string]*Theme
}

func NewThemeManager(themesPath string) *ThemeManager {
	return &ThemeManager{
		themesPath: themesPath,
		themes:     make(map[string]*Theme),
	}
}

func (tm *ThemeManager) LoadThemes() error {
	base16Path := filepath.Join(tm.themesPath, "base16")
	base24Path := filepath.Join(tm.themesPath, "base24")

	if err := tm.loadThemesFromDir(base16Path); err != nil {
		return fmt.Errorf("failed to load base16 themes: %w", err)
	}

	if _, err := os.Stat(base24Path); err == nil {
		if err := tm.loadThemesFromDir(base24Path); err != nil {
			return fmt.Errorf("failed to load base24 themes: %w", err)
		}
	}

	return nil
}

func (tm *ThemeManager) loadThemesFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		themePath := filepath.Join(dir, entry.Name())
		theme, err := tm.loadTheme(themePath)
		if err != nil {
			fmt.Printf("Warning: failed to load theme %s: %v\n", entry.Name(), err)
			continue
		}

		themeName := strings.TrimSuffix(entry.Name(), ".yaml")
		tm.themes[themeName] = theme
	}

	return nil
}

func (tm *ThemeManager) loadTheme(path string) (*Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var theme Theme
	if err := yaml.Unmarshal(data, &theme); err != nil {
		return nil, err
	}

	if err := tm.validateTheme(&theme); err != nil {
		return nil, err
	}

	return &theme, nil
}

func (tm *ThemeManager) validateTheme(theme *Theme) error {
	if theme.System == "" {
		return fmt.Errorf("theme missing system field")
	}

	if theme.System != "base16" && theme.System != "base24" {
		return fmt.Errorf("unsupported theme system: %s", theme.System)
	}

	expectedColors := 16
	if theme.System == "base24" {
		expectedColors = 24
	}

	if len(theme.Palette) < expectedColors {
		return fmt.Errorf("theme has %d colors, expected at least %d for %s",
			len(theme.Palette), expectedColors, theme.System)
	}

	requiredColors := []string{
		"base00", "base01", "base02", "base03", "base04", "base05", "base06", "base07",
		"base08", "base09", "base0A", "base0B", "base0C", "base0D", "base0E", "base0F",
	}

	if theme.System == "base24" {
		for i := 16; i < 24; i++ {
			requiredColors = append(requiredColors, fmt.Sprintf("base%02X", i))
		}
	}

	for _, color := range requiredColors {
		if _, exists := theme.Palette[color]; !exists {
			return fmt.Errorf("theme missing required color: %s", color)
		}
	}

	return nil
}

func (tm *ThemeManager) GetTheme(name string) (*Theme, error) {
	// First try the exact name
	if theme, exists := tm.themes[name]; exists {
		return theme, nil
	}

	// If not found, try removing base16- or base24- prefix
	if strings.HasPrefix(name, "base16-") {
		shortName := strings.TrimPrefix(name, "base16-")
		if theme, exists := tm.themes[shortName]; exists {
			return theme, nil
		}
	} else if strings.HasPrefix(name, "base24-") {
		shortName := strings.TrimPrefix(name, "base24-")
		if theme, exists := tm.themes[shortName]; exists {
			return theme, nil
		}
	}

	return nil, fmt.Errorf("theme not found: %s", name)
}

func (tm *ThemeManager) ListThemes() []string {
	var names []string
	for name := range tm.themes {
		names = append(names, name)
	}
	return names
}

func (tm *ThemeManager) GetThemeInfo(name string) (*Theme, error) {
	return tm.GetTheme(name)
}

func (tm *ThemeManager) SaveTheme(theme *Theme) error {
	// Determine the directory based on the theme system
	var themeDir string
	if theme.System == "base24" {
		themeDir = filepath.Join(tm.themesPath, "base24")
	} else {
		themeDir = filepath.Join(tm.themesPath, "base16")
	}

	// Ensure the directory exists
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		return fmt.Errorf("failed to create theme directory: %w", err)
	}

	// Create properly formatted YAML content
	data, err := tm.formatThemeYAML(theme)
	if err != nil {
		return fmt.Errorf("failed to format theme: %w", err)
	}

	// Write to file
	filename := fmt.Sprintf("%s.yaml", theme.Name)
	themePath := filepath.Join(themeDir, filename)

	if err := os.WriteFile(themePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write theme file: %w", err)
	}

	// Add to in-memory themes map
	tm.themes[theme.Name] = theme

	return nil
}

func (tm *ThemeManager) formatThemeYAML(theme *Theme) ([]byte, error) {
	var result strings.Builder

	// Write header fields with proper quotes
	result.WriteString(fmt.Sprintf("system: \"%s\"\n", theme.System))
	result.WriteString(fmt.Sprintf("name: \"%s\"\n", theme.Name))
	result.WriteString(fmt.Sprintf("author: \"%s\"\n", theme.Author))
	result.WriteString(fmt.Sprintf("variant: \"%s\"\n", theme.Variant))
	result.WriteString("palette:\n")

	// Write palette colors in correct order with proper indentation
	baseOrder := []string{
		"base00", "base01", "base02", "base03", "base04", "base05", "base06", "base07",
		"base08", "base09", "base0A", "base0B", "base0C", "base0D", "base0E", "base0F",
	}

	for _, base := range baseOrder {
		if color, exists := theme.Palette[base]; exists {
			result.WriteString(fmt.Sprintf("  %s: \"%s\"\n", base, color))
		}
	}

	return []byte(result.String()), nil
}
