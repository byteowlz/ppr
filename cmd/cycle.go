package cmd

import (
	"fmt"
	"io"
	"os"
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

	// Determine output directories and paths
	baseOutputDir := cfg.OutputPath
	if cycleOutputPath != "" {
		baseOutputDir = cycleOutputPath
	}

	// Create theme subdirectory under ppr for named variants
	themeSubDir := filepath.Join(baseOutputDir, "ppr", themeToUse)
	if err := os.MkdirAll(themeSubDir, 0755); err != nil {
		return fmt.Errorf("failed to create theme subdirectory: %w", err)
	}

	// Generate simplified filename for named variant (no timestamp)
	namedFilename := cycleOutputFilename
	if namedFilename == "" {
		templateName := filepath.Base(nextTemplate)
		// Remove .svg extension if present
		if filepath.Ext(templateName) == ".svg" {
			templateName = templateName[:len(templateName)-4]
		}
		if cycleOutputSVG {
			namedFilename = fmt.Sprintf("%s.svg", templateName)
		} else {
			namedFilename = fmt.Sprintf("%s.png", templateName)
		}
	}

	// Paths for both files
	namedVariantPath := filepath.Join(themeSubDir, namedFilename)
	currentWallpaperPath := filepath.Join(baseOutputDir, "current.png")

	if cycleOutputSVG {
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

			fmt.Printf("Cycled to template '%s' with theme '%s': %s\n", nextTemplate, themeToUse, currentWallpaperPath)
		}
	}

	// Use current.png for wallpaper setting (or named variant for SVG)
	wallpaperPath := currentWallpaperPath
	if cycleOutputSVG {
		wallpaperPath = namedVariantPath
	}

	// Always set wallpaper by default for cycle command, unless explicitly disabled
	if cycleSetWallpaper {
		// For macOS wallpaper caching issue, create a temporary file with unique name
		// This ensures the system recognizes it as a new wallpaper file
		timestamp := time.Now().Format("20060102-150405")
		tempWallpaperPath := filepath.Join(baseOutputDir, fmt.Sprintf("current_temp_%s.png", timestamp))

		// Copy current wallpaper to temp file for setting
		if !cycleOutputSVG {
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
						// Inline cleanup to avoid function duplication
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
	cfg.CurrentTheme = themeToUse
	cfg.CurrentTemplate = filepath.Base(nextTemplate)
	cfg.LastOutputPath = wallpaperPath
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

// inlineCleanupOldTempFiles removes old temporary wallpaper files to prevent clutter
func inlineCleanupOldTempFiles(baseDir string) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".png" {
			// Check if it's an old temp file (current_temp_*.png)
			if len(entry.Name()) > 12 && entry.Name()[:12] == "current_temp" {
				// Get file info to check age
				if info, err := entry.Info(); err == nil {
					// Remove temp files older than 1 hour
					if time.Since(info.ModTime()) > time.Hour {
						os.Remove(filepath.Join(baseDir, entry.Name()))
					}
				}
			}
		}
	}
}
