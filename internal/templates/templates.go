package templates

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed assets/config.yaml assets/base/*.md assets/blocks/*/*.md
var assets embed.FS

func Scaffold(targetRoot string) error {
	paths := []string{
		"assets/config.yaml",
		"assets/base/global.md",
		"assets/blocks/intro/senior-dev.md",
		"assets/blocks/communication/concise.md",
		"assets/blocks/communication/detailed.md",
		"assets/blocks/tools/dev-tools.md",
		"assets/blocks/tasks/feature.md",
		"assets/blocks/tasks/review.md",
		"assets/blocks/tasks/bugfix.md",
	}

	for _, assetPath := range paths {
		info, err := fs.Stat(assets, assetPath)
		if err != nil {
			return err
		}
		if info.IsDir() {
			continue
		}

		relPath, err := filepath.Rel("assets", assetPath)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetRoot, relPath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		if _, err := os.Stat(targetPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}

		data, err := assets.ReadFile(assetPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			return err
		}
	}

	return nil
}
