package image

import (
	"fmt"
	"image"
	"image/png"
	"os"
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

	icon.SetTarget(0, 0, float64(width), float64(height))

	rgba := image.NewRGBA(image.Rect(0, 0, width, height))

	scanner := rasterx.NewScannerGV(width, height, rgba, rgba.Bounds())
	raster := rasterx.NewDasher(width, height, scanner)

	icon.Draw(raster, 1.0)

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, rgba); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	return nil
}

func (g *Generator) GenerateWallpaper(svgContent string, width, height int, outputPath string) error {
	return g.SVGToPNG(svgContent, width, height, outputPath)
}
