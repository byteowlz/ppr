package cmd

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/byteowlz/ppr/pkg/config"
	"github.com/byteowlz/ppr/pkg/theme"
	"github.com/spf13/cobra"
)

var extractColorsCmd = &cobra.Command{
	Use:   "extract-colors <svg-file> <theme-name>",
	Short: "Extract color scheme from SVG file and create a new theme",
	Long: `Extract base16 color scheme from an SVG file that contains 16 color swatches.
The SVG should contain exactly 16 unique fill colors, which will be mapped to base00-base0F in order of appearance.
This creates a new theme file that can be used with other templates.`,
	Args: cobra.ExactArgs(2),
	RunE: runExtractColors,
}

func init() {
	rootCmd.AddCommand(extractColorsCmd)
}

func runExtractColors(cmd *cobra.Command, args []string) error {
	svgFile := args[0]
	themeName := args[1]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	// Read SVG file
	content, err := ioutil.ReadFile(svgFile)
	if err != nil {
		return fmt.Errorf("failed to read SVG file: %w", err)
	}

	// Extract colors from SVG
	colors, err := extractColorsFromSVG(content)
	if err != nil {
		return fmt.Errorf("failed to extract colors: %w", err)
	}

	if len(colors) != 16 {
		return fmt.Errorf("expected 16 base colors, found %d", len(colors))
	}

	// Create theme
	newTheme := &theme.Theme{
		System:  "base16",
		Name:    themeName,
		Author:  "extracted",
		Variant: "dark",
		Palette: colors,
	}

	// Save theme
	themeManager := theme.NewThemeManager(cfg.ThemesPath)
	if err := themeManager.SaveTheme(newTheme); err != nil {
		return fmt.Errorf("failed to save theme: %w", err)
	}

	fmt.Printf("Successfully extracted colors and created theme '%s'\n", themeName)
	fmt.Printf("Theme saved to: %s/base16/%s.yaml\n", cfg.ThemesPath, themeName)

	// Print extracted colors for verification
	fmt.Println("\nExtracted colors:")
	baseOrder := []string{"base00", "base01", "base02", "base03", "base04", "base05", "base06", "base07",
		"base08", "base09", "base0A", "base0B", "base0C", "base0D", "base0E", "base0F"}

	for _, base := range baseOrder {
		if color, exists := colors[base]; exists {
			fmt.Printf("  %s: %s\n", base, color)
		}
	}

	return nil
}

func extractColorsFromSVG(content []byte) (map[string]string, error) {
	svgContent := string(content)
	var colors []string
	seen := make(map[string]bool)

	// Extract colors from direct fill attributes
	fillRegex := regexp.MustCompile(`fill="(#[0-9A-Fa-f]{6})"`)
	fillMatches := fillRegex.FindAllStringSubmatch(svgContent, -1)

	for _, match := range fillMatches {
		if len(match) > 1 {
			color := strings.ToUpper(match[1])
			if !seen[color] {
				colors = append(colors, color)
				seen[color] = true
			}
		}
	}

	// Also extract colors from CSS styles
	cssRegex := regexp.MustCompile(`fill:\s*(#[0-9A-Fa-f]{6})`)
	cssMatches := cssRegex.FindAllStringSubmatch(svgContent, -1)

	for _, match := range cssMatches {
		if len(match) > 1 {
			color := strings.ToUpper(match[1])
			if !seen[color] {
				colors = append(colors, color)
				seen[color] = true
			}
		}
	}

	// If we have exactly 16 colors, map them to base00-base0F in order
	if len(colors) == 16 {
		result := make(map[string]string)
		baseNames := []string{
			"base00", "base01", "base02", "base03", "base04", "base05", "base06", "base07",
			"base08", "base09", "base0A", "base0B", "base0C", "base0D", "base0E", "base0F",
		}

		for i, color := range colors {
			result[baseNames[i]] = color
		}

		return result, nil
	}

	// If we don't have exactly 16, return error with helpful info
	return nil, fmt.Errorf("found %d unique colors, expected 16. Colors found: %v", len(colors), colors)
}
