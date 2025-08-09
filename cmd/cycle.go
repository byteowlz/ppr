package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/byteowlz/ppr/pkg/config"
	"github.com/byteowlz/ppr/pkg/image"
	"github.com/byteowlz/ppr/pkg/resolution"
	"github.com/byteowlz/ppr/pkg/svg"
	"github.com/byteowlz/ppr/pkg/theme"
	"github.com/byteowlz/ppr/pkg/wallpaper"
	"github.com/spf13/cobra"
)

var cycleCmd = &cobra.Command{
	Use:   "cycle [theme-name]",
	Short: "Cycle through preferred templates and set as wallpaper",
	Long: `Cycle through the preferred templates configured in config.toml and set as wallpaper.
If preferred_templates contains "all", it will cycle through all available templates.
Otherwise, it cycles through the specified list of preferred templates.
Uses the current theme if no theme is specified.
The wallpaper is set automatically by default.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCycle,
}

var (
	cycleSetWallpaper   bool
	cycleOutputPath     string
	cycleOutputFilename string
	cycleResolutionStr  string
	cycleOutputSVG      bool
)

func init() {
	cycleCmd.Flags().BoolVarP(&cycleSetWallpaper, "set-wallpaper", "w", true, "Set generated image as wallpaper (default: true)")
	cycleCmd.Flags().StringVarP(&cycleOutputPath, "output", "o", "", "Output directory (optional)")
	cycleCmd.Flags().StringVarP(&cycleOutputFilename, "filename", "f", "", "Output filename (optional)")
	cycleCmd.Flags().StringVarP(&cycleResolutionStr, "resolution", "r", "", "Output resolution (e.g., 1920x1080)")
	cycleCmd.Flags().BoolVar(&cycleOutputSVG, "svg", false, "Output SVG file instead of PNG")
}

func runCycle(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	// Determine theme to use
	themeToUse := cfg.CurrentTheme
	if len(args) > 0 {
		themeToUse = args[0]
	}
	if themeToUse == "" {
		themeToUse = cfg.DefaultTheme
		fmt.Printf("No current or specified theme, using default: %s\n", themeToUse)
	}

	themeManager := theme.NewThemeManager(cfg.ThemesPath)
	if err := themeManager.LoadThemes(); err != nil {
		return fmt.Errorf("failed to load themes: %w", err)
	}

	selectedTheme, err := themeManager.GetTheme(themeToUse)
	if err != nil {
		return fmt.Errorf("failed to get theme: %w", err)
	}

	// Get templates to cycle through
	templates, err := getTemplatesToCycle(cfg)
	if err != nil {
		return fmt.Errorf("failed to get templates: %w", err)
	}

	if len(templates) == 0 {
		return fmt.Errorf("no templates available to cycle through")
	}

	// Find next template to use
	nextTemplate := getNextTemplate(templates, cfg.CurrentTemplate)
	fmt.Printf("Cycling to template: %s\n", nextTemplate)

	// Build full template path
	templatePath := nextTemplate
	if !filepath.IsAbs(templatePath) {
		templatePath = filepath.Join(cfg.TemplatesPath, templatePath)
	}

	// Add .svg extension if not present
	if filepath.Ext(templatePath) == "" {
		templatePath += ".svg"
	}

	processor := svg.NewProcessor()
	svgContent, err := processor.ProcessTemplate(templatePath, selectedTheme)
	if err != nil {
		return fmt.Errorf("failed to process template: %w", err)
	}

	var res *resolution.Resolution
	if cycleResolutionStr != "" {
		res, err = resolution.ParseResolution(cycleResolutionStr)
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

	outputDir := cfg.OutputPath
	if cycleOutputPath != "" {
		outputDir = cycleOutputPath
	}

	filename := cycleOutputFilename
	if filename == "" {
		timestamp := time.Now().Format("20060102-150405")
		if cycleOutputSVG {
			filename = fmt.Sprintf("%s-%s-%s.svg", themeToUse, filepath.Base(nextTemplate), timestamp)
		} else {
			filename = fmt.Sprintf("%s-%s-%s.png", themeToUse, filepath.Base(nextTemplate), timestamp)
		}
	}

	finalOutputPath := filepath.Join(outputDir, filename)

	if cycleOutputSVG {
		// Write SVG directly with resolved colors
		if err := processor.WriteSVG(svgContent, finalOutputPath); err != nil {
			return fmt.Errorf("failed to write SVG: %w", err)
		}
		fmt.Printf("Generated SVG: %s\n", finalOutputPath)
	} else {
		generator := image.NewGenerator()
		if err := generator.GenerateWallpaper(svgContent, res.Width, res.Height, finalOutputPath); err != nil {
			return fmt.Errorf("failed to generate wallpaper: %w", err)
		}
		fmt.Printf("Cycled to template '%s' with theme '%s': %s (%s)\n", nextTemplate, themeToUse, finalOutputPath, res.String())
	}

	// Always set wallpaper by default for cycle command, unless explicitly disabled
	if cycleSetWallpaper {
		setter := wallpaper.NewSetter()
		if err := setter.SetWallpaper(finalOutputPath); err != nil {
			fmt.Printf("Warning: failed to set wallpaper: %v\n", err)
		} else {
			fmt.Println("Wallpaper set successfully!")
		}
	}

	// Update current state in config
	cfg.CurrentTheme = themeToUse
	cfg.CurrentTemplate = filepath.Base(nextTemplate)
	cfg.LastOutputPath = finalOutputPath
	if err := cfg.Save(); err != nil {
		fmt.Printf("Warning: failed to save current state: %v\n", err)
	}

	return nil
}

func getTemplatesToCycle(cfg *config.Config) ([]string, error) {
	if len(cfg.PreferredTemplates) == 0 {
		return nil, fmt.Errorf("no preferred templates configured")
	}

	// Check if "all" is specified
	for _, template := range cfg.PreferredTemplates {
		if template == "all" {
			// Get all available templates
			allTemplates, err := findTemplates(cfg.TemplatesPath)
			if err != nil {
				return nil, fmt.Errorf("failed to find all templates: %w", err)
			}
			sort.Strings(allTemplates)
			return allTemplates, nil
		}
	}

	// Return the specific preferred templates
	return cfg.PreferredTemplates, nil
}

func getNextTemplate(templates []string, currentTemplate string) string {
	if len(templates) == 0 {
		return ""
	}

	if len(templates) == 1 {
		return templates[0]
	}

	// Find current template index
	currentIndex := -1
	for i, template := range templates {
		if template == currentTemplate || filepath.Base(template) == currentTemplate {
			currentIndex = i
			break
		}
	}

	// If current template not found or is the last one, return first template
	if currentIndex == -1 || currentIndex == len(templates)-1 {
		return templates[0]
	}

	// Return next template
	return templates[currentIndex+1]
}
