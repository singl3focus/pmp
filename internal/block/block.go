package block

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Block struct {
	Path        string
	Name        string
	Category    string
	Title       string
	Description string
	Tags        []string
	Weight      int
	Hidden      bool
	Template    *bool
	Content     string
	Source      string
}

// NeedsRender reports whether the block should be processed through
// text/template. When front matter sets template: true/false explicitly,
// that value wins. Otherwise the block is rendered only when its content
// contains the "{{ ." pattern, avoiding accidental breakage of blocks
// that mention Go/Helm/Actions template syntax as literal text.
func (b Block) NeedsRender() bool {
	if b.Template != nil {
		return *b.Template
	}
	return strings.Contains(b.Content, "{{ .")
}

type Root struct {
	Dir    string
	Source string
}

type frontMatter struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Weight      int      `yaml:"weight"`
	Hidden      bool     `yaml:"hidden"`
	Template    *bool    `yaml:"template"`
}

func Resolve(relPath string, roots []Root) (Block, error) {
	relPath, err := normalizeRelativePath(relPath)
	if err != nil {
		return Block{}, err
	}
	for i := len(roots) - 1; i >= 0; i-- {
		root := roots[i]
		if root.Dir == "" {
			continue
		}

		absPath, err := resolveWithinRoot(root.Dir, relPath)
		if err != nil {
			return Block{}, err
		}
		if _, err := os.Stat(absPath); err == nil {
			return LoadFile(absPath, relPath, root.Source)
		}
	}
	return Block{}, fmt.Errorf("block %q not found", relPath)
}

func LoadMerged(roots []Root) (map[string]Block, error) {
	merged := map[string]Block{}
	for _, root := range roots {
		if root.Dir == "" {
			continue
		}
		if _, err := os.Stat(root.Dir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		err := filepath.WalkDir(root.Dir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".md" {
				return nil
			}

			relPath, err := filepath.Rel(root.Dir, path)
			if err != nil {
				return err
			}

			item, err := LoadFile(path, filepath.ToSlash(relPath), root.Source)
			if err != nil {
				return err
			}
			merged[item.Path] = item
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return merged, nil
}

func SortedPaths(blocks map[string]Block) []string {
	paths := make([]string, 0, len(blocks))
	for path := range blocks {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func LoadFile(absPath, relPath, source string) (Block, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return Block{}, err
	}

	meta, content, err := splitFrontMatter(data)
	if err != nil {
		return Block{}, fmt.Errorf("%s: %w", relPath, err)
	}

	category := ""
	if idx := strings.Index(relPath, "/"); idx > 0 {
		category = relPath[:idx]
	}

	return Block{
		Path:        relPath,
		Name:        strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath)),
		Category:    category,
		Title:       meta.Title,
		Description: meta.Description,
		Tags:        append([]string(nil), meta.Tags...),
		Weight:      meta.Weight,
		Hidden:      meta.Hidden,
		Template:    meta.Template,
		Content:     strings.TrimSpace(content),
		Source:      source,
	}, nil
}

func splitFrontMatter(data []byte) (frontMatter, string, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	trimmed := bytes.TrimSpace(data)
	if !bytes.HasPrefix(trimmed, []byte("---\n")) && !bytes.HasPrefix(trimmed, []byte("---\r\n")) {
		return frontMatter{}, string(data), nil
	}

	text := string(data)
	lines := strings.Split(text, "\n")
	if len(lines) < 3 {
		return frontMatter{}, "", fmt.Errorf("incomplete front matter")
	}

	var metaLines []string
	end := -1
	for idx := 1; idx < len(lines); idx++ {
		if strings.TrimSpace(lines[idx]) == "---" {
			end = idx
			break
		}
		metaLines = append(metaLines, lines[idx])
	}
	if end == -1 {
		return frontMatter{}, "", fmt.Errorf("front matter is missing closing delimiter")
	}

	var meta frontMatter
	if err := yaml.Unmarshal([]byte(strings.Join(metaLines, "\n")), &meta); err != nil {
		return frontMatter{}, "", fmt.Errorf("invalid front matter: %w", err)
	}

	body := strings.Join(lines[end+1:], "\n")
	return meta, body, nil
}

func normalizeRelativePath(relPath string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(relPath)))
	switch {
	case clean == "", clean == ".":
		return "", fmt.Errorf("block path is required")
	case filepath.IsAbs(clean):
		return "", fmt.Errorf("block %q must be relative to the block root", relPath)
	case clean == "..":
		return "", fmt.Errorf("block %q escapes the block root", relPath)
	case strings.HasPrefix(clean, ".."+string(filepath.Separator)):
		return "", fmt.Errorf("block %q escapes the block root", relPath)
	default:
		return filepath.ToSlash(clean), nil
	}
}

func resolveWithinRoot(rootDir, relPath string) (string, error) {
	rootDir = filepath.Clean(rootDir)
	absPath := filepath.Clean(filepath.Join(rootDir, filepath.FromSlash(relPath)))
	relToRoot, err := filepath.Rel(rootDir, absPath)
	if err != nil {
		return "", err
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("block %q escapes the block root", relPath)
	}
	return absPath, nil
}
