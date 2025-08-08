package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ppr",
	Short: "PPR - Programmable Palette Renderer",
	Long: `PPR is a CLI tool for creating themed wallpapers from SVG templates.
It uses base16/base24 color schemes to generate beautiful wallpapers
with customizable resolutions and automatic wallpaper setting.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(listThemesCmd)
	rootCmd.AddCommand(initConfigCmd)
	rootCmd.AddCommand(setWallpaperCmd)
}
