package svg

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/byteowlz/ppr/pkg/theme"
)

type Processor struct{}

func NewProcessor() *Processor {
	return &Processor{}
}

func (p *Processor) ProcessTemplate(templatePath string, theme *theme.Theme) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %w", err)
	}

	svgContent := string(content)

	for colorKey, colorValue := range theme.Palette {
		placeholder := fmt.Sprintf("{{%s}}", colorKey)
		svgContent = strings.ReplaceAll(svgContent, placeholder, colorValue)
	}

	if err := p.validateProcessedSVG(svgContent); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	return svgContent, nil
}

func (p *Processor) ProcessTemplateWithColors(templatePath string, colors map[string]string) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %w", err)
	}

	svgContent := string(content)

	for colorKey, colorValue := range colors {
		placeholder := fmt.Sprintf("{{%s}}", colorKey)
		svgContent = strings.ReplaceAll(svgContent, placeholder, colorValue)
	}

	if err := p.validateProcessedSVG(svgContent); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	return svgContent, nil
}

func (p *Processor) validateProcessedSVG(content string) error {
	placeholderPattern := regexp.MustCompile(`\{\{base[0-9A-F]{2}\}\}`)
	matches := placeholderPattern.FindAllString(content, -1)

	if len(matches) > 0 {
		return fmt.Errorf("unresolved placeholders found: %v", matches)
	}

	if !strings.Contains(content, "<svg") {
		return fmt.Errorf("invalid SVG content: missing <svg> tag")
	}

	return nil
}

func (p *Processor) ExtractPlaceholders(templatePath string) ([]string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	placeholderPattern := regexp.MustCompile(`\{\{(base[0-9A-F]{2})\}\}`)
	matches := placeholderPattern.FindAllStringSubmatch(string(content), -1)

	placeholderSet := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			placeholderSet[match[1]] = true
		}
	}

	var placeholders []string
	for placeholder := range placeholderSet {
		placeholders = append(placeholders, placeholder)
	}

	return placeholders, nil
}
