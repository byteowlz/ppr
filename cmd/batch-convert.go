package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/byteowlz/ppr/pkg/config"
	"github.com/byteowlz/ppr/pkg/theme"
	"github.com/spf13/cobra"
)

var batchConvertCmd = &cobra.Command{
	Use:   "batch-convert [files...]",
	Short: "Convert multiple SVG files to templates with a single color theme",
	Long: `Convert multiple SVG files to templates by replacing colors with Base16 placeholders.
This creates blank theme templates from multiple SVGs using one color theme for all files.

Examples:
  ppr batch-convert *.svg --from-theme nord
  ppr batch-convert file1.svg file2.svg --from-theme dracula
  ppr batch-convert --input "*.svg" --from-theme nord
  ppr batch-convert --input-dir ./designs/ --from-theme monokai`,
	RunE: runBatchConvert,
}

var (
	batchInputPattern string
	batchInputDir     string
	batchInputFiles   []string
	batchFromTheme    string
	batchOutputDir    string
	batchOutputSuffix string
)

func init() {
	batchConvertCmd.Flags().StringVar(&batchInputPattern, "input", "", "Input pattern for SVG files (e.g., '*.svg')")
	batchConvertCmd.Flags().StringVar(&batchInputDir, "input-dir", "", "Directory containing SVG files to process")
	batchConvertCmd.Flags().StringSliceVar(&batchInputFiles, "files", []string{}, "Comma-separated list of specific SVG files")
	batchConvertCmd.Flags().StringVar(&batchFromTheme, "from-theme", "", "Theme to use for color mapping (required)")
	batchConvertCmd.Flags().StringVar(&batchOutputDir, "output-dir", "", "Output directory for templates (defaults to config templates path)")
	batchConvertCmd.Flags().StringVar(&batchOutputSuffix, "suffix", "-template", "Suffix to add to output filenames")

	batchConvertCmd.MarkFlagRequired("from-theme")
}

func runBatchConvert(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	// Load theme manager and get the specified theme
	themeManager := theme.NewThemeManager(cfg.ThemesPath)
	if err := themeManager.LoadThemes(); err != nil {
		return fmt.Errorf("failed to load themes: %w", err)
	}

	selectedTheme, err := themeManager.GetTheme(batchFromTheme)
	if err != nil {
		return fmt.Errorf("failed to get theme '%s': %w", batchFromTheme, err)
	}

	// Determine output directory
	outputDir := cfg.TemplatesPath
	if batchOutputDir != "" {
		outputDir = batchOutputDir
	}

	// Collect input files
	var inputFiles []string

	// First, add files from command line arguments
	for _, arg := range args {
		if strings.HasSuffix(strings.ToLower(arg), ".svg") {
			inputFiles = append(inputFiles, arg)
		}
	}

	if batchInputPattern != "" {
		matches, err := filepath.Glob(batchInputPattern)
		if err != nil {
			return fmt.Errorf("failed to match pattern '%s': %w", batchInputPattern, err)
		}
		inputFiles = append(inputFiles, matches...)
	}

	if batchInputDir != "" {
		entries, err := os.ReadDir(batchInputDir)
		if err != nil {
			return fmt.Errorf("failed to read directory '%s': %w", batchInputDir, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".svg") {
				inputFiles = append(inputFiles, filepath.Join(batchInputDir, entry.Name()))
			}
		}
	}

	if len(batchInputFiles) > 0 {
		inputFiles = append(inputFiles, batchInputFiles...)
	}

	if len(inputFiles) == 0 {
		return fmt.Errorf("no SVG files found to process")
	}

	// Remove duplicates
	inputFiles = removeDuplicates(inputFiles)

	fmt.Printf("Processing %d SVG files with theme '%s':\n", len(inputFiles), batchFromTheme)

	// Create reverse mapping from color values to Base16 placeholders
	reverseMapping := make(map[string]string)
	for placeholder, colorValue := range selectedTheme.Palette {
		normalizedColor := strings.ToUpper(colorValue)
		reverseMapping[normalizedColor] = placeholder
	}

	successCount := 0
	errorCount := 0

	for _, inputFile := range inputFiles {
		fmt.Printf("\nProcessing: %s\n", inputFile)

		if err := processSingleFile(inputFile, outputDir, reverseMapping, batchOutputSuffix); err != nil {
			fmt.Printf("  Error: %v\n", err)
			errorCount++
		} else {
			fmt.Printf("  Success\n")
			successCount++
		}
	}

	fmt.Printf("\nBatch conversion completed:\n")
	fmt.Printf("  Successful: %d\n", successCount)
	fmt.Printf("  Failed: %d\n", errorCount)
	fmt.Printf("  Total: %d\n", len(inputFiles))

	if errorCount > 0 {
		return fmt.Errorf("batch conversion completed with %d errors", errorCount)
	}

	return nil
}

func processSingleFile(inputFile, outputDir string, reverseMapping map[string]string, suffix string) error {
	// Read input SVG
	content, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read SVG file: %w", err)
	}

	svgContent := string(content)

	// Extract colors from the SVG
	colors := extractColors(svgContent)

	if len(colors) == 0 {
		return fmt.Errorf("no colors found in SVG")
	}

	// Map SVG colors to placeholders
	mapping := make(map[string]string)
	mappedCount := 0

	for _, color := range colors {
		if placeholder, exists := reverseMapping[color]; exists {
			mapping[color] = placeholder
			mappedCount++
		}
	}

	if mappedCount == 0 {
		return fmt.Errorf("no colors from the SVG matched the theme")
	}

	// Apply mappings (case-insensitive)
	for color, placeholder := range mapping {
		upperColor := strings.ToUpper(color)
		lowerColor := strings.ToLower(color)
		placeholderStr := fmt.Sprintf("{{%s}}", placeholder)

		svgContent = strings.ReplaceAll(svgContent, upperColor, placeholderStr)
		svgContent = strings.ReplaceAll(svgContent, lowerColor, placeholderStr)
	}

	// Determine output path
	baseName := filepath.Base(inputFile)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	outputName := nameWithoutExt + suffix + ".svg"
	outputPath := filepath.Join(outputDir, outputName)

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write template
	if err := os.WriteFile(outputPath, []byte(svgContent), 0644); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	fmt.Printf("  Created: %s (mapped %d/%d colors)\n", outputPath, mappedCount, len(colors))
	return nil
}

func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
