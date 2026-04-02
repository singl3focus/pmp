# Final Plan: pmp

## 1. Product Summary

`pmp` is a CLI-first prompt builder for assembling high-quality LLM prompts from reusable Markdown blocks.

The goal is not just to concatenate text files, but to give the user a fast, predictable, and extensible way to build prompts as composable artifacts:
- reusable
- structured
- project-aware
- token-conscious
- pleasant to use every day

The product should be strong in two dimensions at once:
- functional for daily engineering work
- high-quality in UX, both in CLI and TUI modes

---

## 2. Problem Statement

Today the user assembles prompts manually:
- copies fragments from multiple `.md` files
- reuses similar task instructions by hand
- repeats the same role/style/tooling/context blocks
- has no reliable way to preview the final result
- loses time on formatting and order
- cannot easily standardize prompt quality across tasks and projects

This creates several concrete problems:
- slow prompt creation
- inconsistent quality
- duplicated effort
- weak reuse of prompt patterns
- poor visibility into prompt size and composition

`pmp` solves this by making prompt construction a first-class workflow.

---

## 3. Product Principles

1. CLI-first
   The main user flow must be solvable in one command.

2. Convention over configuration
   The system should work with a simple directory structure and sensible defaults.

3. Local overrides global
   Project-specific prompt logic must be able to override global defaults cleanly.

4. Progressive power
   The simple use case should stay simple, while advanced features remain opt-in.

5. Safe extensibility
   Dynamic context and hooks should be introduced carefully and only with explicit control.

6. Strong UX
   The tool should explain what it is doing, what it assembled, and why something failed.

7. Portable output
   Prompt output must be usable in clipboard, terminal, files, and scripts.

---

## 4. Core User Scenarios

### 4.1 Fast daily assembly

The user wants a prompt immediately:

```bash
pmp --preset feature -m "Добавить профили в карточки товаров"
```

Expected result:
- prompt assembled
- copied to clipboard by default
- summary shown in terminal
- token estimate shown

### 4.2 Build with additional context

```bash
pmp --preset review --block security,performance -m "Проверь auth flow"
```

Expected result:
- preset blocks included
- extra blocks appended in a predictable order
- duplicate blocks not repeated

### 4.3 Interactive assembly

```bash
pmp ui
```

Expected result:
- search and select blocks
- preview final prompt
- see token estimate
- save selection as preset if needed

### 4.4 Script usage

```bash
pmp --preset bugfix -m "Исправить race condition" --json
```

Expected result:
- machine-readable output
- usable from scripts, editors, launchers, automations

### 4.5 Project-specific prompting

The user works inside a repository with its own `.pmp/` and expects local blocks and presets to override global ones.

### 4.6 Diagnostics

```bash
pmp doctor
```

Expected result:
- detect missing config
- detect broken block references
- detect clipboard issues
- explain resolution steps

---

## 5. Product Scope

### 5.1 In scope for V1

- assemble prompts from Markdown blocks
- presets from config
- local and global config resolution
- clipboard/stdout/file outputs
- dry-run mode
- token estimate
- interactive TUI builder
- diagnostics command
- testable core engine

### 5.2 Out of scope for initial release

- cloud sync
- GUI desktop app
- web app
- online prompt registry
- collaborative editing
- unrestricted external shell hooks
- complex domain-specific language for templates

---

## 6. UX Model

The tool should support two top-level modes.

### 6.1 CLI mode

Primary command:

```bash
pmp --preset feature -m "Добавить профили"
```

Short alias:

```bash
pmp -p feature -m "Добавить профили"
```

This is the default productivity path.

Optional explicit form:

```bash
pmp build --preset feature -m "Добавить профили"
```

The explicit `build` command may exist for readability and scripting, but it is not the primary path.

### 6.2 TUI mode

Interactive builder:

```bash
pmp ui
```

This is the precision path for:
- discovering blocks
- exploring categories and tags
- previewing prompt composition
- saving reusable prompt sets

### 6.3 UX standards

Every successful build should display:
- selected preset
- number of included blocks
- output target
- token estimate
- warnings if any

Every failure should display:
- what failed
- where it failed
- how to fix it

---

## 7. Information Architecture

Two levels of configuration are supported.

