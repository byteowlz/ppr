package resolution

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type Resolution struct {
	Width  int
	Height int
}

func (r Resolution) String() string {
	return fmt.Sprintf("%dx%d", r.Width, r.Height)
}

type Detector struct{}

func NewDetector() *Detector {
	return &Detector{}
}

func (d *Detector) GetPrimaryDisplayResolution() (*Resolution, error) {
	switch runtime.GOOS {
	case "darwin":
		return d.getMacOSResolution()
	case "linux":
		return d.getLinuxResolution()
	case "windows":
		return d.getWindowsResolution()
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func (d *Detector) getMacOSResolution() (*Resolution, error) {
	cmd := exec.Command("system_profiler", "SPDisplaysDataType")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get display info: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Resolution:") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				widthStr := parts[1]
				heightStr := parts[3]

				width, err := strconv.Atoi(widthStr)
				if err != nil {
					continue
				}

				height, err := strconv.Atoi(heightStr)
				if err != nil {
					continue
				}

				return &Resolution{Width: width, Height: height}, nil
			}
		}
	}

	return &Resolution{Width: 1920, Height: 1080}, nil
}

func (d *Detector) getLinuxResolution() (*Resolution, error) {
	cmd := exec.Command("xrandr")
	output, err := cmd.Output()
	if err != nil {
		return d.getLinuxResolutionFallback()
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, " connected primary") ||
			(strings.Contains(line, " connected") && !strings.Contains(line, "disconnected")) {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.Contains(part, "x") && strings.Contains(part, "+") {
					resPart := strings.Split(part, "+")[0]
					if dims := strings.Split(resPart, "x"); len(dims) == 2 {
						width, err1 := strconv.Atoi(dims[0])
						height, err2 := strconv.Atoi(dims[1])
						if err1 == nil && err2 == nil {
							return &Resolution{Width: width, Height: height}, nil
						}
					}
				}
			}
		}
	}

	return d.getLinuxResolutionFallback()
}

func (d *Detector) getLinuxResolutionFallback() (*Resolution, error) {
	cmd := exec.Command("xdpyinfo")
	output, err := cmd.Output()
	if err != nil {
		return &Resolution{Width: 1920, Height: 1080}, nil
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "dimensions:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				dims := strings.Split(parts[1], "x")
				if len(dims) == 2 {
					width, err1 := strconv.Atoi(dims[0])
					height, err2 := strconv.Atoi(dims[1])
					if err1 == nil && err2 == nil {
						return &Resolution{Width: width, Height: height}, nil
					}
				}
			}
		}
	}

	return &Resolution{Width: 1920, Height: 1080}, nil
}

func (d *Detector) getWindowsResolution() (*Resolution, error) {
	cmd := exec.Command("wmic", "path", "Win32_VideoController", "get", "CurrentHorizontalResolution,CurrentVerticalResolution", "/format:value")
	output, err := cmd.Output()
	if err != nil {
		return &Resolution{Width: 1920, Height: 1080}, nil
	}

	var width, height int
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "CurrentHorizontalResolution=") {
			widthStr := strings.TrimPrefix(line, "CurrentHorizontalResolution=")
			if w, err := strconv.Atoi(widthStr); err == nil {
				width = w
			}
		} else if strings.HasPrefix(line, "CurrentVerticalResolution=") {
			heightStr := strings.TrimPrefix(line, "CurrentVerticalResolution=")
			if h, err := strconv.Atoi(heightStr); err == nil {
				height = h
			}
		}
	}

	if width > 0 && height > 0 {
		return &Resolution{Width: width, Height: height}, nil
	}

	return &Resolution{Width: 1920, Height: 1080}, nil
}

func ParseResolution(resStr string) (*Resolution, error) {
	parts := strings.Split(resStr, "x")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid resolution format: %s (expected WIDTHxHEIGHT)", resStr)
	}

	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid width: %s", parts[0])
	}

	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid height: %s", parts[1])
	}

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("resolution must be positive: %dx%d", width, height)
	}

	return &Resolution{Width: width, Height: height}, nil
}
