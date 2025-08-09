package cmd

import (
	"fmt"
	"sort"

	"github.com/byteowlz/ppr/pkg/config"
	"github.com/byteowlz/ppr/pkg/theme"
	"github.com/spf13/cobra"
)

var listThemesCmd = &cobra.Command{
	Use:   "list-themes",
	Short: "List all available themes",
	Long:  `List all available base16 and base24 themes from the configured themes directory.`,
	RunE:  runListThemes,
}

var (
	showDetails   bool
	filterVariant string
)

func init() {
	listThemesCmd.Flags().BoolVarP(&showDetails, "details", "d", false, "Show theme details")
	listThemesCmd.Flags().StringVarP(&filterVariant, "variant", "v", "", "Filter by variant (dark/light)")
}

func runListThemes(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	themeManager := theme.NewThemeManager(cfg.ThemesPath)
	if err := themeManager.LoadThemes(); err != nil {
		return fmt.Errorf("failed to load themes: %w", err)
	}

	themeNames := themeManager.ListThemes()
	if len(themeNames) == 0 {
		fmt.Println("No themes found. Make sure your themes directory is configured correctly.")
		fmt.Printf("Themes path: %s\n", cfg.ThemesPath)
		return nil
	}

	sort.Strings(themeNames)

	if showDetails {
		fmt.Printf("Found %d themes:\n\n", len(themeNames))
		for _, name := range themeNames {
			themeInfo, err := themeManager.GetThemeInfo(name)
			if err != nil {
				fmt.Printf("ERROR %s (error: %v)\n", name, err)
				continue
			}

			if filterVariant != "" && themeInfo.Variant != filterVariant {
				continue
			}

			fmt.Printf("%s\n", name)
			fmt.Printf("   Name: %s\n", themeInfo.Name)
			fmt.Printf("   Author: %s\n", themeInfo.Author)
			fmt.Printf("   System: %s\n", themeInfo.System)
			fmt.Printf("   Variant: %s\n", themeInfo.Variant)
			fmt.Printf("   Colors: %d\n", len(themeInfo.Palette))
			fmt.Println()
		}
	} else {
		fmt.Printf("Available themes (%d):\n", len(themeNames))
		for _, name := range themeNames {
			if filterVariant != "" {
				themeInfo, err := themeManager.GetThemeInfo(name)
				if err != nil || themeInfo.Variant != filterVariant {
					continue
				}
			}
			fmt.Printf("  • %s\n", name)
		}
	}

	return nil
}
