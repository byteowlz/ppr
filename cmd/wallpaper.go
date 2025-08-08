package cmd

import (
	"fmt"

	"github.com/byteowlz/ppr/pkg/wallpaper"
	"github.com/spf13/cobra"
)

var setWallpaperCmd = &cobra.Command{
	Use:   "set-wallpaper [image-path]",
	Short: "Set an image as wallpaper",
	Long:  `Set the specified image as the desktop wallpaper. Works cross-platform.`,
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