### 7.1 Global directory

```text
~/.pmp/
  config.yaml
  base/
    global.md
  blocks/
    intro/
    communication/
    tools/
    tasks/
    custom/
```

### 7.2 Project directory

```text
./.pmp/
  config.yaml
  base/
  blocks/
```

### 7.3 Priority rules

Resolution order:
1. project config and files from `./.pmp/`
2. global config and files from `~/.pmp/`

Override rule:
- if the same relative block path exists in both places, the project-local version wins

This gives strong project customizability without breaking global defaults.

---

## 8. Data Model

### 8.1 Block

A block is a Markdown file with optional front matter.

Example:

```markdown
---
title: Senior Go Developer
description: Production-grade Go engineering persona
tags: [go, backend, senior]
weight: 10
hidden: false
---
Ты senior Go-разработчик. Пиши production-ready код, объясняй архитектурные решения кратко и по делу.
```

Block model:

```go
type Block struct {
    Path        string
    Name        string
    Category    string
    Title       string
    Description string
    Tags        []string
    Weight      int
    Hidden      bool
    Content     string
    Source      string
}
```

Notes:
- `Path` is the relative logical path, for example `intro/senior-dev.md`
- `Source` indicates whether the winning block came from `project` or `global`
- front matter is optional

### 8.2 Preset

A preset defines a reusable prompt recipe.

```go
type Preset struct {
    Name        string
    Description string
    Blocks      []string
    DefaultVars map[string]string
    Output      OutputDefaults
}
```

### 8.3 Build request

```go
type BuildRequest struct {
    PresetName   string
    Message      string
    ExtraBlocks  []string
    Vars         map[string]string
    TokenLimit   int
    NoCopy       bool
    OutFile      string
    JSON         bool
    DryRun       bool
}
```

### 8.4 Build result

```go
type BuildResult struct {
    Prompt         string
    BlocksUsed      []string
    EstimatedTokens int
    Warnings        []string
    OutputMode      string
}
```

---

## 9. Config Format

Recommended `config.yaml`:

```yaml
version: 1

separator: "\n\n"
copy_by_default: true
token_warning_threshold: 24000

base:
  always_include:
    - global.md

presets:
  feature:
    description: "New feature implementation"
    blocks:
      - intro/senior-dev.md
      - communication/concise.md
      - tools/dev-tools.md
      - tasks/feature.md

  review:
    description: "Code review"
    blocks:
      - intro/senior-dev.md
      - communication/detailed.md
      - tasks/review.md

  bugfix:
    description: "Bug fixing"
    blocks:
      - intro/senior-dev.md
      - communication/concise.md
      - tasks/bugfix.md
```

Optional future extension:

```yaml
vars:
  language: ru
  repo_style: concise
```

Config rules:
- stay human-editable
- avoid deep nesting unless it brings real value
- preserve backward compatibility when possible

---

## 10. Build Engine

The engine should run in five stages.

### 10.1 Resolve

Find and load:
- active config
- preset
- base blocks
- extra blocks
- variables
- output settings

Responsibilities:
- merge local and global state
- validate block references
- detect missing preset

### 10.2 Compose

Assemble final block order:
1. base blocks
2. preset blocks
3. extra blocks from CLI/TUI
4. user message at the top of final prompt

Message placement rule:
- `message` goes first in the final prompt because it represents the immediate task request

### 10.3 Render

Render final text with `text/template`.

Initial supported values:
- `.Vars`
- `.Date`
- `.Preset`

Initial rendering rules:
- fail clearly on invalid templates
- do not silently swallow template errors
- no arbitrary command execution in V1

### 10.4 Post-process

Apply:
- trim extra whitespace
- normalize separators
- deduplicate repeated block paths
- estimate token count
- produce warnings for size thresholds

### 10.5 Output

Supported outputs:
- clipboard
- stdout
- file
- json envelope

Priority:
- if `--out` is set, write file
- if `--no-copy` is set, print to stdout
- otherwise copy to clipboard
- `--json` should wrap metadata and prompt in machine-readable form

---

## 11. Command Design

### 11.1 Main command

```bash
pmp --preset feature -m "Task"
```

Supported flags:

