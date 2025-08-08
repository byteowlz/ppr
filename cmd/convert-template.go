package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/byteowlz/ppr/pkg/config"
	"github.com/spf13/cobra"
)

var convertTemplateCmd = &cobra.Command{
	Use:   "convert-template",
	Short: "Convert an SVG file to a template with Base16 placeholders",
	Long: `Convert an SVG file to a template by replacing specific colors with Base16 placeholders.
This helps you create templates from existing SVG designs.`,
	RunE: runConvertTemplate,
}

var (
	inputSVG    string
	outputName  string
	colorMap    []string
	interactive bool
)

func init() {
	convertTemplateCmd.Flags().StringVarP(&inputSVG, "input", "i", "", "Input SVG file path")
	convertTemplateCmd.Flags().StringVarP(&outputName, "output", "o", "", "Output template name (without .svg extension)")
	convertTemplateCmd.Flags().StringSliceVarP(&colorMap, "map", "m", []string{}, "Color mappings in format 'color=placeholder' (e.g., '#2E3440=base00')")
	convertTemplateCmd.Flags().BoolVar(&interactive, "interactive", false, "Interactive mode to map colors")

	convertTemplateCmd.MarkFlagRequired("input")
}

func runConvertTemplate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	// Read input SVG
	content, err := os.ReadFile(inputSVG)
	if err != nil {
		return fmt.Errorf("failed to read input SVG: %w", err)
	}

	svgContent := string(content)

	// Extract unique colors from the SVG
	colors := extractColors(svgContent)
	fmt.Printf("Found %d unique colors in the SVG:\n", len(colors))
	for i, color := range colors {
		fmt.Printf("  %d. %s\n", i+1, color)
	}

	// Create color mapping
	mapping := make(map[string]string)

	if interactive {
		mapping, err = createInteractiveMapping(colors)
		if err != nil {
			return fmt.Errorf("failed to create interactive mapping: %w", err)
		}
	} else {
		mapping = parseColorMappings(colorMap)
	}

	// Apply mappings
	for color, placeholder := range mapping {
		svgContent = strings.ReplaceAll(svgContent, color, fmt.Sprintf("{{%s}}", placeholder))
	}

	// Determine output path
	outputPath := outputName
	if outputPath == "" {
		base := filepath.Base(inputSVG)
		outputPath = strings.TrimSuffix(base, filepath.Ext(base)) + "-template.svg"
	}
	if !strings.HasSuffix(outputPath, ".svg") {
		outputPath += ".svg"
	}

	finalPath := filepath.Join(cfg.TemplatesPath, outputPath)

	// Write template
	if err := os.WriteFile(finalPath, []byte(svgContent), 0644); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	fmt.Printf("Template created: %s\n", finalPath)
	fmt.Printf("Applied %d color mappings\n", len(mapping))

	return nil
}

func extractColors(svgContent string) []string {
	colorRegex := regexp.MustCompile(`(?i)(?:fill|stroke)="(#[0-9A-Fa-f]{6}|#[0-9A-Fa-f]{3})"`)
	matches := colorRegex.FindAllStringSubmatch(svgContent, -1)

	colorSet := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			color := strings.ToUpper(match[1])
			// Convert 3-digit hex to 6-digit
			if len(color) == 4 {
				color = fmt.Sprintf("#%c%c%c%c%c%c", color[1], color[1], color[2], color[2], color[3], color[3])
			}
			colorSet[color] = true
		}
	}

	var colors []string
	for color := range colorSet {
		colors = append(colors, color)
	}

	return colors
}

func parseColorMappings(mappings []string) map[string]string {
	result := make(map[string]string)
	for _, mapping := range mappings {
		parts := strings.Split(mapping, "=")
		if len(parts) == 2 {
			color := strings.ToUpper(strings.TrimSpace(parts[0]))
			placeholder := strings.TrimSpace(parts[1])
			result[color] = placeholder
		}
	}
	return result
}

func createInteractiveMapping(colors []string) (map[string]string, error) {
	mapping := make(map[string]string)

	base16Colors := []string{
		"base00", "base01", "base02", "base03", "base04", "base05", "base06", "base07",
		"base08", "base09", "base0A", "base0B", "base0C", "base0D", "base0E", "base0F",
	}

	fmt.Println("\nBase16 color meanings:")
	fmt.Println("  base00-base03: Background shades (darkest to lighter)")
	fmt.Println("  base04-base07: Foreground shades (darker to lightest)")
	fmt.Println("  base08: Red")
	fmt.Println("  base09: Orange")
	fmt.Println("  base0A: Yellow")
	fmt.Println("  base0B: Green")
	fmt.Println("  base0C: Cyan")
	fmt.Println("  base0D: Blue")
	fmt.Println("  base0E: Purple")
	fmt.Println("  base0F: Brown")
	fmt.Println()

	for _, color := range colors {
		fmt.Printf("Map color %s to which Base16 placeholder? ", color)
		fmt.Printf("(Available: %s, or 'skip'): ", strings.Join(base16Colors, ", "))

		var input string
		fmt.Scanln(&input)

		input = strings.TrimSpace(input)
		if input == "skip" || input == "" {
			continue
		}

		// Validate input
		valid := false
		for _, base := range base16Colors {
			if input == base {
				valid = true
				break
			}
		}

		if valid {
			mapping[color] = input
		} else {
			fmt.Printf("Invalid placeholder '%s', skipping...\n", input)
		}
	}

	return mapping, nil
}
