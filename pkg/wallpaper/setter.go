package wallpaper

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"howett.net/plist"
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

	// Update the wallpaper plist so ALL display configurations use this image.
	// This prevents macOS from reverting to a stale wallpaper when monitors are
	// connected/disconnected (each display config gets its own entry in the plist).
	if err := s.updateWallpaperPlistAllDisplays(imagePath); err != nil {
		fmt.Printf("Warning: failed to update wallpaper plist for all displays: %v\n", err)
	}

	// Verify the wallpaper was set by checking current desktop picture
	if err := s.verifyWallpaperSet(imagePath); err != nil {
		fmt.Printf("Warning: wallpaper verification failed: %v\n", err)
	}

	return nil
}

// updateWallpaperPlistAllDisplays reads the macOS wallpaper store plist and updates
// every display configuration entry to use the given image. This fixes the issue where
// connecting/disconnecting monitors causes macOS to revert to a stale wallpaper because
// each display topology gets its own wallpaper config in the plist.
//
// The plist structure (macOS Sequoia) is:
//
//	AllSpacesAndDisplays: $null
//	Displays:
//	  <display-uuid>:
//	    Desktop: { Content: { Choices: [{ Files: [{ relative: "file://..." }], Provider: "..." }] } }
//	    Idle: ...
//	Spaces:
//	  <space-uuid>:
//	    Default:
//	      Desktop: { Content: ... }
//	    Displays:
//	      <display-uuid>:
//	        Desktop: { Content: ... }
//	SystemDefault:
//	  Desktop: { Content: ... }
func (s *Setter) updateWallpaperPlistAllDisplays(imagePath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	plistPath := filepath.Join(homeDir, "Library", "Application Support", "com.apple.wallpaper", "Store", "Index.plist")

	data, err := os.ReadFile(plistPath)
	if err != nil {
		return fmt.Errorf("failed to read wallpaper plist: %w", err)
	}

	var store map[string]interface{}
	if _, err := plist.Unmarshal(data, &store); err != nil {
		return fmt.Errorf("failed to parse wallpaper plist: %w", err)
	}

	// Build the file URL for the image
	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	fileURL := "file://" + absPath

	updated := 0

	// Update SystemDefault (top-level entry with Desktop/Idle directly)
	if sysDefault, ok := store["SystemDefault"].(map[string]interface{}); ok {
		updateDesktopEntry(sysDefault, fileURL)
		updated++
	}

	// Update Displays (flat map of display-uuid -> {Desktop, Idle})
	if displays, ok := store["Displays"].(map[string]interface{}); ok {
		for _, displayVal := range displays {
			if displayMap, ok := displayVal.(map[string]interface{}); ok {
				updateDesktopEntry(displayMap, fileURL)
				updated++
			}
		}
	}

	// Update Spaces (map of space-uuid -> {Default: {Desktop,Idle}, Displays: {uuid: {Desktop,Idle}}})
	if spaces, ok := store["Spaces"].(map[string]interface{}); ok {
		for _, spaceVal := range spaces {
			spaceMap, ok := spaceVal.(map[string]interface{})
			if !ok {
				continue
			}

			// Update the Default entry for this space
			if defaultEntry, ok := spaceMap["Default"].(map[string]interface{}); ok {
				updateDesktopEntry(defaultEntry, fileURL)
				updated++
			}

			// Update each display within this space
			if spaceDisplays, ok := spaceMap["Displays"].(map[string]interface{}); ok {
				for _, displayVal := range spaceDisplays {
					if displayMap, ok := displayVal.(map[string]interface{}); ok {
						updateDesktopEntry(displayMap, fileURL)
						updated++
					}
				}
			}
		}
	}

	// Write back in binary plist format (same as macOS uses)
	outData, err := plist.Marshal(store, plist.BinaryFormat)
	if err != nil {
		return fmt.Errorf("failed to marshal wallpaper plist: %w", err)
	}

	if err := os.WriteFile(plistPath, outData, 0644); err != nil {
		return fmt.Errorf("failed to write wallpaper plist: %w", err)
	}

	fmt.Printf("Updated wallpaper plist: %d entries set to %s\n", updated, filepath.Base(imagePath))
	return nil
}

// updateDesktopEntry updates the Desktop -> Content -> Choices -> Files entry
// within a wallpaper plist display/default entry to point to the given file URL.
func updateDesktopEntry(entry map[string]interface{}, fileURL string) {
	desktop, ok := entry["Desktop"].(map[string]interface{})
	if !ok {
		return
	}
	content, ok := desktop["Content"].(map[string]interface{})
	if !ok {
		return
	}
	choices, ok := content["Choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return
	}
	files, ok := choice["Files"].([]interface{})
	if !ok || len(files) == 0 {
		// Create a files entry if none exists
		choice["Files"] = []interface{}{
			map[string]interface{}{
				"relative": fileURL,
			},
		}
		choice["Provider"] = "com.apple.wallpaper.choice.image"
		return
	}
	fileEntry, ok := files[0].(map[string]interface{})
	if !ok {
		return
	}
	fileEntry["relative"] = fileURL
	choice["Provider"] = "com.apple.wallpaper.choice.image"
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
	case "hyprland":
		return s.setHyprlandWallpaper(imagePath)
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
	if s.isHyprland() {
		return "hyprland"
	}
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

func (s *Setter) isHyprland() bool {
	if !s.commandExists("hyprctl") {
		return false
	}
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	return strings.Contains(desktop, "hyprland") || strings.Contains(desktop, "hypr")
}

// setHyprlandWallpaper applies a wallpaper on a Hyprland session by relaunching
// swaybg detached from the user session, matching Omarchy's wallpaper behavior.
func (s *Setter) setHyprlandWallpaper(imagePath string) error {
	if s.commandExists("swaybg") {
		// Stop any existing swaybg instance so only one stays alive.
		if err := exec.Command("pkill", "-x", "swaybg").Run(); err != nil {
			fmt.Printf("Note: no running swaybg to stop: %v\n", err)
		}

		args := []string{"swaybg", "-i", imagePath, "-m", "fill"}
		if s.commandExists("uwsm-app") {
			args = append([]string{"--"}, args...)
			args = append([]string{"uwsm-app"}, args...)
		}
		var cmd *exec.Cmd
		if s.commandExists("setsid") {
			cmd = exec.Command("setsid", args...)
		} else {
			cmd = exec.Command(args[0], args[1:]...)
		}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to set wallpaper with swaybg: %w", err)
		}
		return nil
	}

	if s.commandExists("hyprctl") {
		cmd := exec.Command("hyprctl", "hyprpaper", "reload")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to reload hyprpaper: %w", err)
		}
		return nil
	}

	return fmt.Errorf("no suitable wallpaper setter found for Hyprland (tried swaybg, hyprctl)")
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
