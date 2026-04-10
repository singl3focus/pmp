# pmp Roadmap

`pmp` is already a working early MVP: prompt assembly works, project/global config works, TUI exists, and output supports clipboard, stdout, file, and JSON.

This roadmap is organized by product priority rather than release numbers.

## Planning Principles

- Keep the engine as the single source of truth for both CLI and TUI.
- Prefer additive config evolution over breaking changes.
- Improve interactive composition before broad integrations.
- Treat presets and block metadata as first-class product features.

## Current Gaps

- The TUI is still a guided wizard rather than a rich composition environment.
- Preset management is minimal and override behavior is coarse.
- Block metadata such as `weight` and `hidden` is parsed but underused.
- Template context is intentionally small.
- There is no editor integration yet.

## Must-Have

Goal: close the biggest usability gaps in daily prompt authoring.

Scope:

- Improve `pmp ui` with a richer preview viewport.
- Support multi-line message editing in the TUI.
- Support reordering extra blocks in the TUI.
- Make block selection more usable with metadata-aware sorting and filtering.
- Expose `hidden` blocks consistently in UI and listing behavior.
- Add CLI preset management commands such as `preset list`, `preset show`, `preset add`, and `preset delete`.

Done when:

- A user can complete the full authoring flow in `pmp ui` without dropping back to manual config edits.
- The TUI preview behaves predictably for long prompts.
- Block ordering and visibility rules are understandable from the UI.
- Basic preset lifecycle can be handled from the CLI.

## Next

Goal: make configuration more expressive and reduce duplication across project and global roots.

Scope:

- Add preset inheritance with `extends`.
- Merge preset `default_vars` intentionally instead of replacing everything wholesale.
- Extend `doctor` to validate preset inheritance, missing variables, and broken references.
- Extend `list` output to show origin, visibility, and ordering metadata where useful.
- Add more built-in template variables such as timestamp, working directory, repo name, and optional git branch.
- Document safe template-context rules so dynamic values stay predictable.

Done when:

- Common global-to-project customization no longer requires copy-pasting whole presets.
- `doctor` catches the most likely configuration mistakes before build time.
- Template variables are more useful while remaining explicit and safe.

## Later

Goal: make `pmp` fit naturally into a broader authoring workflow beyond the terminal.

Scope:

- Add first editor integration, starting with VS Code or a simple command bridge.
- Support import/export or sharing of preset and block libraries.
- Add optional remote or team-shared block packs.
- Improve composition controls for advanced flows, such as more flexible insertion points or rule-based selection.

Done when:

- A user can trigger `pmp` from their editor with a workflow that feels native enough for daily use.
- Teams can share reusable prompt assets without manually copying files between repositories.
- Advanced composition features remain understandable and do not fragment the engine model.

## Not Prioritized Yet

- Cloud sync or hosted accounts.
- Complex GUI outside the terminal/editor workflow.
- Model-specific prompt optimization logic beyond token-awareness.
- Large plugin systems before the preset and block model is fully mature.

## Suggested Implementation Order

1. TUI viewport and multi-line editing
2. TUI block reordering and metadata-aware block UX
3. Preset CLI management
4. Preset inheritance and merge rules
5. Stronger `doctor` validation
6. Richer template context
7. Editor integration
8. Shared block library workflows

## How to Use This Document

Use this roadmap as a prioritization guide, not as a rigid release contract. If real usage shows that editor integration matters sooner than richer config, move it up. If TUI usage stays low, shift investment back toward CLI and config ergonomics.
