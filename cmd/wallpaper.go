package cmd

import (
	"fmt"

	"github.com/byteowlz/ppr/pkg/wallpaper"
	"github.com/spf13/cobra"
)

var setWallpaperCmd = &cobra.Command{
	Use:   "from-file <image-path>",
	Short: "Set wallpaper from an existing image file",
	Long:  `Set the desktop wallpaper from an existing image file (PNG, JPG, etc.).`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSetWallpaper,
}

func runSetWallpaper(cmd *cobra.Command, args []string) error {
	imagePath := args[0]

	setter := wallpaper.NewSetter()
	if err := setter.SetWallpaper(imagePath); err != nil {
		return fmt.Errorf("failed to set wallpaper: %w", err)
	}

	fmt.Printf("✅ Wallpaper set successfully: %s\n", imagePath)
	return nil
}
