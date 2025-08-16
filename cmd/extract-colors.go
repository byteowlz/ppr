package cmd

import (
	"encoding/xml"
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
	Long: `Extract base16 color scheme from an SVG file that contains color swatches.
The SVG should contain elements with fills that match the base00-base0F pattern.
This creates a new theme file that can be used with other templates.`,
	Args: cobra.ExactArgs(2),
	RunE: runExtractColors,
}

type SVGElement struct {
	XMLName xml.Name `xml:"rect"`
	Fill    string   `xml:"fill,attr"`
}

type SVGGroup struct {
	XMLName  xml.Name     `xml:"g"`
	Elements []SVGElement `xml:"rect"`
	Text     []SVGText    `xml:"text"`
}

type SVGText struct {
	XMLName xml.Name   `xml:"text"`
	Content string     `xml:",chardata"`
	Tspans  []SVGTspan `xml:"tspan"`
}

type SVGTspan struct {
	XMLName xml.Name `xml:"tspan"`
	Content string   `xml:",chardata"`
}

type SVGDocument struct {
	XMLName xml.Name     `xml:"svg"`
	Groups  []SVGGroup   `xml:"g"`
	Rects   []SVGElement `xml:"rect"`
	Defs    []SVGDefs    `xml:"defs"`
}

type SVGDefs struct {
	XMLName xml.Name   `xml:"defs"`
	Styles  []SVGStyle `xml:"style"`
}

type SVGStyle struct {
	XMLName xml.Name `xml:"style"`
	Content string   `xml:",chardata"`
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
	colors := make(map[string]string)

	// Parse XML
	var doc SVGDocument
	if err := xml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse SVG: %w", err)
	}

	// Extract CSS classes to color mapping from styles
	cssColors := extractCSSColors(doc.Defs)

	// Extract colors from groups and their text labels
	for _, group := range doc.Groups {
		var fillColor string
		var baseLabel string

		// Get fill color from rect elements in the group
		for _, rect := range group.Elements {
			if rect.Fill != "" && rect.Fill != "#222" && !strings.Contains(rect.Fill, "url(") {
				fillColor = rect.Fill
				break
			}
		}

		// If no direct fill found, look for CSS class colors
		if fillColor == "" {
			for _, rect := range group.Elements {
				if rect.XMLName.Local == "rect" {
					// Look for class attribute in the raw XML
					classColor := extractColorFromClass(string(content), cssColors)
					if classColor != "" {
						fillColor = classColor
						break
					}
				}
			}
		}

		// Get base label from text elements
		for _, text := range group.Text {
			// Check direct text content
			if strings.Contains(text.Content, "base") {
				baseLabel = extractBaseLabel(text.Content)
				break
			}
			// Check tspan content
			for _, tspan := range text.Tspans {
				if strings.Contains(tspan.Content, "base") {
					baseLabel = extractBaseLabel(tspan.Content)
					break
				}
			}
			if baseLabel != "" {
				break
			}
		}

		// Map color to base if both are found
		if fillColor != "" && baseLabel != "" {
			colors[baseLabel] = fillColor
		}
	}

	// Fallback: extract colors from tspan text directly with regex
	if len(colors) == 0 {
		colors = extractColorsFromText(string(content))
	}

	return colors, nil
}

func extractBaseLabel(text string) string {
	// Match base00 through base0F pattern
	re := regexp.MustCompile(`base[0-9A-F][0-9A-F]`)
	match := re.FindString(text)
	return match
}

func extractCSSColors(defs []SVGDefs) map[string]string {
	cssColors := make(map[string]string)

	for _, def := range defs {
		for _, style := range def.Styles {
			// Parse CSS style content
			lines := strings.Split(style.Content, "\n")
			var currentClass string

			for _, line := range lines {
				line = strings.TrimSpace(line)

				// Check for class definition
				if strings.HasPrefix(line, ".") && strings.Contains(line, "{") {
					currentClass = strings.Split(strings.TrimPrefix(line, "."), " ")[0]
					currentClass = strings.TrimSuffix(currentClass, ",")
					currentClass = strings.TrimSuffix(currentClass, "{")
				}

				// Check for fill property
				if strings.Contains(line, "fill:") && currentClass != "" {
					colorValue := strings.TrimSpace(strings.Split(line, "fill:")[1])
					colorValue = strings.TrimSuffix(colorValue, ";")
					if strings.HasPrefix(colorValue, "#") {
						cssColors[currentClass] = colorValue
					}
				}
			}
		}
	}

	return cssColors
}

func extractColorFromClass(content string, cssColors map[string]string) string {
	// This is a simplified approach - in a real implementation,
	// we'd need to properly parse the class attribute from rect elements
	return ""
}

func extractColorsFromText(content string) map[string]string {
	colors := make(map[string]string)

	// Look for base color labels with hex colors in the text
	re := regexp.MustCompile(`(base[0-9A-F][0-9A-F])[^#]*#([0-9A-Fa-f]{6})`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			baseLabel := match[1]
			colorValue := "#" + strings.ToUpper(match[2])
			colors[baseLabel] = colorValue
		}
	}

	return colors
}
