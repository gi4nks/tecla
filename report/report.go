package report

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gi4nks/tecla/gitinfo"
)

type Report struct {
	Roots       []string           `json:"roots"`
	GeneratedAt time.Time          `json:"generated_at"`
	Repos       []gitinfo.RepoInfo `json:"repos"`
	ScanErrors  []error            `json:"-"`
}

func (r Report) MarshalJSON() ([]byte, error) {
	type Alias Report
	var scanErrors []string
	for _, e := range r.ScanErrors {
		scanErrors = append(scanErrors, e.Error())
	}
	return json.Marshal(&struct {
		ScanErrors []string `json:"scan_errors,omitempty"`
		Alias
	}{
		ScanErrors: scanErrors,
		Alias:      (Alias)(r),
	})
}

func RenderTable(w io.Writer, report Report) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "REPO\tBRANCH\tSTATE\tSTAGED\tAHEAD/BEHIND\tUPSTREAM\tREMOTE\tSTASH\tSUBMODULES\tLAST COMMIT\tACTIONS\tERROR")
	for _, repo := range report.Repos {
		repoPath := repoPathSummaryMulti(report.Roots, repo.Path)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			shorten(repoPath, 40),
			branchSummary(repo),
			statusSummary(repo.Status),
			boolSummary(repo.Status.Staged),
			shorten(aheadBehindSummary(repo), 12),
			shorten(emptyDash(repo.Upstream), 24),
			shorten(emptyDash(repo.Remote), 40),
			stashSummary(repo.StashCount),
			submoduleSummary(repo.Submodules),
			timeAgo(repo.LastCommitAt),
			actionsCountSummary(repo.Recommendations),
			errorFlagSummary(strings.TrimSpace(repo.Error)),
		)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if err := renderActionsMulti(w, report.Roots, report.Repos); err != nil {
		return err
	}
	if err := renderRepoErrorsMulti(w, report.Roots, report.Repos); err != nil {
		return err
	}
	return renderScanErrors(w, report.ScanErrors)
}

func RenderJSON(w io.Writer, report Report) error {
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(payload))
	return err
}

func RenderMarkdown(w io.Writer, report Report) error {
	fmt.Fprintln(w, "# tecla scan report")
	fmt.Fprintf(w, "- Roots: `%s`\n", strings.Join(report.Roots, ", "))
	fmt.Fprintf(w, "- Generated: `%s`\n\n", report.GeneratedAt.Format(time.RFC3339))

	fmt.Fprintln(w, "| Repo | Branch | State | Staged | Ahead/Behind | Upstream | Remote | Stash | Submodules | Last Commit | Actions | Error |")
	fmt.Fprintln(w, "| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |")
	for _, repo := range report.Repos {
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			escapeMarkdown(repo.Path),
			escapeMarkdown(branchSummary(repo)),
			escapeMarkdown(statusSummary(repo.Status)),
			escapeMarkdown(boolSummary(repo.Status.Staged)),
			escapeMarkdown(aheadBehindSummary(repo)),
			escapeMarkdown(emptyDash(repo.Upstream)),
			escapeMarkdown(emptyDash(repo.Remote)),
			escapeMarkdown(stashSummary(repo.StashCount)),
			escapeMarkdown(submoduleSummary(repo.Submodules)),
			escapeMarkdown(timeAgo(repo.LastCommitAt)),
			escapeMarkdown(actionsSummaryMarkdown(repo.Recommendations)),
			escapeMarkdown(emptyDash(strings.TrimSpace(repo.Error))),
		)
	}

	return renderScanErrorsMarkdown(w, report.ScanErrors)
}

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	duration := time.Since(t)
	days := int(duration.Hours() / 24)
	if days == 0 {
		return "today"
	}
	if days == 1 {
		return "yesterday"
	}
	if days < 30 {
		return fmt.Sprintf("%d days ago", days)
	}
	months := days / 30
	if months == 1 {
		return "1 month ago"
	}
	if months < 12 {
		return fmt.Sprintf("%d months ago", months)
	}
	years := months / 12
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

func statusSummary(status gitinfo.StatusInfo) string {
	if status.Clean {
		return "clean"
	}
	var parts []string
	if status.Modified {
		parts = append(parts, "modified")
	}
	if status.Untracked {
		parts = append(parts, "untracked")
	}
	if len(parts) == 0 && status.Staged {
		parts = append(parts, "staged")
	}
	return strings.Join(parts, "+")
}

