package templates

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed data/*
var templatesFS embed.FS

func CopyEmbeddedTemplates(destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	return fs.WalkDir(templatesFS, "data", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		data, err := templatesFS.ReadFile(path)
		if err != nil {
			return err
		}

		filename := filepath.Base(path)
		destPath := filepath.Join(destDir, filename)

		if _, err := os.Stat(destPath); err == nil {
			return nil
		}

		return os.WriteFile(destPath, data, 0644)
	})
}
