package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/byteowlz/ppr/pkg/config"
	"github.com/spf13/cobra"
)

var listTemplatesCmd = &cobra.Command{
	Use:   "list-templates",
	Short: "List all available SVG templates",
	Long:  `List all available SVG templates from the configured templates directory.`,
	RunE:  runListTemplates,
}

var (
	showTemplateDetails bool
)

func init() {
	listTemplatesCmd.Flags().BoolVarP(&showTemplateDetails, "details", "d", false, "Show template details")
}

func runListTemplates(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	templates, err := findTemplates(cfg.TemplatesPath)
	if err != nil {
		return fmt.Errorf("failed to find templates: %w", err)
	}

	if len(templates) == 0 {
		fmt.Println("No templates found. Make sure your templates directory is configured correctly.")
		fmt.Printf("Templates path: %s\n", cfg.TemplatesPath)
		return nil
	}

	sort.Strings(templates)

	if showTemplateDetails {
		fmt.Printf("Found %d templates:\n\n", len(templates))
		for _, template := range templates {
			templatePath := filepath.Join(cfg.TemplatesPath, template)
			info, err := os.Stat(templatePath)
			if err != nil {
				fmt.Printf("❌ %s (error: %v)\n", template, err)
				continue
			}

			fmt.Printf("📄 %s\n", template)
			fmt.Printf("   Size: %d bytes\n", info.Size())
			fmt.Printf("   Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
			fmt.Println()
		}
	} else {
		fmt.Printf("Available templates (%d):\n", len(templates))
		for _, template := range templates {
			fmt.Printf("  • %s\n", template)
		}
	}

	return nil
}

func findTemplates(templatesPath string) ([]string, error) {
	var templates []string

	if _, err := os.Stat(templatesPath); os.IsNotExist(err) {
		return templates, nil
	}

	err := filepath.Walk(templatesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(strings.ToLower(info.Name()), ".svg") {
			relPath, err := filepath.Rel(templatesPath, path)
			if err != nil {
				return err
			}
			templates = append(templates, relPath)
		}

		return nil
	})

	return templates, err
}
