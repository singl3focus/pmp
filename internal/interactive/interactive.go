package interactive

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/singl3focus/pmp/internal/block"
	"github.com/singl3focus/pmp/internal/config"
	"github.com/singl3focus/pmp/internal/engine"
	"github.com/singl3focus/pmp/internal/output"
)

type step int

const (
	stepPreset step = iota
	stepBlocks
	stepMessage
	stepOutput
)

type blockEntry struct {
	Path        string
	Title       string
	Description string
	Tags        []string
}

type model struct {
	active          config.Active
	step            step
	width           int
	height          int
	presetNames     []string
	presetIndex     int
	blocks          []blockEntry
	blockCursor     int
	selected        map[string]bool
	filter          string
	filterMode      bool
	message         string
	outputIndex     int
	filePath        string
	previewOffset   int
	saveMode        bool
	saveField       int
	saveName        string
	saveDescription string
	statusMessage   string
	result          engine.BuildResult
	buildErr        error
	cancelled       bool
	finished        bool
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	panelStyle  = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
)

func Run(active config.Active, in io.Reader, out io.Writer) error {
	m, err := newModel(active)
	if err != nil {
		return err
	}

	program := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(out))
	finalModel, err := program.Run()
	if err != nil {
		return err
	}

	finishedModel, ok := finalModel.(model)
	if !ok {
		return fmt.Errorf("unexpected interactive model type %T", finalModel)
	}
	if finishedModel.cancelled || !finishedModel.finished {
		return nil
	}

	emittedMode, err := output.Emit(finishedModel.result, outputOptions(finishedModel.outputIndex, finishedModel.filePath))
	if err != nil {
		return err
	}

	if emittedMode != output.ModeStdout {
		_, err = fmt.Fprintf(out, "Output: %s\n", emittedMode)
		return err
	}
	return nil
}

func newModel(active config.Active) (model, error) {
	mergedBlocks, err := block.LoadMerged(active.BlockRoots())
	if err != nil {
		return model{}, err
	}

	presetNames := make([]string, 0, len(active.Config.Presets))
	for name := range active.Config.Presets {
		presetNames = append(presetNames, name)
	}
	sort.Strings(presetNames)

	entries := make([]blockEntry, 0, len(mergedBlocks))
	for _, path := range block.SortedPaths(mergedBlocks) {
		item := mergedBlocks[path]
		entries = append(entries, blockEntry{
			Path:        item.Path,
			Title:       item.Title,
			Description: item.Description,
			Tags:        append([]string(nil), item.Tags...),
		})
	}

	m := model{
		active:      active,
		step:        stepPreset,
		presetNames: append([]string{""}, presetNames...),
		blocks:      entries,
		selected:    map[string]bool{},
		width:       120,
		height:      32,
	}
	m.rebuild()
	return m, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "pgdown", "ctrl+f":
			m.previewOffset += 8
			m.clampPreviewOffset()
			return m, nil
		case "pgup", "ctrl+b":
			m.previewOffset -= 8
			m.clampPreviewOffset()
			return m, nil
		}

		if m.saveMode {
			return m.updateSave(msg)
		}

		switch m.step {
		case stepPreset:
			return m.updatePreset(msg)
		case stepBlocks:
			return m.updateBlocks(msg)
		case stepMessage:
			return m.updateMessage(msg)
		case stepOutput:
			return m.updateOutput(msg)
		}
	}
	return m, nil
}

func (m model) View() string {
	leftWidth := maxInt(42, m.width/3)
	rightWidth := maxInt(50, m.width-leftWidth-4)
	previewHeight := maxInt(12, m.height-8)

	left := panelStyle.Width(leftWidth).Render(m.leftPanel())
	right := panelStyle.Width(rightWidth).Height(previewHeight).Render(m.previewPanel(previewHeight))

	header := titleStyle.Render("pmp ui") + "\n" + helpStyle.Render(m.stepTitle())
	if m.statusMessage != "" {
		header += "\n" + accentStyle.Render(m.statusMessage)
	}
	footer := helpStyle.Render(m.helpText())
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	if m.saveMode {
		body += "\n\n" + panelStyle.Width(maxInt(60, m.width-4)).Render(m.renderSavePreset())
	}
	return header + "\n\n" + body + "\n\n" + footer
}

