package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/byteowlz/ppr/pkg/config"
	"github.com/byteowlz/ppr/pkg/image"
	"github.com/byteowlz/ppr/pkg/resolution"
	"github.com/byteowlz/ppr/pkg/svg"
	"github.com/byteowlz/ppr/pkg/theme"
	"github.com/byteowlz/ppr/pkg/wallpaper"
	"github.com/spf13/cobra"
)

var switchCurrentCmd = &cobra.Command{
	Use:   "switch-current [theme-name]",
	Short: "Switch the color scheme of the currently used template",
	Long: `Switch the color scheme of the currently used template to a new theme.
This command uses the last generated template and applies a new color theme to it.
If no current template is found, it will use the default template from config.`,
	Args: cobra.ExactArgs(1),
	RunE: runSwitchCurrent,
}

var (
	switchSetWallpaper   bool
	switchOutputPath     string
	switchOutputFilename string
	switchResolutionStr  string
	switchOutputSVG      bool
)

func init() {
	switchCurrentCmd.Flags().BoolVarP(&switchSetWallpaper, "set-wallpaper", "w", false, "Set generated image as wallpaper")
	switchCurrentCmd.Flags().StringVarP(&switchOutputPath, "output", "o", "", "Output directory (optional)")
	switchCurrentCmd.Flags().StringVarP(&switchOutputFilename, "filename", "f", "", "Output filename (optional)")
	switchCurrentCmd.Flags().StringVarP(&switchResolutionStr, "resolution", "r", "", "Output resolution (e.g., 1920x1080)")
	switchCurrentCmd.Flags().BoolVar(&switchOutputSVG, "svg", false, "Output SVG file instead of PNG")
}

func runSwitchCurrent(cmd *cobra.Command, args []string) error {
	newThemeName := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	// Determine which template to use
	templateToUse := cfg.CurrentTemplate
	if templateToUse == "" {
		templateToUse = cfg.DefaultTemplate
		fmt.Printf("No current template found, using default: %s\n", templateToUse)
	} else {
		fmt.Printf("Using current template: %s\n", templateToUse)
	}

	themeManager := theme.NewThemeManager(cfg.ThemesPath)
	if err := themeManager.LoadThemes(); err != nil {
		return fmt.Errorf("failed to load themes: %w", err)
	}

	selectedTheme, err := themeManager.GetTheme(newThemeName)
	if err != nil {
		return fmt.Errorf("failed to get theme: %w", err)
	}

	// Build full template path
	templatePath := templateToUse
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
	if switchResolutionStr != "" {
		res, err = resolution.ParseResolution(switchResolutionStr)
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
	if switchOutputPath != "" {
		outputDir = switchOutputPath
	}

	filename := switchOutputFilename
	if filename == "" {
		timestamp := time.Now().Format("20060102-150405")
		if switchOutputSVG {
			filename = fmt.Sprintf("%s-%s-%s.svg", newThemeName, filepath.Base(templatePath), timestamp)
		} else {
			filename = fmt.Sprintf("%s-%s-%s.png", newThemeName, filepath.Base(templatePath), timestamp)
		}
	}

	finalOutputPath := filepath.Join(outputDir, filename)

	if switchOutputSVG {
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
		fmt.Printf("Switched to theme '%s': %s (%s)\n", newThemeName, finalOutputPath, res.String())
	}

	if switchSetWallpaper || cfg.AutoSetWallpaper {
		setter := wallpaper.NewSetter()
		if err := setter.SetWallpaper(finalOutputPath); err != nil {
			fmt.Printf("Warning: failed to set wallpaper: %v\n", err)
		} else {
			fmt.Println("Wallpaper set successfully!")
		}
	}

	// Update current state in config
	cfg.CurrentTheme = newThemeName
	cfg.CurrentTemplate = filepath.Base(templatePath)
	cfg.LastOutputPath = finalOutputPath
	if err := cfg.Save(); err != nil {
		fmt.Printf("Warning: failed to save current state: %v\n", err)
	}

	return nil
}
