package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/byteowlz/ppr/pkg/config"
	"github.com/byteowlz/ppr/pkg/templates"
	"github.com/spf13/cobra"
)

var initConfigCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize PPR configuration",
	Long: `Initialize PPR configuration by creating the config file and directory structure.
This will create ~/.config/ppr/ with default settings and required directories.`,
	RunE: runInitConfig,
}

var (
	force bool
)

func init() {
	initConfigCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing configuration")
}

func runInitConfig(cmd *cobra.Command, args []string) error {
	configPath := config.GetConfigPath()
	configDir := config.GetConfigDir()

	if _, err := os.Stat(configPath); err == nil && !force {
		return fmt.Errorf("configuration already exists at %s. Use --force to overwrite", configPath)
	}

	cfg := config.DefaultConfig()

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	fmt.Printf("Configuration initialized at: %s\n", configPath)
	fmt.Printf("Config directory: %s\n", configDir)
	fmt.Printf("Themes directory: %s\n", cfg.ThemesPath)
	fmt.Printf("Templates directory: %s\n", cfg.TemplatesPath)
	fmt.Printf("Output directory: %s\n", cfg.OutputPath)
	fmt.Println()

	if err := templates.CopyEmbeddedTemplates(cfg.TemplatesPath); err != nil {
		fmt.Printf("Warning: failed to copy example templates: %v\n", err)
	} else {
		fmt.Println("Example templates copied to templates directory")
	}

	if err := createExampleConfig(cfg.ThemesPath); err != nil {
		fmt.Printf("Warning: failed to create example theme symlink: %v\n", err)
	} else {
		fmt.Println("Symlink created to existing themes directory")
	}

	fmt.Println()
	fmt.Println("PPR is ready to use! Try:")
	fmt.Println("   ppr list-themes")
	fmt.Println("   ppr generate --theme nord --template geometric-simple.svg")

	return nil
}

func createExampleConfig(themesDir string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	sourceThemesDir := filepath.Join(cwd, "themes")
	if _, err := os.Stat(sourceThemesDir); os.IsNotExist(err) {
		return fmt.Errorf("source themes directory not found: %s", sourceThemesDir)
	}

	if _, err := os.Stat(themesDir); os.IsNotExist(err) {
		return os.Symlink(sourceThemesDir, themesDir)
	}

	return nil
}
