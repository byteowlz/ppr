package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	versionInfo struct {
		version string
		commit  string
		date    string
	}
)

var rootCmd = &cobra.Command{
	Use:   "ppr",
	Short: "PPR - Programmable Palette Renderer",
	Long: `PPR is a CLI tool for creating themed wallpapers from SVG templates.
It uses base16/base24 color schemes to generate beautiful wallpapers
with customizable resolutions and automatic wallpaper setting.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ppr version %s\n", versionInfo.version)
		fmt.Printf("commit: %s\n", versionInfo.commit)
		fmt.Printf("built: %s\n", versionInfo.date)
	},
}

func Execute(version, commit, date string) {
	versionInfo.version = version
	versionInfo.commit = commit
	versionInfo.date = date

	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(listThemesCmd)
	rootCmd.AddCommand(listTemplatesCmd)
	rootCmd.AddCommand(initConfigCmd)
	rootCmd.AddCommand(setWallpaperCmd)
	rootCmd.AddCommand(convertTemplateCmd)
	rootCmd.AddCommand(batchConvertCmd)
	rootCmd.AddCommand(switchCurrentCmd)
	rootCmd.AddCommand(cycleCmd)
	rootCmd.AddCommand(versionCmd)
}
