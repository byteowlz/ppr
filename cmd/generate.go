package cmd

import (
	"fmt"
	"os"
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

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a themed wallpaper from an SVG template",
	Long: `Generate a themed wallpaper by applying a color theme to an SVG template.
The output will be a PNG file with the specified or auto-detected resolution.
If no template is specified, the default template from config will be used.`,
	RunE: runGenerate,
}

var (
	themeName      string
	templatePath   string
	outputPath     string
	resolutionStr  string
	setWallpaper   bool
	outputFilename string
	outputSVG      bool
)

func init() {
	generateCmd.Flags().StringVarP(&themeName, "theme", "t", "", "Theme name to apply")
	generateCmd.Flags().StringVarP(&templatePath, "template", "s", "", "Path to SVG template file (uses default template if not specified)")
	generateCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output directory (optional)")
	generateCmd.Flags().StringVarP(&resolutionStr, "resolution", "r", "", "Output resolution (e.g., 1920x1080)")
	generateCmd.Flags().BoolVarP(&setWallpaper, "set-wallpaper", "w", false, "Set generated image as wallpaper")
	generateCmd.Flags().StringVarP(&outputFilename, "filename", "f", "", "Output filename (optional)")
	generateCmd.Flags().BoolVar(&outputSVG, "svg", false, "Output SVG file instead of PNG (for Illustrator compatibility)")

	generateCmd.MarkFlagRequired("theme")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	themeManager := theme.NewThemeManager(cfg.ThemesPath)
	if err := themeManager.LoadThemes(); err != nil {
		return fmt.Errorf("failed to load themes: %w", err)
	}

	selectedTheme, err := themeManager.GetTheme(themeName)
	if err != nil {
		return fmt.Errorf("failed to get theme: %w", err)
	}

	// Use default template if none specified
	if templatePath == "" {
		templatePath = cfg.DefaultTemplate
		fmt.Printf("Using default template: %s\n", templatePath)
	}

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
	if resolutionStr != "" {
		res, err = resolution.ParseResolution(resolutionStr)
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
	if outputPath != "" {
		baseOutputDir = outputPath
	}

	// Create ppr subdirectory for named variants
	pprSubDir := filepath.Join(baseOutputDir, "ppr")
	if err := os.MkdirAll(pprSubDir, 0755); err != nil {
		return fmt.Errorf("failed to create ppr subdirectory: %w", err)
	}

	// Generate filename for named variant
	namedFilename := outputFilename
	if namedFilename == "" {
		timestamp := time.Now().Format("20060102-150405")
		if outputSVG {
			namedFilename = fmt.Sprintf("%s-%s-%s.svg", themeName, filepath.Base(templatePath), timestamp)
		} else {
			namedFilename = fmt.Sprintf("%s-%s-%s.png", themeName, filepath.Base(templatePath), timestamp)
		}
	}

	// Paths for both files
	namedVariantPath := filepath.Join(pprSubDir, namedFilename)
	currentWallpaperPath := filepath.Join(baseOutputDir, "current.png")

	if outputSVG {
		// For SVG, only write the named variant (current.png doesn't make sense for SVG)
		if err := processor.WriteSVG(svgContent, namedVariantPath); err != nil {
			return fmt.Errorf("failed to write SVG: %w", err)
		}
		fmt.Printf("Generated SVG: %s\n", namedVariantPath)
	} else {
		generator := image.NewGenerator()

		// Generate the named variant in ppr/ subdirectory
		if err := generator.GenerateWallpaper(svgContent, res.Width, res.Height, namedVariantPath); err != nil {
			return fmt.Errorf("failed to generate named wallpaper: %w", err)
		}

		// Generate current.png in the main wallpaper directory
		if err := generator.GenerateWallpaper(svgContent, res.Width, res.Height, currentWallpaperPath); err != nil {
			return fmt.Errorf("failed to generate current wallpaper: %w", err)
		}

		fmt.Printf("Generated wallpaper: %s (%s)\n", namedVariantPath, res.String())
		fmt.Printf("Current wallpaper saved as: %s\n", currentWallpaperPath)
	}

	// Use current.png for wallpaper setting (or named variant for SVG)
	wallpaperPath := currentWallpaperPath
	if outputSVG {
		wallpaperPath = namedVariantPath
	}

	if setWallpaper || cfg.AutoSetWallpaper {
		setter := wallpaper.NewSetter()
		if err := setter.SetWallpaper(wallpaperPath); err != nil {
			fmt.Printf("Warning: failed to set wallpaper: %v\n", err)
		} else {
			fmt.Println("Wallpaper set successfully!")
		}
	}

	// Update current state in config
	cfg.CurrentTheme = themeName
	cfg.CurrentTemplate = filepath.Base(templatePath)
	cfg.LastOutputPath = wallpaperPath
	if err := cfg.Save(); err != nil {
		fmt.Printf("Warning: failed to save current state: %v\n", err)
	}

	return nil
}
