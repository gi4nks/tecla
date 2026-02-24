# Tecla Contributor & Agent Guidelines

This document provides technical context for human contributors and AI agents working on Tecla.

## Core Architectural Pillars

1. **Scanner (`/scanner`):** Handles filesystem traversal. Uses `filepath.WalkDir` and respects ignore patterns. Keep it pure; it should only identify paths that *might* be repositories.
2. **Git Info (`/gitinfo`):** The data extraction layer. It executes Git commands via the `Runner` and parses porcelain output.
3. **Runner (`/internal/runner`):** A centralized abstraction for command execution. Use this for all external calls to ensure consistent timeouts and shell handling.
4. **TUI (`/tui`):** Built with the Bubble Tea (ELM architecture). 
   - `model.go`: State management.
   - `view.go`: Lipgloss-based rendering.
   - `ui.go`: Program lifecycle.
5. **Commands (`/cmd`):** Cobra-based CLI definitions. Subcommands should be thin wrappers around the internal packages.

## Development Workflow

- **Testing:** Integration tests are located in `gitinfo/gitinfo_integration_test.go`. They require `git` to be installed on the host. Always run `go test ./...` before submitting changes.
- **Dependency Management:** We use `bubbletea`, `lipgloss`, and `atotto/clipboard`. Avoid adding heavy dependencies unless strictly necessary.
- **CI:** GitHub Actions automatically run tests and `staticcheck` on all PRs.

## Coding Standards

- **Error Handling:** Don't let a single repository error fail the entire scan. Capture errors into the `RepoInfo.Error` field and display them in the UI.
- **Performance:** Use the worker pool in `gitinfo.Collect` for parallel processing. Avoid blocking the main TUI thread.
- **UI Design:** Follow the "Lens" aesthetic: Purple accents (`#7D56F4`), generous margins, and high-contrast title bars.

## Recommendation Engine

When adding new recommendations in `gitinfo.go`:
- Ensure the text is concise.
- Provide a `Command` whenever possible.
- Use placeholders like `<branch-name>` for commands that require user input; the TUI handles substitution automatically.