```text
-p, --preset <name>
-m, --message <text>
--block <a,b,c>
--var <key=value>
--token-limit <n>
--dry-run
--no-copy
--out <file>
--json
```

### 11.2 Root alias behavior

The root command should perform prompt assembly when build flags are present.

Examples:
- `pmp --preset feature -m "Task"`
- `pmp -p feature -m "Task"`

Design rule:
- prompt assembly is the default root action
- `build` may remain as an explicit alias or subcommand
- users should not be forced to type `build` for the common case

This preserves a fast UX while keeping the command model understandable.

### 11.3 Other commands

```text
pmp build
pmp ui
pmp list
pmp init
pmp doctor
pmp version
```

### 11.4 Command purposes

`pmp list`
- show presets
- show blocks
- show descriptions
- support filtering later

`pmp init`
- scaffold starter config
- scaffold example blocks
- support global and project mode

`pmp doctor`
- validate config discovery
- validate preset references
- validate block parsing
- validate clipboard availability

`pmp version`
- show semantic version and build metadata if available

---

## 12. TUI Design

The TUI should be more than a checkbox list.

### 12.1 Layout

Recommended layout:
- left pane: categories, search, block list
- right pane: live prompt preview
- bottom bar: token estimate, warnings, actions

### 12.2 User capabilities

- search by name, path, description, tags
- filter by category
- select and unselect blocks
- reorder selected blocks if needed
- toggle preview
- enter message
- save selection as preset

### 12.3 UX requirements

- keyboard-first navigation
- clear selection state
- immediate preview updates
- obvious exit and confirm actions
- readable token warnings

### 12.4 TUI release priority

The TUI should ship after the core engine is stable, but still be part of the near-term product, not a vague future idea.

---

## 13. Token Awareness

Token awareness is a product feature, not just an implementation detail.

V1 requirements:
- estimate token count on every build
- show token estimate in CLI result
- show live estimate in TUI
- warn when threshold is exceeded

Future improvements:
- model-specific budgets
- budget presets
- prompt trimming suggestions

Implementation note:
- V1 can use a practical estimate
- exact model-specific counting can come later

---

## 14. Smart Context Strategy

This is a major differentiator, but it must be introduced carefully.

### 14.1 Not in MVP

Do not include in initial release:
- arbitrary shell execution from templates
- unrestricted file reads
- auto-including full git diff without safeguards

### 14.2 Phase 2 safe additions

Allowed candidates:
- current date
- current branch
- repository name
- staged diff summary
- explicit file include by opt-in helper

### 14.3 Safety model

Any dynamic context feature must be:
- explicit
- inspectable
- limited in scope
- easy to disable

---

## 15. Technical Architecture

Recommended repository structure:

```text
cmd/
  pmp/
    main.go

cli/
  root.go
  build.go
  ui.go
  init.go
  list.go
  doctor.go
  version.go

internal/
  block/
    block.go
    block_test.go
  config/
    config.go
    config_test.go
  preset/
    preset.go
    preset_test.go
  engine/
    engine.go
    engine_test.go
  render/
    render.go
    render_test.go
  output/
    output.go
    output_test.go
  clipboard/
    clipboard.go
  interactive/
    interactive.go
  doctor/
    doctor.go

templates/
  init/
    config.yaml
    blocks/...
```

### 15.1 Responsibility split

`block`
- load blocks
- parse front matter
- normalize metadata

`config`
- discover config paths
- merge global and project config

`preset`
- resolve preset definitions

`engine`
- compose build request into final ordered material

`render`
- render templates safely

`output`
- send prompt to clipboard/stdout/file/json

`interactive`
- Bubble Tea TUI application

`doctor`
- validation and diagnostics

---

## 16. Dependencies

Recommended dependencies:

```text
github.com/spf13/cobra
github.com/spf13/viper
gopkg.in/yaml.v3
github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles
github.com/atotto/clipboard
github.com/stretchr/testify
```

Notes:
- `cobra` is appropriate for command structure
- `viper` is acceptable for config, though the code should keep config logic isolated
- clipboard access should be wrapped behind an internal interface for testability

---

## 17. Release Phases

### Phase 1: Core MVP

