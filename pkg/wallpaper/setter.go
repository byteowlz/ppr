package wallpaper

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Setter struct{}

func NewSetter() *Setter {
	return &Setter{}
}

func (s *Setter) SetWallpaper(imagePath string) error {
	switch runtime.GOOS {
	case "darwin":
		return s.setMacOSWallpaper(imagePath)
	case "linux":
		return s.setLinuxWallpaper(imagePath)
	case "windows":
		return s.setWindowsWallpaper(imagePath)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func (s *Setter) setMacOSWallpaper(imagePath string) error {
	// Check if file exists and is readable
	if _, err := os.Stat(imagePath); err != nil {
		return fmt.Errorf("wallpaper file not accessible: %w", err)
	}

	// Method 1: Try using System Events for all desktops
	script := fmt.Sprintf(`tell application "System Events"
		tell every desktop
			set picture to "%s"
		end tell
	end tell`, imagePath)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("System Events method failed: %s\n", string(output))

		// Method 2: Fallback to Finder method with POSIX file
		script2 := fmt.Sprintf(`tell application "Finder" to set desktop picture to POSIX file "%s"`, imagePath)
		cmd2 := exec.Command("osascript", "-e", script2)
		output2, err2 := cmd2.CombinedOutput()

		if err2 != nil {
			fmt.Printf("Finder method failed: %s\n", string(output2))
			return fmt.Errorf("both AppleScript methods failed: Finder error: %w, System Events error: %v", err2, err)
		}
	}

	// Force desktop refresh
	refreshCmd := exec.Command("osascript", "-e", `tell application "Finder" to activate`)
	refreshCmd.Run()

	// Verify the wallpaper was set by checking current desktop picture
	if err := s.verifyWallpaperSet(imagePath); err != nil {
		fmt.Printf("Warning: wallpaper verification failed: %v\n", err)
	}

	return nil
}

func (s *Setter) verifyWallpaperSet(expectedPath string) error {
	// Get current desktop picture path
	cmd := exec.Command("osascript", "-e", `tell application "System Events" to get picture of first desktop`)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current desktop picture: %w", err)
	}

	currentPath := string(output)
	currentPath = strings.TrimSpace(currentPath)

	// Remove any quotes that might be around the path
	currentPath = strings.Trim(currentPath, "\"")

	if currentPath != expectedPath {
		return fmt.Errorf("wallpaper verification failed: expected %s, got %s", expectedPath, currentPath)
	}

	fmt.Printf("Wallpaper verification successful: %s\n", currentPath)
	return nil
}

func (s *Setter) setLinuxWallpaper(imagePath string) error {
	desktopEnv := s.detectLinuxDesktopEnvironment()

	switch desktopEnv {
	case "gnome":
		return s.setGnomeWallpaper(imagePath)
	case "kde":
		return s.setKDEWallpaper(imagePath)
	case "xfce":
		return s.setXfceWallpaper(imagePath)
	case "i3", "sway":
		return s.setI3SwayWallpaper(imagePath)
	default:
		return s.setGenericLinuxWallpaper(imagePath)
	}
}

func (s *Setter) detectLinuxDesktopEnvironment() string {
	if s.commandExists("gnome-session") {
		return "gnome"
	}
	if s.commandExists("kwin") || s.commandExists("plasmashell") {
		return "kde"
	}
	if s.commandExists("xfce4-session") {
		return "xfce"
	}
	if s.commandExists("i3") {
		return "i3"
	}
	if s.commandExists("sway") {
		return "sway"
	}
	return "generic"
}

func (s *Setter) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func (s *Setter) setGnomeWallpaper(imagePath string) error {
	cmd := exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", fmt.Sprintf("file://%s", imagePath))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set GNOME wallpaper: %w", err)
	}

	cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", fmt.Sprintf("file://%s", imagePath))
	cmd.Run()

	return nil
}

func (s *Setter) setKDEWallpaper(imagePath string) error {
	script := fmt.Sprintf(`
var allDesktops = desktops();
for (i=0;i<allDesktops.length;i++) {
	d = allDesktops[i];
	d.wallpaperPlugin = "org.kde.image";
	d.currentConfigGroup = Array("Wallpaper", "org.kde.image", "General");
	d.writeConfig("Image", "%s");
}`, imagePath)

	cmd := exec.Command("qdbus", "org.kde.plasmashell", "/PlasmaShell", "org.kde.PlasmaShell.evaluateScript", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set KDE wallpaper: %w", err)
	}

	return nil
}

func (s *Setter) setXfceWallpaper(imagePath string) error {
	cmd := exec.Command("xfconf-query", "-c", "xfce4-desktop", "-p", "/backdrop/screen0/monitor0/workspace0/last-image", "-s", imagePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set XFCE wallpaper: %w", err)
	}

	return nil
}

func (s *Setter) setI3SwayWallpaper(imagePath string) error {
	if s.commandExists("feh") {
		cmd := exec.Command("feh", "--bg-scale", imagePath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set wallpaper with feh: %w", err)
		}
		return nil
	}

	if s.commandExists("swaybg") {
		cmd := exec.Command("swaybg", "-i", imagePath, "-m", "fill")
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to set wallpaper with swaybg: %w", err)
		}
		return nil
	}

	return fmt.Errorf("no suitable wallpaper setter found (tried feh, swaybg)")
}

func (s *Setter) setGenericLinuxWallpaper(imagePath string) error {
	commands := [][]string{
		{"feh", "--bg-scale", imagePath},
		{"nitrogen", "--set-scaled", imagePath},
		{"pcmanfm", "--set-wallpaper", imagePath},
	}

	for _, cmd := range commands {
		if s.commandExists(cmd[0]) {
			if err := exec.Command(cmd[0], cmd[1:]...).Run(); err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("no suitable wallpaper setter found")
}

func (s *Setter) setWindowsWallpaper(imagePath string) error {
	cmd := exec.Command("powershell", "-Command", fmt.Sprintf(`
Add-Type -TypeDefinition "
using System;
using System.Runtime.InteropServices;
public class Wallpaper {
    [DllImport(\"user32.dll\", CharSet=CharSet.Auto)]
    public static extern int SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni);
}
"
[Wallpaper]::SystemParametersInfo(20, 0, "%s", 3)
`, imagePath))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set Windows wallpaper: %w", err)
	}

	return nil
}
