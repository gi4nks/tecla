package gitinfo

import (
	"strconv"
	"strings"
)

type branchInfo struct {
	Head      string
	Upstream  string
	Ahead     int
	Behind    int
	IsInitial bool
}

func parsePorcelainV2(output string) (StatusInfo, branchInfo) {
	status := StatusInfo{}
	branch := branchInfo{}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# branch.") {
			parseBranchLine(line, &branch)
			continue
		}

		switch line[0] {
		case '1', '2', 'u':
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			xy := fields[1]
			if len(xy) >= 1 && xy[0] != '.' {
				status.Staged = true
			}
			if len(xy) >= 2 && xy[1] != '.' {
				status.Modified = true
			}
		case '?':
			status.Untracked = true
		}
	}

	if !status.Staged && !status.Modified && !status.Untracked {
		status.Clean = true
	}

	return status, branch
}

func parseBranchLine(line string, branch *branchInfo) {
	if strings.HasPrefix(line, "# branch.head ") {
		branch.Head = strings.TrimSpace(strings.TrimPrefix(line, "# branch.head "))
		return
	}
	if strings.HasPrefix(line, "# branch.upstream ") {
		branch.Upstream = strings.TrimSpace(strings.TrimPrefix(line, "# branch.upstream "))
		return
	}
	if strings.HasPrefix(line, "# branch.oid ") {
		oid := strings.TrimSpace(strings.TrimPrefix(line, "# branch.oid "))
		if strings.Contains(oid, "initial") {
			branch.IsInitial = true
		}
		return
	}
	if strings.HasPrefix(line, "# branch.ab ") {
		fields := strings.Fields(strings.TrimPrefix(line, "# branch.ab "))
		for _, field := range fields {
			if strings.HasPrefix(field, "+") {
				branch.Ahead = parseInt(strings.TrimPrefix(field, "+"))
			}
			if strings.HasPrefix(field, "-") {
				branch.Behind = parseInt(strings.TrimPrefix(field, "-"))
			}
		}
		return
	}
}

func parseInt(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
