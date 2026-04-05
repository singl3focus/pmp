package engine

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/singl3focus/pmp/internal/block"
	"github.com/singl3focus/pmp/internal/config"
	"github.com/tiktoken-go/tokenizer"
)

type BuildRequest struct {
	PresetName  string
	Message     string
	ExtraBlocks []string
	Vars        map[string]string
	TokenLimit  int
	DryRun      bool
}

type BuildResult struct {
	PresetName      string   `json:"preset_name"`
	Message         string   `json:"message,omitempty"`
	Prompt          string   `json:"prompt"`
	BlocksUsed      []string `json:"blocks_used"`
	EstimatedTokens int      `json:"estimated_tokens"`
	Warnings        []string `json:"warnings,omitempty"`
}

type renderInput struct {
	Vars   map[string]string
	Date   string
	Preset string
}

func Build(req BuildRequest, active config.Active) (BuildResult, error) {
	preset := config.Preset{}
	if req.PresetName != "" {
		var ok bool
		preset, ok = active.Config.Presets[req.PresetName]
		if !ok {
			return BuildResult{}, fmt.Errorf("preset %q not found", req.PresetName)
		}
	}

	var ordered []block.Block
	var used []string
	seen := map[string]struct{}{}

	appendBlock := func(path string, roots []block.Root) error {
		item, err := block.Resolve(path, roots)
		if err != nil {
			return err
		}
		if _, ok := seen[item.Path]; ok {
			return nil
		}
		seen[item.Path] = struct{}{}
		ordered = append(ordered, item)
		used = append(used, item.Path)
		return nil
	}

	for _, path := range active.Config.Base.AlwaysInclude {
		if err := appendBlock(path, active.BaseRoots()); err != nil {
			return BuildResult{}, err
		}
	}
	for _, path := range preset.Blocks {
		if err := appendBlock(path, active.BlockRoots()); err != nil {
			return BuildResult{}, err
		}
	}
	for _, path := range req.ExtraBlocks {
		if err := appendBlock(path, active.BlockRoots()); err != nil {
			return BuildResult{}, err
		}
	}

	data := renderInput{
		Vars:   mergeVars(preset.DefaultVars, req.Vars),
		Date:   time.Now().Format("2006-01-02"),
		Preset: req.PresetName,
	}

	msg := strings.TrimSpace(req.Message)

	var parts []string
	if msg != "" && active.Config.MessagePosition == config.MessagePositionTop {
		parts = append(parts, msg)
	}

	for _, item := range ordered {
		text := item.Content
		if item.NeedsRender() {
			var err error
			text, err = render(text, data, item.Path)
			if err != nil {
				return BuildResult{}, err
			}
		}
		if text = strings.TrimSpace(text); text != "" {
			parts = append(parts, text)
		}
	}

	if msg != "" && active.Config.MessagePosition != config.MessagePositionTop {
		parts = append(parts, msg)
	}

	prompt := strings.Join(parts, active.Config.Separator)
	result := BuildResult{
		PresetName:      req.PresetName,
		Message:         strings.TrimSpace(req.Message),
		Prompt:          prompt,
		BlocksUsed:      used,
		EstimatedTokens: countTokens(prompt),
	}

	threshold := active.Config.TokenWarningThreshold
	if req.TokenLimit > 0 {
		threshold = req.TokenLimit
	}
	if threshold > 0 && result.EstimatedTokens > threshold {
		result.Warnings = append(result.Warnings, fmt.Sprintf("estimated tokens %d exceed limit %d", result.EstimatedTokens, threshold))
	}

	return result, nil
}

func render(content string, data renderInput, name string) (string, error) {
	tpl, err := template.New(filepath.ToSlash(name)).Option("missingkey=error").Parse(content)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}
	return buf.String(), nil
}

func mergeVars(base, override map[string]string) map[string]string {
	result := map[string]string{}
	for key, value := range base {
		result[key] = value
	}
	for key, value := range override {
		result[key] = value
	}
	return result
}

var codec, _ = tokenizer.Get(tokenizer.Cl100kBase)

func countTokens(text string) int {
	if text == "" {
		return 0
	}
	if codec != nil {
		if n, err := codec.Count(text); err == nil {
			return n
		}
	}
	// Fallback: heuristic estimation when codec is unavailable.
	charCount := len([]rune(text))
	wordCount := len(strings.Fields(text))
	estimate := charCount / 4
	if estimate < wordCount {
		return wordCount
	}
	return estimate
}
