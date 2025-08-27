package cmd

import (
	"fmt"
	"io"
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

	// Create theme subdirectory under ppr for named variants
	themeSubDir := filepath.Join(baseOutputDir, "ppr", themeName)
	if err := os.MkdirAll(themeSubDir, 0755); err != nil {
		return fmt.Errorf("failed to create theme subdirectory: %w", err)
	}

	// Generate simplified filename for named variant (no timestamp)
	namedFilename := outputFilename
	if namedFilename == "" {
		templateName := filepath.Base(templatePath)
		// Remove .svg extension if present
		if filepath.Ext(templateName) == ".svg" {
			templateName = templateName[:len(templateName)-4]
		}
		if outputSVG {
			namedFilename = fmt.Sprintf("%s.svg", templateName)
		} else {
			namedFilename = fmt.Sprintf("%s.png", templateName)
		}
	}

	// Paths for both files
	namedVariantPath := filepath.Join(themeSubDir, namedFilename)
	currentWallpaperPath := filepath.Join(baseOutputDir, "current.png")

	generator := image.NewGenerator()
	var pngGenerated bool

	if outputSVG {
		// Generate SVG version
		if _, err := os.Stat(namedVariantPath); err == nil {
			fmt.Printf("Reusing existing SVG: %s\n", namedVariantPath)
		} else {
			if err := processor.WriteSVG(svgContent, namedVariantPath); err != nil {
				return fmt.Errorf("failed to write SVG: %w", err)
			}
			fmt.Printf("Generated SVG: %s\n", namedVariantPath)
		}
	}

	// Always generate PNG if not outputSVG, or if outputSVG but wallpaper setting is requested
	if !outputSVG || setWallpaper || cfg.AutoSetWallpaper {
		// For PNG filename when outputSVG is true but we need PNG for wallpaper
		pngFilename := outputFilename
		if outputSVG && pngFilename != "" {
			// Replace .svg extension with .png for the PNG version
			if filepath.Ext(pngFilename) == ".svg" {
				pngFilename = pngFilename[:len(pngFilename)-4] + ".png"
			}
		} else if outputSVG {
			// Generate PNG filename from template name
			templateName := filepath.Base(templatePath)
			if filepath.Ext(templateName) == ".svg" {
				templateName = templateName[:len(templateName)-4]
			}
			pngFilename = fmt.Sprintf("%s.png", templateName)
		}

		pngPath := namedVariantPath
		if outputSVG {
			pngPath = filepath.Join(themeSubDir, pngFilename)
		}

		// Check if PNG variant already exists, generate if not
		if _, err := os.Stat(pngPath); err == nil {
			fmt.Printf("Reusing existing wallpaper: %s (%s)\n", pngPath, res.String())
			pngGenerated = true
		} else {
			if err := generator.GenerateWallpaper(svgContent, res.Width, res.Height, pngPath); err != nil {
				return fmt.Errorf("failed to generate wallpaper: %w", err)
			}
			fmt.Printf("Generated wallpaper: %s (%s)\n", pngPath, res.String())
			pngGenerated = true
		}

		// Copy PNG variant to current.png (more efficient than regenerating)
		if pngGenerated {
			if err := copyFile(pngPath, currentWallpaperPath); err != nil {
				return fmt.Errorf("failed to copy to current wallpaper: %w", err)
			}
			fmt.Printf("Current wallpaper saved as: %s\n", currentWallpaperPath)
		}
	}

	// Use current.png for wallpaper setting (always PNG, even when SVG was also generated)
	wallpaperPath := currentWallpaperPath

	if setWallpaper || cfg.AutoSetWallpaper {
		if !pngGenerated {
			fmt.Printf("Warning: Cannot set wallpaper without PNG file\n")
		} else {
			// For macOS wallpaper caching issue, create a temporary file with unique name
			// This ensures the system recognizes it as a new wallpaper file
			timestamp := time.Now().Format("20060102-150405")
			tempWallpaperPath := filepath.Join(baseOutputDir, fmt.Sprintf("current_temp_%s.png", timestamp))

			// Copy current wallpaper to temp file for setting
			if err := copyFile(wallpaperPath, tempWallpaperPath); err != nil {
				fmt.Printf("Warning: failed to create temp wallpaper file: %v\n", err)
			} else {
				wallpaperPath = tempWallpaperPath
				// Clean up old temp files (keep only the current one)
				inlineCleanupOldTempFiles(baseOutputDir)
			}

			setter := wallpaper.NewSetter()
			if err := setter.SetWallpaper(wallpaperPath); err != nil {
				fmt.Printf("Warning: failed to set wallpaper: %v\n", err)
			} else {
				fmt.Println("Wallpaper set successfully!")
			}
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

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
