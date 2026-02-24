package deps

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ScanRepo scansiona un repository alla ricerca di dipendenze (es. go.mod)
// Restituisce (moduleName, dependencies)
func ScanRepo(path string) (string, []string) {
	var moduleName string
	var deps []string

	// Analisi Go
	goModPath := filepath.Join(path, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		name, d := parseGoMod(goModPath)
		moduleName = name
		deps = append(deps, d...)
	}

	// Analisi Node
	packageJsonPath := filepath.Join(path, "package.json")
	if _, err := os.Stat(packageJsonPath); err == nil {
		name, d := parsePackageJson(packageJsonPath)
		if moduleName == "" {
			moduleName = name
		}
		deps = append(deps, d...)
	}

	return moduleName, unique(deps)
}

func parseGoMod(path string) (string, []string) {
	var moduleName string
	var deps []string
	// #nosec G304 - path is derived from repository scan
	file, err := os.Open(path)
	if err != nil {
		return "", nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "(" || line == ")" {
			continue
		}
		if strings.HasPrefix(line, "module") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				moduleName = parts[1]
			}
			continue
		}

		// Skip requirement lines that are just starting a block
		if line == "require (" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 1 {
			target := parts[0]
			if target == "require" && len(parts) > 1 {
				target = parts[1]
			}

			if strings.Contains(target, ".") && !strings.HasPrefix(target, "//") {
				deps = append(deps, target)
			}
		}
	}
	return moduleName, deps
}

func parsePackageJson(path string) (string, []string) {
	// Simple string-based parsing to avoid heavy JSON dependencies if possible,
	// but let's use a simple heuristic for dependencies
	var name string
		var deps []string
		
		// #nosec G304 - path is derived from repository scan
		data, err := os.ReadFile(path)
		if err != nil {
			return "", nil
		}
	lines := strings.Split(string(data), "\n")
	inDeps := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "\"name\":") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				name = strings.Trim(strings.TrimSpace(parts[1]), "\",")
			}
		}
		if strings.Contains(line, "\"dependencies\":") || strings.Contains(line, "\"devDependencies\":") {
			inDeps = true
			continue
		}
		if inDeps && strings.HasPrefix(line, "}") {
			inDeps = false
			continue
		}
		if inDeps {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				depName := strings.Trim(strings.TrimSpace(parts[0]), "\"")
				if depName != "" {
					deps = append(deps, depName)
				}
			}
		}
	}
	return name, deps
}

func unique(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