func boolSummary(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func stashSummary(count int) string {
	if count == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", count)
}

func submoduleSummary(info gitinfo.SubmoduleInfo) string {
	if info.Count == 0 {
		return "-"
	}
	if info.Dirty {
		return fmt.Sprintf("%d dirty", info.Count)
	}
	return fmt.Sprintf("%d clean", info.Count)
}

func branchSummary(repo gitinfo.RepoInfo) string {
	if !repo.IsRepo {
		return "[naked]"
	}
	if repo.IsEmpty {
		if repo.Branch != "" {
			return fmt.Sprintf("%s (empty)", repo.Branch)
		}
		return "empty"
	}
	if repo.Detached {
		return "DETACHED"
	}
	if repo.Branch == "" {
		return "-"
	}
	return repo.Branch
}

func aheadBehindSummary(repo gitinfo.RepoInfo) string {
	if repo.Upstream == "" {
		return "-"
	}
	return fmt.Sprintf("%d/%d", repo.Ahead, repo.Behind)
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func actionsSummary(actions []gitinfo.Recommendation) string {
	if len(actions) == 0 {
		return "-"
	}
	var texts []string
	for _, a := range actions {
		texts = append(texts, a.Text)
	}
	return strings.Join(texts, "; ")
}

func actionsCountSummary(actions []gitinfo.Recommendation) string {
	if len(actions) == 0 {
		return "-"
	}
	if len(actions) == 1 {
		return "1 action"
	}
	return fmt.Sprintf("%d actions", len(actions))
}

func errorFlagSummary(value string) string {
	if value == "" {
		return "-"
	}
	return "error"
}

func actionsSummaryMarkdown(actions []gitinfo.Recommendation) string {
	if len(actions) == 0 {
		return "-"
	}
	var texts []string
	for _, a := range actions {
		text := a.Text
		if a.Command != "" {
			text = fmt.Sprintf("%s: `%s`", a.Text, a.Command)
		}
		texts = append(texts, text)
	}
	return strings.Join(texts, "<br>")
}

func escapeMarkdown(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", "<br>")
	return value
}

func renderScanErrors(w io.Writer, errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "\nScan errors:"); err != nil {
		return err
	}
	for _, e := range errs {
		if _, err := fmt.Fprintf(w, "- %v\n", e); err != nil {
			return err
		}
	}
	return nil
}

func renderScanErrorsMarkdown(w io.Writer, errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	fmt.Fprintln(w, "\n## Scan Errors")
	for _, e := range errs {
		fmt.Fprintf(w, "- %s\n", escapeMarkdown(e.Error()))
	}
	return nil
}

func renderActionsMulti(w io.Writer, roots []string, repos []gitinfo.RepoInfo) error {
	hasActions := false
	for _, repo := range repos {
		if len(repo.Recommendations) > 0 {
			hasActions = true
			break
		}
	}
	if !hasActions {
		return nil
	}
	if _, err := fmt.Fprintln(w, "\nActions:"); err != nil {
		return err
	}
	for _, repo := range repos {
		if len(repo.Recommendations) == 0 {
			continue
		}
		repoPath := repoPathSummaryMulti(roots, repo.Path)
		if _, err := fmt.Fprintf(w, "- %s\n", repoPath); err != nil {
			return err
		}
		for _, action := range repo.Recommendations {
			text := action.Text
			if action.Command != "" {
				text = fmt.Sprintf("%s: %s", action.Text, action.Command)
			}
			if _, err := fmt.Fprintf(w, "  - %s\n", text); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderRepoErrorsMulti(w io.Writer, roots []string, repos []gitinfo.RepoInfo) error {
	hasErrors := false
	for _, repo := range repos {
		if strings.TrimSpace(repo.Error) != "" {
			hasErrors = true
			break
		}
	}
	if !hasErrors {
		return nil
	}
	if _, err := fmt.Fprintln(w, "\nRepo errors:"); err != nil {
		return err
	}
	for _, repo := range repos {
		errMsg := strings.TrimSpace(repo.Error)
		if errMsg == "" {
			continue
		}
		repoPath := repoPathSummaryMulti(roots, repo.Path)
		if _, err := fmt.Fprintf(w, "- %s: %s\n", repoPath, errMsg); err != nil {
			return err
		}
	}
	return nil
}

func repoPathSummaryMulti(roots []string, path string) string {
	for _, root := range roots {
		rel, err := filepath.Rel(root, path)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return path
}

func repoPathSummary(root string, path string) string {
	return repoPathSummaryMulti([]string{root}, path)
}

func shorten(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}