Deliver:
- root build flow via `pmp --preset ...`
- optional explicit `build`
- `list`
- `init`
- `version`
- local/global config discovery
- block loading
- preset loading
- assembly engine
- clipboard/stdout/file outputs
- dry-run
- token estimate
- unit tests

Success criteria:
- user can install, init, assemble, and reuse prompts immediately

### Phase 2: UX Release

Deliver:
- `ui`
- front matter metadata in block browsing
- search and filter
- live preview
- token warnings in TUI
- save preset from UI
- `doctor`

Success criteria:
- the product feels polished, not just functional

### Phase 3: Smart Context

Deliver:
- template vars
- safe built-in context functions
- token budget configuration
- context warnings

Success criteria:
- advanced users gain power without losing predictability

### Phase 4: Extensibility

Deliver:
- `--json`
- better script/editor integration
- model-aware token profiles
- optional integration surfaces

Success criteria:
- `pmp` becomes an ecosystem-friendly tool, not only a personal CLI

---

## 18. Error Handling Requirements

The product should never fail with vague messages when a more useful explanation is possible.

Expected error classes:
- config not found
- preset not found
- block not found
- invalid front matter
- invalid template
- clipboard failure
- write failure for `--out`

Error output should include:
- what happened
- affected file or preset
- suggested next action

Example:

```text
Preset "review" references missing block "tasks/review.md".
Checked:
- ./.pmp/blocks/tasks/review.md
- ~/.pmp/blocks/tasks/review.md
Fix the preset or create the missing block.
```

---

## 19. Observability and Diagnostics

`pmp doctor` should report:
- resolved config path
- whether project config is active
- available presets
- broken preset references
- unreadable or malformed blocks
- clipboard availability

Optional later:
- `--verbose`
- timing breakdown
- debug mode for resolution decisions

---

## 20. Testing Strategy

### 20.1 Unit tests

Must cover:
- front matter parsing
- block discovery
- config resolution
- preset loading
- composition ordering
- deduplication
- output routing
- template render behavior

### 20.2 Integration tests

Must cover:
- build using project config
- fallback to global config
- dry-run flow
- write to file
- JSON output

### 20.3 Manual verification

Must verify:
- clipboard behavior on target OS
- TUI usability
- init scaffolding
- doctor diagnostics

---

## 21. Acceptance Criteria

The first meaningful release is approved if all of the following are true:

1. A user can run `pmp init` and get a usable setup.
2. A user can assemble prompts with one command.
3. Local project config cleanly overrides global config.
4. The build output is predictable and easy to inspect.
5. Token estimate is visible on every build.
6. Errors are actionable.
7. The core engine is covered by tests.
8. The TUI materially improves discovery and composition.

---

## 22. Implementation Priorities

Recommended build order:

1. project skeleton and root CLI
2. config discovery and merge model
3. block loading and parsing
4. preset resolution
5. engine composition
6. output modes
7. dry-run and token estimate
8. init scaffolding
9. list command
10. tests
11. TUI
12. doctor
13. smart context

This order minimizes risk and keeps the product shippable throughout development.

---

## 23. Product Decisions to Lock

The following decisions are part of the final plan:

1. Primary command shape:
   `pmp --preset <name> -m "<task>"`

2. Fast alias:
   `pmp -p <name> -m "<task>"`

3. Explicit command support:
   `pmp build --preset <name> -m "<task>"` may exist, but only as a secondary form

4. Config hierarchy:
   `./.pmp/` overrides `~/.pmp/`

5. Message placement:
   `message` goes at the top of the final prompt

6. Output default:
   clipboard unless overridden

7. TUI position:
   important product surface, but follows core engine stabilization

8. Smart context:
   phased, safe, explicit, never arbitrary by default

---

## 24. Why This Plan Is Stronger

This plan keeps the strongest parts of both earlier proposals:
- from the Gemini direction: composable prompt architecture, staged engine, token awareness, smart-context roadmap
- from the Claude direction: concrete modules, practical commands, config-driven presets, testability, release readiness

It also improves on both:
- stronger command model
- better TUI UX target
- clearer config hierarchy
- better diagnostics
- safer extensibility
- cleaner path from MVP to durable product

The result is a tool that can be useful quickly, but also grow into a polished and differentiated prompt workflow system.
