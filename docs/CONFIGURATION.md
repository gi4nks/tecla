# Configuration Guide

Tecla is highly customizable through a JSON configuration file. By default, it looks for the config at:
- **macOS/Linux:** `~/.config/tecla/config.json`
- **Windows:** `%AppData%	ecla\config.json`

## Configuration Options

| Field | Type | Description |
|-------|------|-------------|
| `ignored_paths` | `[]string` | Specific paths or patterns to always skip during scanning. |
| `default_ignored_dirs` | `[]string` | Directory names to skip (e.g., `node_modules`, `.git`). |
| `stale_threshold_days` | `int` | Number of days since the last commit after which a repo is marked "Stale". |
| `auto_fetch` | `bool` | (Experimental) Automatically perform `git fetch` on startup. |
| `profiles` | `[]Object` | Named groups of root paths for different working contexts. |
| `active_profile` | `string` | The name of the profile currently in use. |
| `custom_recommendations` | `[]Object` | Custom rules to trigger specific recommendations. |

## Workspace Profiles

Profiles allow you to switch between different groups of repositories without passing multiple `--root` flags every time.

### Profile Structure
- `name`: Unique identifier for the profile.
- `roots`: List of directory paths to scan when this profile is active.

### Example
```json
{
  "active_profile": "work",
  "profiles": [
    {
      "name": "work",
      "roots": ["~/Projects/acme", "~/Projects/client-x"]
    },
    {
      "name": "personal",
      "roots": ["~/GitHub/gi4nks", "~/Documents/Notes"]
    }
  ]
}
```

## Custom Recommendations

You can define your own "Doctor" rules to highlight specific states in your repositories.

### Object Structure
- `condition`: The state to check for.
- `text`: The message to display in the TUI.
- `command`: (Optional) The shell command to associate with this recommendation.

### Supported Conditions
- `is_dirty`: Repo has uncommitted changes.
- `is_detached`: Repo is in a detached HEAD state.
- `has_untracked`: Repo has untracked files.
- `is_stale`: Last commit is older than the `stale_threshold_days`.

### Example
```json
{
  "custom_recommendations": [
    {
      "condition": "has_untracked",
      "text": "Found untracked files. Consider adding them to .gitignore",
      "command": "git clean -fd --dry-run"
    }
  ]
}
```

## Default Ignore Lists
If `default_ignored_dirs` is not provided, Tecla defaults to:
`node_modules`, `dist`, `build`, `.cache`, `.venv`, `target`, `.terraform`.