func (m model) updatePreset(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "up", "k":
		if m.presetIndex > 0 {
			m.presetIndex--
		}
	case "down", "j":
		if m.presetIndex < len(m.presetNames)-1 {
			m.presetIndex++
		}
	case "enter", "right", "tab":
		m.step = stepBlocks
	case "esc", "q":
		m.cancelled = true
		return m, tea.Quit
	}
	m.rebuild()
	return m, nil
}

func (m model) updateBlocks(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filterMode {
		switch key.String() {
		case "esc":
			m.filterMode = false
		case "backspace":
			if m.filter != "" {
				m.filter = deleteLastRune(m.filter)
			}
			m.blockCursor = 0
		case "enter":
			m.filterMode = false
		default:
			if text, ok := keyText(key); ok {
				m.filter += text
				m.blockCursor = 0
			}
		}
		m.rebuild()
		return m, nil
	}

	paths := m.filteredBlocks()
	switch key.String() {
	case "up", "k":
		if m.blockCursor > 0 {
			m.blockCursor--
		}
	case "down", "j":
		if m.blockCursor < len(paths)-1 {
			m.blockCursor++
		}
	case " ":
		if len(paths) > 0 {
			path := paths[m.blockCursor].Path
			if !m.isPresetBlock(path) {
				m.selected[path] = !m.selected[path]
				if !m.selected[path] {
					delete(m.selected, path)
				}
			}
		}
	case "/":
		m.filterMode = true
	case "ctrl+s":
		m.openSaveMode()
	case "backspace":
		if m.filter != "" {
			m.filter = deleteLastRune(m.filter)
			m.blockCursor = 0
		}
	case "tab", "right", "enter":
		m.step = stepMessage
	case "left", "esc":
		m.step = stepPreset
	}

	if len(paths) == 0 {
		m.blockCursor = 0
	} else if m.blockCursor >= len(paths) {
		m.blockCursor = len(paths) - 1
	}
	m.rebuild()
	return m, nil
}

func (m model) updateMessage(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "backspace":
		if m.message != "" {
			m.message = deleteLastRune(m.message)
		}
	case "ctrl+u":
		m.message = ""
	case "ctrl+s":
		m.openSaveMode()
	case "enter", "tab", "right":
		m.step = stepOutput
	case "esc", "left":
		m.step = stepBlocks
	default:
		if text, ok := keyText(key); ok {
			m.message += text
		}
	}
	m.rebuild()
	return m, nil
}

func (m model) updateOutput(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "up", "k":
		if m.outputIndex > 0 {
			m.setOutputIndex(m.outputIndex - 1)
		}
	case "down", "j":
		if m.outputIndex < 2 {
			m.setOutputIndex(m.outputIndex + 1)
		}
	case "backspace":
		if m.outputIndex == 2 && m.filePath != "" {
			m.filePath = deleteLastRune(m.filePath)
		}
	case "enter":
		if m.outputIndex == 2 && strings.TrimSpace(m.filePath) == "" {
			return m, nil
		}
		if m.buildErr != nil {
			m.statusMessage = "Resolve the preview error before confirming output"
			return m, nil
		}
		m.finished = true
		return m, tea.Quit
	case "esc", "left":
		m.step = stepMessage
	case "ctrl+s":
		m.openSaveMode()
	case "q":
		m.cancelled = true
		return m, tea.Quit
	default:
		if m.outputIndex == 2 {
			if text, ok := keyText(key); ok {
				m.filePath += text
			}
		}
	}
	return m, nil
}

func (m model) updateSave(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.saveMode = false
		m.statusMessage = "Preset save cancelled"
	case "tab", "up", "down":
		if m.saveField == 0 {
			m.saveField = 1
		} else {
			m.saveField = 0
		}
	case "backspace":
		if m.saveField == 0 {
			m.saveName = deleteLastRune(m.saveName)
		} else {
			m.saveDescription = deleteLastRune(m.saveDescription)
		}
	case "enter":
		if m.saveField == 0 {
			m.saveField = 1
			return m, nil
		}
		return m.saveCurrentPreset()
	default:
		if text, ok := keyText(key); ok {
			if m.saveField == 0 {
				m.saveName += text
			} else {
				m.saveDescription += text
			}
		}
	}
	return m, nil
}

