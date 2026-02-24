# CI/CD Integration

Tecla's `check` command is optimized for headless environments like GitHub Actions, GitLab CI, or local pre-commit hooks. It allows you to enforce repository hygiene across large workspaces.

## The `check` Command

The command returns an exit code `1` if any issues are found, and `0` otherwise.

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--dirty` | Fail if a repository has uncommitted changes. | `true` |
| `--behind` | Fail if a repository is behind its upstream branch. | `false` |
| `--root` | Root path(s) to scan. | `.` |
| `--max-depth` | Limit how deep the scanner goes. | `-1` (Unlimited) |

## Example: GitHub Action

You can use Tecla to ensure that all repositories in a monorepo or a collection of submodules are in a clean state.

```yaml
name: Workspace Health Check

on: [push, pull_request]

jobs:
  health:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install Tecla
        run: go install github.com/gi4nks/tecla/cmd/tecla@latest
        
      - name: Run Health Check
        run: tecla check --root . --dirty --behind
```

## Example: Local Pre-push Hook

Add this to `.git/hooks/pre-push` to ensure you don't have lingering dirty repos in your project tree before pushing.

```bash
#!/bin/bash
tecla check --root ~/Projects/my-org --dirty
if [ $? -ne 0 ]; then
  echo "Error: Some repositories are in a dirty state. Please clean them up before pushing."
  exit 1
fi
```
