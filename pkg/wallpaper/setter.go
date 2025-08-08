package wallpaper

import (
	"fmt"
	"os/exec"
	"runtime"
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
	script := fmt.Sprintf(`tell application "Finder" to set desktop picture to POSIX file "%s"`, imagePath)
	cmd := exec.Command("osascript", "-e", script)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set macOS wallpaper: %w", err)
	}

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
