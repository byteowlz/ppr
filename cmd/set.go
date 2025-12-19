package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/byteowlz/ppr/pkg/config"
	"github.com/byteowlz/ppr/pkg/image"
	"github.com/byteowlz/ppr/pkg/resolution"
	"github.com/byteowlz/ppr/pkg/svg"
	"github.com/byteowlz/ppr/pkg/theme"
	"github.com/byteowlz/ppr/pkg/wallpaper"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set <template_name>",
	Short: "Switch to a template and generate wallpaper",
	Long: `Switch to a template by name, generate the wallpaper with the current theme,
and optionally set it as the desktop wallpaper. The template name can be provided
with or without the .svg extension. Fuzzy matching is supported, so partial names
will match the closest template.

Examples:
  ppr set shapes
  ppr set shapes.svg
  ppr set vert       # matches vertical_bar.svg
  ppr set horizontal # matches horizontal_bar.svg`,
	Args: cobra.ExactArgs(1),
	RunE: runSet,
}

var (
	setWallpaperFlag bool
	setOutputPath    string
	setResolutionStr string
	setOutputSVG     bool
)

func init() {
	setCmd.Flags().BoolVarP(&setWallpaperFlag, "set-wallpaper", "w", false, "Set generated image as wallpaper")
	setCmd.Flags().StringVarP(&setOutputPath, "output", "o", "", "Output directory (optional)")
	setCmd.Flags().StringVarP(&setResolutionStr, "resolution", "r", "", "Output resolution (e.g., 1920x1080)")
	setCmd.Flags().BoolVar(&setOutputSVG, "svg", false, "Output SVG file instead of PNG")
}

func runSet(cmd *cobra.Command, args []string) error {
	templateInput := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	// Get all available templates
	templates, err := findTemplates(cfg.TemplatesPath)
	if err != nil {
		return fmt.Errorf("failed to find templates: %w", err)
	}

	if len(templates) == 0 {
		return fmt.Errorf("no templates found in %s", cfg.TemplatesPath)
	}

	// Find the best matching template
	matchedTemplate, err := fuzzyMatchTemplate(templateInput, templates)
	if err != nil {
		return err
	}

	// Determine which theme to use
	themeToUse := cfg.CurrentTheme
	if themeToUse == "" {
		themeToUse = cfg.DefaultTheme
		fmt.Printf("No current theme found, using default: %s\n", themeToUse)
	}

	// Load themes
	themeManager := theme.NewThemeManager(cfg.ThemesPath)
	if err := themeManager.LoadThemes(); err != nil {
		return fmt.Errorf("failed to load themes: %w", err)
	}

	selectedTheme, err := themeManager.GetTheme(themeToUse)
	if err != nil {
		return fmt.Errorf("failed to get theme: %w", err)
	}

	// Build full template path
	templatePath := filepath.Join(cfg.TemplatesPath, matchedTemplate)

	// Process template
	processor := svg.NewProcessor()
	svgContent, err := processor.ProcessTemplate(templatePath, selectedTheme)
	if err != nil {
		return fmt.Errorf("failed to process template: %w", err)
	}

	// Determine resolution
	var res *resolution.Resolution
	if setResolutionStr != "" {
		res, err = resolution.ParseResolution(setResolutionStr)
		if err != nil {
			return fmt.Errorf("failed to parse resolution: %w", err)
		}
	} else {
		detector := resolution.NewDetector()
		res, err = detector.GetPrimaryDisplayResolution()
		if err != nil {
			fmt.Printf("Warning: failed to detect resolution, using default: %v\n", err)
			res = &resolution.Resolution{Width: cfg.DefaultWidth, Height: cfg.DefaultHeight}
		}
	}

	// Determine output directories and paths
	baseOutputDir := cfg.OutputPath
	if setOutputPath != "" {
		baseOutputDir = setOutputPath
	}

	// Create theme subdirectory under ppr for named variants
	themeSubDir := filepath.Join(baseOutputDir, "ppr", themeToUse)
	if err := os.MkdirAll(themeSubDir, 0755); err != nil {
		return fmt.Errorf("failed to create theme subdirectory: %w", err)
	}

	// Generate filename
	templateName := filepath.Base(templatePath)
	templateName = strings.TrimSuffix(templateName, ".svg")
	var namedFilename string
	if setOutputSVG {
		namedFilename = fmt.Sprintf("%s.svg", templateName)
	} else {
		namedFilename = fmt.Sprintf("%s.png", templateName)
	}

	// Paths for files
	namedVariantPath := filepath.Join(themeSubDir, namedFilename)
	currentWallpaperPath := filepath.Join(baseOutputDir, "current.png")

	if setOutputSVG {
		if _, err := os.Stat(namedVariantPath); err == nil {
			fmt.Printf("Reusing existing SVG: %s\n", namedVariantPath)
		} else {
			if err := processor.WriteSVG(svgContent, namedVariantPath); err != nil {
				return fmt.Errorf("failed to write SVG: %w", err)
			}
			fmt.Printf("Generated SVG: %s\n", namedVariantPath)
		}
	} else {
		generator := image.NewGenerator()
		var namedVariantExists bool

		if _, err := os.Stat(namedVariantPath); err == nil {
			fmt.Printf("Reusing existing wallpaper: %s (%s)\n", namedVariantPath, res.String())
			namedVariantExists = true
		} else {
			if err := generator.GenerateWallpaper(svgContent, res.Width, res.Height, namedVariantPath); err != nil {
				return fmt.Errorf("failed to generate wallpaper: %w", err)
			}
			fmt.Printf("Generated wallpaper: %s (%s)\n", namedVariantPath, res.String())
			namedVariantExists = true
		}

		// Copy named variant to current.png
		if namedVariantExists {
			sourceFile, err := os.Open(namedVariantPath)
			if err != nil {
				return fmt.Errorf("failed to open source file: %w", err)
			}
			defer sourceFile.Close()

			destFile, err := os.Create(currentWallpaperPath)
			if err != nil {
				return fmt.Errorf("failed to create dest file: %w", err)
			}
			defer destFile.Close()

			if _, err := io.Copy(destFile, sourceFile); err != nil {
				return fmt.Errorf("failed to copy file: %w", err)
			}

			fmt.Printf("Switched to template '%s': %s\n", matchedTemplate, currentWallpaperPath)
		}
	}

	// Set wallpaper if requested
	wallpaperPath := currentWallpaperPath
	if setOutputSVG {
		wallpaperPath = namedVariantPath
	}

	if setWallpaperFlag || cfg.AutoSetWallpaper {
		timestamp := time.Now().Format("20060102-150405")
		tempWallpaperPath := filepath.Join(baseOutputDir, fmt.Sprintf("current_temp_%s.png", timestamp))

		if !setOutputSVG {
			sourceFile, err := os.Open(wallpaperPath)
			if err != nil {
				fmt.Printf("Warning: failed to open wallpaper file: %v\n", err)
			} else {
				defer sourceFile.Close()

				destFile, err := os.Create(tempWallpaperPath)
				if err != nil {
					fmt.Printf("Warning: failed to create temp wallpaper file: %v\n", err)
				} else {
					defer destFile.Close()

					if _, err := io.Copy(destFile, sourceFile); err != nil {
						fmt.Printf("Warning: failed to copy to temp wallpaper: %v\n", err)
					} else {
						wallpaperPath = tempWallpaperPath
						inlineCleanupOldTempFiles(baseOutputDir)
					}
				}
			}
		}

		setter := wallpaper.NewSetter()
		if err := setter.SetWallpaper(wallpaperPath); err != nil {
			fmt.Printf("Warning: failed to set wallpaper: %v\n", err)
		} else {
			fmt.Println("Wallpaper set successfully!")
		}
	}

	// Update config
	cfg.CurrentTemplate = matchedTemplate
	cfg.LastOutputPath = wallpaperPath
	if err := cfg.Save(); err != nil {
		fmt.Printf("Warning: failed to save config: %v\n", err)
	}

	return nil
}