func (m *model) rebuild() {
	extraBlocks := make([]string, 0, len(m.selected))
	for _, entry := range m.blocks {
		if m.selected[entry.Path] {
			extraBlocks = append(extraBlocks, entry.Path)
		}
	}

	result, err := engine.Build(engine.BuildRequest{
		PresetName:  m.selectedPreset(),
		Message:     m.message,
		ExtraBlocks: extraBlocks,
	}, m.active)
	m.result = result
	m.buildErr = err
	m.clampPreviewOffset()
}

func (m model) selectedPreset() string {
	return m.presetNames[m.presetIndex]
}

func (m model) presetBlockSet() map[string]struct{} {
	result := map[string]struct{}{}
	name := m.selectedPreset()
	if name == "" {
		return result
	}
	for _, path := range m.active.Config.Presets[name].Blocks {
		result[path] = struct{}{}
	}
	return result
}

func (m model) isPresetBlock(path string) bool {
	_, ok := m.presetBlockSet()[path]
	return ok
}

func (m model) filteredBlocks() []blockEntry {
	return filterBlocks(m.blocks, m.filter)
}

func (m *model) openSaveMode() {
	if m.saveMode {
		return
	}
	suggested := m.selectedPreset()
	if suggested == "" {
		suggested = "custom"
	}
	m.saveMode = true
	m.saveField = 0
	m.saveName = suggested
	m.saveDescription = m.currentSaveDescription()
	m.statusMessage = "Save the current preset to config"
}

func (m model) currentSaveDescription() string {
	name := m.selectedPreset()
	if name != "" {
		if preset, ok := m.active.Config.Presets[name]; ok && preset.Description != "" {
			return preset.Description
		}
	}
	return "Saved from pmp ui"
}

func filterBlocks(entries []blockEntry, filter string) []blockEntry {
	filter = strings.TrimSpace(strings.ToLower(filter))
	if filter == "" {
		return entries
	}

	result := make([]blockEntry, 0, len(entries))
	for _, entry := range entries {
		var haystack strings.Builder
		haystack.WriteString(strings.ToLower(entry.Path))
		haystack.WriteString(" ")
		haystack.WriteString(strings.ToLower(entry.Title))
		haystack.WriteString(" ")
		haystack.WriteString(strings.ToLower(entry.Description))
		haystack.WriteString(" ")
		haystack.WriteString(strings.ToLower(strings.Join(entry.Tags, " ")))
		if strings.Contains(haystack.String(), filter) {
			result = append(result, entry)
		}
	}
	return result
}

func (m model) leftPanel() string {
	switch m.step {
	case stepPreset:
		return m.renderPresets()
	case stepBlocks:
		return m.renderBlocks()
	case stepMessage:
		return m.renderMessage()
	case stepOutput:
		return m.renderOutput()
	default:
		return ""
	}
}

