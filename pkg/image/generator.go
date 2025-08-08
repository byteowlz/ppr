package image

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) SVGToPNG(svgContent string, width, height int, outputPath string) error {
	icon, err := oksvg.ReadIconStream(strings.NewReader(svgContent))
	if err != nil {
		return fmt.Errorf("failed to parse SVG: %w", err)
	}

	// Extract original SVG dimensions
	svgWidth, svgHeight, err := g.extractSVGDimensions(svgContent)
	if err != nil {
		return fmt.Errorf("failed to extract SVG dimensions: %w", err)
	}

	// Calculate scaling to maintain aspect ratio
	scaleX := float64(width) / float64(svgWidth)
	scaleY := float64(height) / float64(svgHeight)
	scale := scaleX
	if scaleY > scaleX {
		scale = scaleY
	}

	// Calculate scaled dimensions
	scaledWidth := int(float64(svgWidth) * scale)
	scaledHeight := int(float64(svgHeight) * scale)

	// Set target to scaled dimensions to maintain aspect ratio
	icon.SetTarget(0, 0, float64(scaledWidth), float64(scaledHeight))

	// Create image with scaled dimensions
	scaledRGBA := image.NewRGBA(image.Rect(0, 0, scaledWidth, scaledHeight))

	scanner := rasterx.NewScannerGV(scaledWidth, scaledHeight, scaledRGBA, scaledRGBA.Bounds())
	raster := rasterx.NewDasher(scaledWidth, scaledHeight, scanner)

	icon.Draw(raster, 1.0)

	// Crop to target dimensions
	finalRGBA := image.NewRGBA(image.Rect(0, 0, width, height))

	// Calculate crop offset to center the image
	offsetX := (scaledWidth - width) / 2
	offsetY := (scaledHeight - height) / 2

	// Copy the cropped portion
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := x + offsetX
			srcY := y + offsetY
			if srcX >= 0 && srcX < scaledWidth && srcY >= 0 && srcY < scaledHeight {
				finalRGBA.Set(x, y, scaledRGBA.At(srcX, srcY))
			}
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, finalRGBA); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	return nil
}

func (g *Generator) GenerateWallpaper(svgContent string, width, height int, outputPath string) error {
	return g.SVGToPNG(svgContent, width, height, outputPath)
}

func (g *Generator) extractSVGDimensions(svgContent string) (int, int, error) {
	// Look for width and height attributes in the SVG tag
	widthRegex := regexp.MustCompile(`width="(\d+)"`)
	heightRegex := regexp.MustCompile(`height="(\d+)"`)

	widthMatch := widthRegex.FindStringSubmatch(svgContent)
	heightMatch := heightRegex.FindStringSubmatch(svgContent)

	if len(widthMatch) < 2 || len(heightMatch) < 2 {
		return 0, 0, fmt.Errorf("could not find width or height attributes in SVG")
	}

	width, err := strconv.Atoi(widthMatch[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid width value: %w", err)
	}

	height, err := strconv.Atoi(heightMatch[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid height value: %w", err)
	}

	return width, height, nil
}
