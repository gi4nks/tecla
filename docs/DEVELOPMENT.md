# Development Guide

This document provides information for developers who want to contribute to Tecla or build it from source.

## Prerequisites

- **Go:** 1.24+ (Recommended 1.25)
- **Git:** Required for scanning repositories.
- **Make:** Used for managing build tasks.

## Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/gi4nks/tecla.git
   cd tecla
   ```

2. Download dependencies:
   ```bash
   go mod download
   ```

## Build Tasks

The project uses a `Makefile` to simplify common tasks.

- **Build the binary:**
  ```bash
  make build
  ```
  The binary will be created as `./tecla` in the root directory.

- **Run unit tests:**
  ```bash
  make test
  ```

- **Run all tests (including integration):**
  ```bash
  make test-all
  ```

- **Install locally:**
  ```bash
  make install
  ```
  By default, this installs to `~/.local/bin/tecla`. You can override the prefix: `make install PREFIX=/usr/local/bin`.

- **Lint the code:**
  ```bash
  golangci-lint run
  ```

## Project Structure

- `main.go`: Entry point of the application.
- `cmd/`: Cobra command definitions (root, scan, ui, check, etc.).
- `gitinfo/`: Logic for interacting with Git and parsing repository status.
- `scanner/`: Parallelized file system traversal and repository discovery.
- `internal/runner/`: Centralized shell command execution.
- `internal/config/`: Configuration file management.
- `tui/`: Bubble Tea models and views for the terminal interface.
- `report/`: Formatting logic for different output types (table, markdown, JSON).

## Versioning

Tecla uses Git tags for versioning. During build, the `Makefile` injects the version, commit hash, and build date using linker flags:

- `github.com/gi4nks/tecla/cmd.Version`
- `github.com/gi4nks/tecla/cmd.GitCommit`
- `github.com/gi4nks/tecla/cmd.BuildDate`

## Docker

You can build and run Tecla using Docker:

```bash
docker build -t tecla .
docker run -it -v $(pwd):/workspace tecla ui --root /workspace
```

## CI/CD

We use GitHub Actions for continuous integration.
- **CI Workflow:** Runs on every push and PR to `main`. It performs linting, security scanning, and testing.
- **Release Workflow:** Runs when a new tag starting with `v` is pushed. It uses GoReleaser to build and publish binaries.