func (m model) renderPresets() string {
	var lines []string
	lines = append(lines, accentStyle.Render("Presets"))
	lines = append(lines, "")
	for idx, name := range m.presetNames {
		label := "No preset"
		desc := "Build from base blocks and manual selections only"
		if name != "" {
			label = name
			desc = m.active.Config.Presets[name].Description
		}
		cursor := "  "
		if idx == m.presetIndex {
			cursor = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s", cursor, label))
		lines = append(lines, fmt.Sprintf("  %s", desc))
	}
	return strings.Join(lines, "\n")
}

func (m model) renderBlocks() string {
	paths := m.filteredBlocks()
	presetBlocks := m.presetBlockSet()
	var lines []string
	lines = append(lines, accentStyle.Render("Blocks"))
	lines = append(lines, fmt.Sprintf("Filter: %s", displayInput(m.filter, m.filterMode)))
	lines = append(lines, fmt.Sprintf("Selected extras: %d", len(m.selected)))
	lines = append(lines, "")
	if len(paths) == 0 {
		lines = append(lines, "No blocks match the current filter.")
		return strings.Join(lines, "\n")
	}

	start := maxInt(0, m.blockCursor-8)
	end := minInt(len(paths), start+18)
	for idx := start; idx < end; idx++ {
		entry := paths[idx]
		cursor := "  "
		if idx == m.blockCursor {
			cursor = "> "
		}

		marker := "[ ]"
		if _, ok := presetBlocks[entry.Path]; ok {
			marker = "[p]"
		} else if m.selected[entry.Path] {
			marker = "[x]"
		}

		line := fmt.Sprintf("%s%s %s", cursor, marker, entry.Path)
		lines = append(lines, line)
		if entry.Description != "" {
			lines = append(lines, "    "+entry.Description)
		}
	}
	return strings.Join(lines, "\n")
}

func (m model) renderMessage() string {
	var lines []string
	lines = append(lines, accentStyle.Render("Message"))
	lines = append(lines, "")
	lines = append(lines, displayInput(m.message, true))
	lines = append(lines, "")
	lines = append(lines, "Single-line task prompt. Enter continues to output selection.")
	return strings.Join(lines, "\n")
}

func (m model) renderOutput() string {
	options := []string{"clipboard", "stdout", "file"}
	var lines []string
	lines = append(lines, accentStyle.Render("Output"))
	lines = append(lines, "")
	for idx, option := range options {
		cursor := "  "
		if idx == m.outputIndex {
			cursor = "> "
		}
		lines = append(lines, cursor+option)
	}
	if m.outputIndex == 2 {
		lines = append(lines, "")
		lines = append(lines, "File path:")
		lines = append(lines, displayInput(m.filePath, true))
	}
	return strings.Join(lines, "\n")
}

func (m model) renderSavePreset() string {
	var lines []string
	lines = append(lines, accentStyle.Render("Save Preset"))
	lines = append(lines, "Persist the current preset + extra blocks into the active config.")
	lines = append(lines, "")

	nameCursor := "  "
	descCursor := "  "
	if m.saveField == 0 {
		nameCursor = "> "
	} else {
		descCursor = "> "
	}

	lines = append(lines, nameCursor+"Name:")
	lines = append(lines, "  "+displayInput(m.saveName, m.saveField == 0))
	lines = append(lines, "")
	lines = append(lines, descCursor+"Description:")
	lines = append(lines, "  "+displayInput(m.saveDescription, m.saveField == 1))
	lines = append(lines, "")
	lines = append(lines, helpStyle.Render("enter moves/saves • tab switches fields • esc closes"))
	return strings.Join(lines, "\n")
}

func (m model) previewPanel(height int) string {
	var lines []string
	lines = append(lines, accentStyle.Render("Preview"))
	lines = append(lines, fmt.Sprintf("Preset: %s", displayPreset(m.selectedPreset())))
	lines = append(lines, fmt.Sprintf("Blocks: %d", len(m.result.BlocksUsed)))
	lines = append(lines, fmt.Sprintf("Estimated tokens: %d", m.result.EstimatedTokens))
	if m.buildErr != nil {
		lines = append(lines, "")
		lines = append(lines, errorStyle.Render(m.buildErr.Error()))
		return strings.Join(lines, "\n")
	}
	if len(m.result.Warnings) > 0 {
		lines = append(lines, helpStyle.Render("Warnings: "+strings.Join(m.result.Warnings, "; ")))
	}
	lines = append(lines, helpStyle.Render(fmt.Sprintf("Preview offset: %d", m.previewOffset)))
	lines = append(lines, "")
	lines = append(lines, m.renderPromptPreview(height-8)...)
	return strings.Join(lines, "\n")
}

func (m model) renderPromptPreview(height int) []string {
	prompt := strings.TrimSpace(m.result.Prompt)
	if m.buildErr != nil || prompt == "" {
		return []string{"(empty prompt)"}
	}

	lines := strings.Split(prompt, "\n")
	limit := maxInt(6, height)
	start := minInt(m.previewOffset, maxInt(0, len(lines)-1))
	if start > 0 {
		lines = lines[start:]
	}
	if len(lines) > limit {
		lines = append(lines[:limit], "...")
	}
	return lines
}

func (m model) stepTitle() string {
	switch m.step {
	case stepPreset:
		return "Step 1/4: choose a preset"
	case stepBlocks:
		return "Step 2/4: add extra blocks and filter the library"
	case stepMessage:
		return "Step 3/4: enter the task message"
	case stepOutput:
		return "Step 4/4: choose the output target"
	default:
		return ""
	}
}

func (m model) helpText() string {
	switch m.step {
	case stepPreset:
		return "up/down move • enter continues • q cancels"
	case stepBlocks:
		if m.filterMode {
			return "type to filter • backspace deletes • enter or esc stops filtering • pgup/pgdown scroll preview"
		}
		return "up/down move • space toggles • / starts filter • ctrl+s saves preset • enter continues • esc goes back"
	case stepMessage:
		return "type message • backspace deletes • ctrl+u clears • ctrl+s saves preset • enter continues • esc goes back"
	case stepOutput:
		return "up/down move • file mode accepts typing for path • ctrl+s saves preset • enter confirms • esc goes back"
	default:
		return ""
	}
}

func displayPreset(name string) string {
	if name == "" {
		return "none"
	}
	return name
}

func displayInput(value string, focused bool) string {
	cursor := ""
	if focused {
		cursor = "█"
	}
	if value == "" {
		return cursor
	}
	return value + cursor
}

func deleteLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

func keyText(key tea.KeyMsg) (string, bool) {
	if key.Alt {
		return "", false
	}

	switch key.Type {
	case tea.KeySpace:
		return " ", true
	case tea.KeyRunes:
		return string(key.Runes), len(key.Runes) > 0
	}
	return "", false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func defaultFilePath() string {
	return filepath.Clean("prompt.md")
}

func outputOptions(outputIndex int, filePath string) output.Options {
	switch outputIndex {
	case 1:
		return output.Options{NoCopy: true}
	case 2:
		return output.Options{OutFile: strings.TrimSpace(filePath)}
	default:
		return output.Options{}
	}
}

func (m *model) setOutputIndex(next int) {
	previous := m.outputIndex
	m.outputIndex = next
	if previous != 2 && next == 2 && strings.TrimSpace(m.filePath) == "" {
		m.filePath = defaultFilePath()
	}
}

func (m *model) saveCurrentPreset() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.saveName)
	if name == "" {
		m.statusMessage = "Preset name is required"
		return m, nil
	}

	blocks := m.currentPresetBlocks()
	preset := m.basePresetForSave(name)
	preset.Description = strings.TrimSpace(m.saveDescription)
	preset.Blocks = blocks
	if err := config.SavePreset(m.active, name, preset); err != nil {
		m.statusMessage = "Failed to save preset: " + err.Error()
		return m, nil
	}

	if m.active.Config.Presets == nil {
		m.active.Config.Presets = map[string]config.Preset{}
	}
	m.active.Config.Presets[name] = preset
	m.refreshPresetNames(name)
	m.selected = map[string]bool{}
	m.saveMode = false
	m.statusMessage = fmt.Sprintf("Saved preset %q", name)
	m.rebuild()
	return m, nil
}