// fuzzyMatchTemplate finds the best matching template for the given input.
// It supports:
// - Exact match (with or without .svg extension)
// - Prefix match
// - Contains match
// - Fuzzy substring match
func fuzzyMatchTemplate(input string, templates []string) (string, error) {
	// Normalize input: lowercase and remove .svg extension if present
	normalizedInput := strings.ToLower(input)
	normalizedInput = strings.TrimSuffix(normalizedInput, ".svg")

	// Build a list of templates with their normalized names (without extension)
	type templateMatch struct {
		original   string
		normalized string
		score      int
	}

	var matches []templateMatch

	for _, t := range templates {
		normalized := strings.ToLower(t)
		normalized = strings.TrimSuffix(normalized, ".svg")

		match := templateMatch{
			original:   t,
			normalized: normalized,
			score:      0,
		}

		// Exact match (highest priority)
		if normalized == normalizedInput {
			match.score = 100
		} else if strings.HasPrefix(normalized, normalizedInput) {
			// Prefix match (high priority)
			match.score = 80
		} else if strings.Contains(normalized, normalizedInput) {
			// Contains match (medium priority)
			match.score = 60
		} else if fuzzyContains(normalized, normalizedInput) {
			// Fuzzy match - all characters appear in order (lower priority)
			match.score = 40
		}

		if match.score > 0 {
			matches = append(matches, match)
		}
	}

	if len(matches) == 0 {
		// Show available templates in error message
		sort.Strings(templates)
		return "", fmt.Errorf("no template matching '%s' found\nAvailable templates:\n  %s",
			input, strings.Join(templates, "\n  "))
	}

	// Sort by score (descending), then by name length (shorter first for tie-breaking)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return len(matches[i].normalized) < len(matches[j].normalized)
	})

	bestMatch := matches[0]

	// If there are multiple matches with the same score, show them
	if len(matches) > 1 && matches[0].score == matches[1].score && matches[0].score < 100 {
		var ambiguous []string
		for _, m := range matches {
			if m.score == bestMatch.score {
				ambiguous = append(ambiguous, m.original)
			}
		}
		if len(ambiguous) > 1 {
			return "", fmt.Errorf("ambiguous template name '%s', matches multiple templates:\n  %s\nPlease be more specific",
				input, strings.Join(ambiguous, "\n  "))
		}
	}

	return bestMatch.original, nil
}

// fuzzyContains checks if all characters of needle appear in haystack in order
func fuzzyContains(haystack, needle string) bool {
	needleIdx := 0
	for i := 0; i < len(haystack) && needleIdx < len(needle); i++ {
		if haystack[i] == needle[needleIdx] {
			needleIdx++
		}
	}
	return needleIdx == len(needle)
}

// getTemplateDisplayName returns a user-friendly display name for a template
func getTemplateDisplayName(templatePath string) string {
	name := filepath.Base(templatePath)
	return strings.TrimSuffix(name, ".svg")
}
