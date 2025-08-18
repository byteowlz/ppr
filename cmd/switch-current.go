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

	// Determine output directories and paths
	baseOutputDir := cfg.OutputPath
	if switchOutputPath != "" {
		baseOutputDir = switchOutputPath
	}

	// Create theme subdirectory under ppr for named variants
	themeSubDir := filepath.Join(baseOutputDir, "ppr", newThemeName)
	if err := os.MkdirAll(themeSubDir, 0755); err != nil {
		return fmt.Errorf("failed to create theme subdirectory: %w", err)
	}

	// Generate simplified filename for named variant (no timestamp)
	namedFilename := switchOutputFilename
	if namedFilename == "" {
		templateName := filepath.Base(templatePath)
		// Remove .svg extension if present
		if filepath.Ext(templateName) == ".svg" {
			templateName = templateName[:len(templateName)-4]
		}
		if switchOutputSVG {
			namedFilename = fmt.Sprintf("%s.svg", templateName)
		} else {
			namedFilename = fmt.Sprintf("%s.png", templateName)
		}
	}

	// Paths for both files
	namedVariantPath := filepath.Join(themeSubDir, namedFilename)
	currentWallpaperPath := filepath.Join(baseOutputDir, "current.png")

	if switchOutputSVG {
		// For SVG, only write the named variant (current.png doesn't make sense for SVG)
		// Check if named variant already exists
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

		// Check if named variant already exists, generate if not
		if _, err := os.Stat(namedVariantPath); err == nil {
			fmt.Printf("Reusing existing wallpaper: %s (%s)\n", namedVariantPath, res.String())
			namedVariantExists = true
		} else {
			if err := generator.GenerateWallpaper(svgContent, res.Width, res.Height, namedVariantPath); err != nil {
				return fmt.Errorf("failed to generate named wallpaper: %w", err)
			}
			fmt.Printf("Generated wallpaper: %s (%s)\n", namedVariantPath, res.String())
			namedVariantExists = true
		}

		// Copy named variant to current.png (more efficient than regenerating)
		if namedVariantExists {
			// Inline file copy to avoid function duplication
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

			fmt.Printf("Switched to theme '%s': %s\n", newThemeName, currentWallpaperPath)
		}
	}

	// Use current.png for wallpaper setting (or named variant for SVG)
	wallpaperPath := currentWallpaperPath
	if switchOutputSVG {
		wallpaperPath = namedVariantPath
	}

	if switchSetWallpaper || cfg.AutoSetWallpaper {
		// For macOS wallpaper caching issue, create a temporary file with unique name
		// This ensures the system recognizes it as a new wallpaper file
		timestamp := time.Now().Format("20060102-150405")
		tempWallpaperPath := filepath.Join(baseOutputDir, fmt.Sprintf("current_temp_%s.png", timestamp))

		// Copy current wallpaper to temp file for setting
		if !switchOutputSVG {
			// Inline file copy for temp wallpaper
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
						// Clean up old temp files (keep only the current one)
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

	// Update current state in config
	cfg.CurrentTheme = newThemeName
	cfg.CurrentTemplate = filepath.Base(templatePath)
	cfg.LastOutputPath = wallpaperPath
	if err := cfg.Save(); err != nil {
		fmt.Printf("Warning: failed to save current state: %v\n", err)
	}

	return nil
}