func (m model) basePresetForSave(name string) config.Preset {
	if existing, ok := m.active.Config.Presets[name]; ok {
		return clonePreset(existing)
	}
	if selected := m.selectedPreset(); selected != "" {
		if existing, ok := m.active.Config.Presets[selected]; ok {
			return clonePreset(existing)
		}
	}
	return config.Preset{}
}

func clonePreset(preset config.Preset) config.Preset {
	return config.Preset{
		Description: preset.Description,
		Blocks:      append([]string(nil), preset.Blocks...),
		DefaultVars: cloneStringMap(preset.DefaultVars),
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func (m model) currentPresetBlocks() []string {
	seen := map[string]struct{}{}
	var blocks []string
	if presetName := m.selectedPreset(); presetName != "" {
		for _, path := range m.active.Config.Presets[presetName].Blocks {
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			blocks = append(blocks, path)
		}
	}
	for _, entry := range m.blocks {
		if !m.selected[entry.Path] {
			continue
		}
		if _, ok := seen[entry.Path]; ok {
			continue
		}
		seen[entry.Path] = struct{}{}
		blocks = append(blocks, entry.Path)
	}
	return blocks
}

func (m *model) refreshPresetNames(selectName string) {
	names := make([]string, 0, len(m.active.Config.Presets))
	for name := range m.active.Config.Presets {
		names = append(names, name)
	}
	sort.Strings(names)
	m.presetNames = append([]string{""}, names...)
	for idx, name := range m.presetNames {
		if name == selectName {
			m.presetIndex = idx
			return
		}
	}
}

func (m *model) clampPreviewOffset() {
	prompt := strings.TrimSpace(m.result.Prompt)
	if prompt == "" {
		m.previewOffset = 0
		return
	}

	lines := strings.Split(prompt, "\n")
	maxOffset := maxInt(0, len(lines)-1)
	if m.previewOffset < 0 {
		m.previewOffset = 0
	}
	if m.previewOffset > maxOffset {
		m.previewOffset = maxOffset
	}
}
