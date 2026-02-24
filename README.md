# Tecla 🎹

**Tecla** (Spanish for "key") is a fast, multi-root Git repository management utility. It scans your workspaces to discover repositories and provides a high-signal overview of their health, status, and recommended next steps.

Whether you're managing a single project or a directory with hundreds of microservices, Tecla helps you keep your local environment synchronized and clean.

[![CI](https://github.com/gi4nks/tecla/actions/workflows/ci.yml/badge.svg)](https://github.com/gi4nks/tecla/actions/workflows/ci.yml)
[![Release](https://github.com/gi4nks/tecla/actions/workflows/release.yml/badge.svg)](https://github.com/gi4nks/tecla/actions/workflows/release.yml)

## ✨ Key Features

- **Multi-Root Scanning:** Traversal of multiple disparate directory roots simultaneously.
- **High-Performance:** Parallelized Git inspection using a configurable worker pool.
- **Interactive TUI:** A modern terminal interface built with Bubble Tea for browsing, filtering, and managing repos.
- **Recommendation Engine:** Intelligent advice on common Git workflows (syncing, stashing, rebasing).
- **Batch Operations:** Execute commands across multiple selected repositories at once.
- **CI/CD Integration:** A headless `check` mode for automated health diagnostics.
- **Customizable:** Extensible via a JSON configuration file for custom ignore patterns and recommendation rules.
- **Clipboard Support:** Quickly copy recommended commands to your clipboard.

## 🚀 Quick Start

### Installation

```bash
go install github.com/gi4nks/tecla@latest
```

### Usage

```bash
# Launch the interactive UI in the current directory
tecla

# Scan specific directories
tecla ui --root ~/Projects --root ~/Work

# Headless scan with report generation
tecla scan --root ~/Projects --format markdown --output report.md

# CI/CD health check
tecla check --root ~/Projects --dirty --behind

# Permanently ignore a path
tecla ignore ./heavy-folder
```

## 🖥️ Interactive TUI Shortcuts

| Key | Action |
|-----|--------|
| `j`/`k` or `↑`/`↓` | Move selection cursor |
| `Enter` | View full repository details |
| `/` | Filter repositories by name, branch, or remote |
| `s` | Cycle through sort modes (Name, Dirty, Workspace) |
| `Space` or `m` | Toggle selection for batch operations |
| `b` | Execute a batch command on selected repositories |
| `f` | Fetch updates for all repositories (`git fetch --all`) |
| `r` | Rescan the workspace |
| `e` | View scanner error logs |
| `Esc` or `q` | Back / Quit |

**In Detail Mode:**
- `Tab`: Cycle through recommended actions.
- `a`: Apply (execute) the selected recommendation.
- `c`: Copy the recommendation command to clipboard.

## ⚙️ Configuration

Tecla searches for a configuration file at `~/.config/tecla/config.json`.

```json
{
  "ignored_paths": ["backup/"],
  "default_ignored_dirs": ["node_modules", "dist", "build", "vendor"],
  "stale_threshold_days": 30,
  "auto_fetch": false,
  "custom_recommendations": [
    {
      "condition": "is_detached",
      "text": "Careful! You are in a detached HEAD state",
      "command": "git switch main"
    }
  ]
}
```

See [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for full details on customizing Tecla.

## 🤖 CI/CD Integration

The `check` command is designed for automation. It exits with a non-zero code if any repository fails your health criteria.

```bash
# Fail if any repo has uncommitted changes or is behind upstream
tecla check --root . --dirty --behind
```

See [docs/CI_CD.md](docs/CI_CD.md) for integration examples.

## 🛠️ Development

For detailed information on building, testing, and contributing, see [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).

### Quick Build
```bash
make build
```

## 📄 License
This project is licensed under the Apache License 2.0.
