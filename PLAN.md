# Tecla Evolution Plan

This document tracks the implementation of new features to transform Tecla from a "viewer" into a "Development Cockpit".

## Phase 1: The Foundation (Profiles & Health)
- [x] **Workspace Profiles**: Allow users to define profiles (e.g., `Work`, `Personal`, `Feature-X`) in the configuration, grouping specific root paths together.
  - [x] Update `internal/config/config.go` to support profiles.
  - [x] Add CLI flag or command to switch/set profiles.
  - [x] Add a TUI shortcut (e.g., `p`) to cycle through or select profiles.
- [x] **Smart Health Score**: Compute a health score for each repository based on metrics (unpushed commits, detached heads, stale branches, untracked files).
  - [x] Add scoring logic to `gitinfo`.
  - [x] Display the health score in the TUI (e.g., using a color-coded bar or an icon from 🟢 to 🔴).

## Phase 2: The Action (Interactive Doctor & Remote)
- [x] **Interactive Doctor Mode**: Add a feature to automatically fix common issues (e.g., deleting merged branches).
  - [x] Add "Doctor" recommendations to `gitinfo`.
  - [x] Extend the command runner to handle interactive prompts or predefined batch fixes.
  - [x] Add a `fix` flag to the `check` command (partially implemented via TUI).
  - [x] Add a `x` or `f` (fix) shortcut in the TUI to run remediation actions.
- [x] **Remote Awareness**: Fetch CI/CD status and open PRs.
  - [x] Integrate with GitHub/GitLab CLI or APIs.
  - [x] Asynchronously fetch remote status to avoid blocking the TUI.
  - [x] **Multi-Remote Support**: Track CI and PRs for all configured remotes (origin, upstream, etc.).
  - [x] Display CI status in the TUI.

## Phase 3: The Intelligence (Impact Analysis)
- [x] **Dependency Mapper**: Scan configuration files (`go.mod`, `package.json`, etc.) to map local dependencies between repositories.
- [x] **TUI Impact View**: When selecting a repository, visually highlight other repositories in the workspace that depend on it.

## Phase 4: Workflow Enhancements
- [ ] **AI-Powered Semantic Search** (Optional/Future): Add semantic search for commit messages.

## Execution Log
- *[Date]*: Created initial plan.
- *[Date]*: Completed Phase 1 (Profiles & Health Score).
- *[Date]*: Completed Phase 2 (Interactive Doctor & Remote Awareness).
- *[Date]*: Completed Phase 3 (Impact Analysis).
