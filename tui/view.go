package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gi4nks/tecla/gitinfo"
	"github.com/muesli/reflow/truncate"
)

var (
	docStyle   = lipgloss.NewStyle().Margin(1, 2)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Bold(true)
	styleAccent = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleClean  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green
	styleDirty  = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow
	styleError  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red
	styleNaked  = lipgloss.NewStyle().Foreground(lipgloss.Color("13")) // Magenta
	styleAction = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	styleSelect = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	styleLabel  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))

	styleHealthHigh = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green
	styleHealthMid  = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow
	styleHealthLow  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red

	styleCIsuc  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))                 // Green
	styleCIfail = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))                  // Red
	styleCIpend = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))                 // Yellow
	styleImpact = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87")).Bold(true) // Pink/Salmon
)

func (m model) mainView() string {
	var s strings.Builder

	s.WriteString(m.headerLine())
	s.WriteString("\n")
	s.WriteString(m.gitConfigLine())
	s.WriteString("\n")
	s.WriteString(m.legendLine())
	s.WriteString("\n\n")

	if m.mode == modeFilter {
		s.WriteString(fmt.Sprintf("Filter: %s\n\n", m.filterInput.View()))
	} else if m.mode == modeInput {
		s.WriteString(fmt.Sprintf("Enter value for %s: %s\n\n", styleAccent.Render(m.inputPlaceholder), m.filterInput.View()))
	} else if m.filter != "" {
		s.WriteString(fmt.Sprintf("Filter: %s\n\n", m.filter))
	}

	headerLinesCount := 5 // title + legend + spacing + filter/input
	if m.mode != modeFilter && m.mode != modeInput && m.filter == "" {
		headerLinesCount = 3
	}

	footerLines := m.footerLines()
	bodyHeight := m.height - headerLinesCount - len(footerLines) - 2 // extra padding
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	leftWidth, rightWidth := splitWidths(m.width - 4) // adjust for docStyle margin
	leftLines := m.listView(bodyHeight, leftWidth)
	rightLines := m.detailView(bodyHeight, rightWidth)
	body := joinColumns(leftLines, rightLines, leftWidth, rightWidth)

	for _, line := range body {
		s.WriteString(line)
		s.WriteString("\n")
	}

	s.WriteString("\n")
	for _, line := range footerLines {
		s.WriteString(line)
		s.WriteString("\n")
	}

	return docStyle.Render(s.String())
}

func (m model) headerLine() string {
	var rootNames []string
	for _, root := range m.opts.Roots {
		base := filepath.Base(root)
		if base == "." || base == "" {
			base = root
		}
		rootNames = append(rootNames, base)
	}
	rootsStr := strings.Join(rootNames, ", ")
	repoCount := len(m.repos)
	title := titleStyle.Render("Tecla - Repository Explorer")
	line := fmt.Sprintf("%s  Roots: %s  Repos: %d  Sort: %s", title, styleDim.Render(rootsStr), repoCount, styleDim.Render(sortLabel(m.sortMode)))
	if m.loading {
		line = fmt.Sprintf("%s  Scanning %s", line, styleAccent.Render(m.spinner.View()))
	}
	return line
}

func (m model) gitConfigLine() string {
	cfg := m.globalConfig
	if cfg.UserName == "" && cfg.UserEmail == "" && cfg.Version == "" {
		return ""
	}

	parts := []string{}
	if cfg.UserName != "" {
		parts = append(parts, fmt.Sprintf("%s %s", styleLabel.Render("User:"), styleDim.Render(cfg.UserName)))
	}
	if cfg.UserEmail != "" {
		parts = append(parts, fmt.Sprintf("%s %s", styleLabel.Render("Email:"), styleDim.Render(cfg.UserEmail)))
	}
	if cfg.Version != "" {
		parts = append(parts, fmt.Sprintf("%s %s", styleLabel.Render("Git:"), styleDim.Render(cfg.Version)))
	}

	return strings.Join(parts, "  ")
}

func (m model) legendLine() string {
	legend := fmt.Sprintf(
		"Status: %s clean  %s dirty  %s error  %s potential",
		styleClean.Render("C"),
		styleDirty.Render("D"),
		styleError.Render("E"),
		styleNaked.Render("P"),
	)
	return styleDim.Render(legend)
}

func (m model) footerLines() []string {
	var keys string
	width := maxInt(0, m.width-4)

	isSmall := width < 80

	switch m.mode {
	case modeDetail:
		if isSmall {
			keys = "esc:back | tab:sel | a:apply | c:copy"
		} else {
			keys = "esc/q:back | up/down:scroll | tab:select | a:apply | c:copy"
		}
	case modeErrors:
		keys = "esc:back | j/k:move"
	default:
		if isSmall {
			keys = "q:quit | ent:det | /:filt | s:sort | r:scan | f:fetch | x:doctor | p:prof | i:ign"
		} else {
			keys = "q:quit | j/k:move | enter:detail | /:filter | s:sort | r:rescan | f:fetch | x:doctor | p:profile | i:ignore | e:errors"
		}
	}

	message := m.message
	if len(m.scanErrors) > 0 {
		message = fmt.Sprintf("%s (%d errs)", message, len(m.scanErrors))
	}

	// Calculate available space for content
	labelWidth := 8 // "  KEYS │ "
	contentWidth := maxInt(0, width-labelWidth)

	keysLine := fmt.Sprintf("  %s %s %s", styleAccent.Render("KEYS"), styleDim.Render("│"), styleDim.Render(shorten(keys, contentWidth)))
	infoLine := fmt.Sprintf("  %s %s %s", styleAccent.Render("INFO"), styleDim.Render("│"), styleDim.Render(shorten(message, contentWidth)))

	return []string{
		styleDim.Render(strings.Repeat("─", width)),
		keysLine,
		infoLine,
	}
}

func (m model) detailModeView() string {
	if len(m.visible) == 0 {
		return m.mainView()
	}

	entry := m.entries[m.visible[m.cursor]]
	repo := entry.Repo

	var s strings.Builder
	s.WriteString(m.headerLine())
	s.WriteString("\n")
	s.WriteString(m.gitConfigLine())
	s.WriteString("\n")
	if m.mode == modeInput {
		s.WriteString(fmt.Sprintf("Enter value for %s: %s\n", styleAccent.Render(m.inputPlaceholder), m.filterInput.View()))
	}
	s.WriteString("\n")

	contentWidth := m.width - 4
	lines := make([]string, 0)
	lines = append(lines, titleStyle.Render(" FULL REPOSITORY DETAILS "))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Path:"), styleDim.Render(entry.Path)))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Workspace:"), styleDim.Render(repo.Workspace)))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Branch:"), styleAccent.Render(branchSummary(repo))))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Upstream:"), styleDim.Render(emptyDash(repo.Upstream))))
	lines = append(lines, styleLabel.Render("Remotes:"))
	for _, r := range repo.Remotes {
		lines = append(lines, fmt.Sprintf("  %s %s %s (%d PRs)", styleAccent.Render(padRight(r.Name+":", 10)), styleDim.Render(r.URL), remoteSummary(r.Status), r.Status.PRCount))
	}
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Ahead/Behind:"), styleDim.Render(aheadBehind(repo))))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Status:"), styleDim.Render(statusSummary(repo.Status))))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Staged:"), styleDim.Render(yesNo(repo.Status.Staged))))
	lines = append(lines, fmt.Sprintf("%s %d", styleLabel.Render("Stash:"), repo.StashCount))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Submodules:"), styleDim.Render(submoduleSummary(repo.Submodules))))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Last Commit:"), styleDim.Render(timeAgo(repo.LastCommitAt))))
	lines = append(lines, fmt.Sprintf("%s %d %s", styleLabel.Render("Health:"), repo.HealthScore, healthBar(repo.HealthScore, 10)))
	lines = append(lines, "")

	impactLines := m.impactView(repo)
	if len(impactLines) > 0 {
		lines = append(lines, impactLines...)
	}

	lines = renderRecommendations(lines, repo.Recommendations, contentWidth, m.recCursor)

	if m.commandResult != "" {
		lines = append(lines, "")
		lines = append(lines, titleStyle.Render(" LAST COMMAND RESULT "))
		lines = append(lines, "")
		resLines := strings.Split(m.commandResult, "\n")
		for _, rl := range resLines {
			lines = append(lines, "  "+styleDim.Render(rl))
		}
	}

	if strings.TrimSpace(repo.Error) != "" {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("%s %s", styleError.Render("Error:"), repo.Error))
	}

	m.detailViewport.Width = contentWidth
	m.detailViewport.Height = m.height - 8 // Adjust for header and footer
	m.detailViewport.SetContent(strings.Join(lines, "\n"))

	s.WriteString(m.detailViewport.View())
	s.WriteString("\n")

	for _, line := range m.footerLines() {
		s.WriteString(line)
		s.WriteString("\n")
	}

	return docStyle.Render(s.String())
}

func (m model) listView(height, width int) []string {
	lines := make([]string, 0, height)
	header := titleStyle.Render(fmt.Sprintf(" REPOSITORIES (%d) ", len(m.visible)))
	if m.filter != "" {
		header += styleDim.Render(" [filtered]")
	}
	lines = append(lines, pad(header, width))
	lines = append(lines, "") // spacer

	if len(lines) < height {
		branchWidth := 12
		healthWidth := 5
		remoteWidth := 10
		pathWidth := maxInt(0, width-(branchWidth+healthWidth+remoteWidth+10))
		pathCol := padRight("PATH", pathWidth)
		branchCol := padRight("BRANCH", branchWidth)
		healthCol := padRight("HEALTH", healthWidth)
		remoteCol := padRight("REMOTE", remoteWidth)
		headerRow := fmt.Sprintf("  %s %s %s %s %s", "S", pathCol, branchCol, healthCol, remoteCol)
		lines = append(lines, styleDim.Render(headerRow))
	}

	itemsHeight := height - len(lines)
	offset := 0
	if m.cursor >= itemsHeight && itemsHeight > 0 {
		offset = m.cursor - itemsHeight + 1
	}

	for i := 0; i < itemsHeight; i++ {
		idx := offset + i
		if idx >= len(m.visible) {
			lines = append(lines, "")
			continue
		}
		entry := m.entries[m.visible[idx]]
		selector := "  "
		style := lipgloss.NewStyle()
		if idx == m.cursor {
			selector = styleSelect.Render("> ")
			style = styleSelect
		}
		status := statusChar(entry)
		branchWidth := 12
		healthWidth := 5
		remoteWidth := 10
		pathWidth := maxInt(0, width-(branchWidth+healthWidth+remoteWidth+10))
		path := padRight(shorten(repoPathSummaryMulti(m.opts.Roots, entry.Path), pathWidth), pathWidth)
		branch := padRight(shorten(branchSummary(entry.Repo), branchWidth), branchWidth)
		health := healthBar(entry.Repo.HealthScore, healthWidth)
		remote := padRight(remoteSummary(entry.Repo.RemoteStatus), remoteWidth)
		row := fmt.Sprintf("%s%s %s %s %s %s", selector, status, style.Render(path), styleDim.Render(branch), health, remote)
		lines = append(lines, row)
	}

	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

func remoteSummary(status gitinfo.RemoteStatus) string {
	var ci string
	switch status.CIStatus {
	case "success":
		ci = styleCIsuc.Render("✓")
	case "failure":
		ci = styleCIfail.Render("✗")
	case "pending":
		ci = styleCIpend.Render("○")
	case "loading":
		ci = styleDim.Render("…")
	default:
		ci = styleDim.Render("-")
	}

	var pr string
	if status.CIStatus == "loading" {
		pr = "…"
	} else if status.PRCount > 0 {
		pr = fmt.Sprintf("%d PR", status.PRCount)
	} else {
		pr = "-"
	}

	return fmt.Sprintf("%s %s", ci, styleDim.Render(pr))
}

func healthBar(score int, width int) string {
	if width <= 0 {
		return ""
	}
	filled := (score * width) / 100
	if filled == 0 && score > 0 {
		filled = 1
	}

	var style lipgloss.Style
	if score < 50 {
		style = styleHealthLow
	} else if score < 80 {
		style = styleHealthMid
	} else {
		style = styleHealthHigh
	}

	bar := strings.Repeat("■", filled) + styleDim.Render(strings.Repeat("□", width-filled))
	return style.Render(bar)
}

func (m model) detailView(height, width int) []string {
	lines := make([]string, 0, height)
	if len(m.visible) == 0 {
		lines = append(lines, styleDim.Render("No repositories found."))
		return fillLines(lines, height, width)
	}

	entry := m.entries[m.visible[m.cursor]]
	repo := entry.Repo
	lines = append(lines, titleStyle.Render(" DETAILS "))
	lines = append(lines, "") // spacer

	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Path:"), styleDim.Render(entry.Path)))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Workspace:"), styleDim.Render(repo.Workspace)))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Branch:"), styleAccent.Render(branchSummary(repo))))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Upstream:"), styleDim.Render(emptyDash(repo.Upstream))))
	lines = append(lines, styleLabel.Render("Remotes:"))
	for _, r := range repo.Remotes {
		lines = append(lines, fmt.Sprintf("  %s %s (%d PRs)", styleAccent.Render(r.Name+":"), remoteSummary(r.Status), r.Status.PRCount))
	}
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Ahead/Behind:"), styleDim.Render(aheadBehind(repo))))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Status:"), styleDim.Render(statusSummary(repo.Status))))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Staged:"), styleDim.Render(yesNo(repo.Status.Staged))))
	lines = append(lines, fmt.Sprintf("%s %d", styleLabel.Render("Stash:"), repo.StashCount))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Submodules:"), styleDim.Render(submoduleSummary(repo.Submodules))))
	lines = append(lines, fmt.Sprintf("%s %s", styleLabel.Render("Last Commit:"), styleDim.Render(timeAgo(repo.LastCommitAt))))
	lines = append(lines, fmt.Sprintf("%s %d %s", styleLabel.Render("Health:"), repo.HealthScore, healthBar(repo.HealthScore, 10)))

	impactLines := m.impactView(repo)
	if len(impactLines) > 0 {
		lines = append(lines, "")
		lines = append(lines, impactLines...)
	}

	lines = append(lines, "")
	lines = renderRecommendations(lines, repo.Recommendations, width, m.recCursor)

	if strings.TrimSpace(repo.Error) != "" {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("%s %s", styleError.Render("Error:"), repo.Error))
	}

	for i := range lines {
		lines[i] = shorten(lines[i], width)
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for i := range lines {
		lines[i] = pad(lines[i], width)
	}
	return lines
}

func (m model) impactView(repo gitinfo.RepoInfo) []string {
	var lines []string
	if repo.ModuleName == "" {
		return nil
	}

	var dependents []string
	for _, r := range m.repos {
		if r.Path == repo.Path {
			continue
		}
		for _, d := range r.Dependencies {
			if d == repo.ModuleName {
				dependents = append(dependents, r.Path)
				break
			}
		}
	}

	if len(dependents) > 0 {
		lines = append(lines, titleStyle.Render(" IMPACT ANALYSIS "))
		lines = append(lines, styleImpact.Render(fmt.Sprintf("  ⚠ %d repositories depend on this module:", len(dependents))))
		for _, d := range dependents {
			lines = append(lines, styleDim.Render("  - ")+repoPathSummaryMulti(m.opts.Roots, d))
		}
		lines = append(lines, "")
	}

	return lines
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

func (m model) errorsView() string {
	var s strings.Builder
	header := titleStyle.Render(fmt.Sprintf(" SCAN ERRORS (%d) ", len(m.scanErrors)))
	s.WriteString(header)
	s.WriteString("\n")
	s.WriteString(styleDim.Render("esc:back | j/k:move"))
	s.WriteString("\n\n")

	if len(m.scanErrors) == 0 {
		s.WriteString(styleDim.Render("  No scan errors."))
		return docStyle.Render(s.String())
	}

	bodyHeight := m.height - 6
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	offset := 0
	if m.errorCursor >= bodyHeight {
		offset = m.errorCursor - bodyHeight + 1
	}

	for i := 0; i < bodyHeight; i++ {
		idx := offset + i
		if idx >= len(m.scanErrors) {
			break
		}
		selector := "  "
		style := lipgloss.NewStyle()
		if idx == m.errorCursor {
			selector = styleSelect.Render("> ")
			style = styleSelect
		}
		s.WriteString(fmt.Sprintf("%s%s\n", selector, style.Render(m.scanErrors[idx].Error())))
	}

	return docStyle.Render(s.String())
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

func shorten(value string, max int) string {
	if max <= 0 || lipgloss.Width(value) <= max {
		return value
	}
	// Use a more robust way to truncate with ANSI codes if possible,
	// but for now, let's just ensure we are using lipgloss.Width correctly.
	return truncate.StringWithTail(value, uint(max), "...")
}

func joinColumns(left, right []string, leftWidth, rightWidth int) []string {
	height := maxInt(len(left), len(right))
	lines := make([]string, 0, height)
	for i := 0; i < height; i++ {
		leftLine := ""
		if i < len(left) {
			leftLine = left[i]
		}
		rightLine := ""
		if i < len(right) {
			rightLine = right[i]
		}
		lines = append(lines, fmt.Sprintf("%s | %s", pad(leftLine, leftWidth), pad(rightLine, rightWidth)))
	}
	return lines
}

func fillLines(lines []string, height, width int) []string {
	for i := range lines {
		lines[i] = shorten(lines[i], width)
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for i := range lines {
		lines[i] = pad(lines[i], width)
	}
	return lines
}

func pad(value string, width int) string {
	if width <= 0 {
		return value
	}
	if lipgloss.Width(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-lipgloss.Width(value))
}

func padRight(value string, width int) string {
	if width <= 0 {
		return value
	}
	if lipgloss.Width(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-lipgloss.Width(value))
}

func splitWidths(total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	left := total / 2
	if left < 32 {
		left = minInt(32, total)
	}
	right := total - left - 3
	if right < 20 {
		right = maxInt(0, total-left-3)
	}
	return left, maxInt(right, 0)
}

func sortLabel(mode sortMode) string {
	switch mode {
	case sortByDirty:
		return "dirty"
	case sortByWorkspace:
		return "workspace"
	default:
		return "name"
	}
}

func statusChar(entry entry) string {
	if !entry.Repo.IsRepo {
		return styleNaked.Render("P")
	}
	if strings.TrimSpace(entry.Repo.Error) != "" {
		return styleError.Render("E")
	}
	if entry.Repo.Status.Clean {
		return styleClean.Render("C")
	}
	return styleDirty.Render("D")
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

func aheadBehind(repo gitinfo.RepoInfo) string {
	if repo.Upstream == "" {
		return "-"
	}
	return fmt.Sprintf("%d/%d", repo.Ahead, repo.Behind)
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

func renderRecommendations(lines []string, recs []gitinfo.Recommendation, width int, cursor int) []string {
	lines = append(lines, titleStyle.Render(" RECOMMENDATIONS "))
	lines = append(lines, "")
	if len(recs) == 0 {
		lines = append(lines, styleDim.Render("  - No actions recommended."))
		return lines
	}

	wrapStyle := lipgloss.NewStyle().Width(width - 6)
	for i, rec := range recs {
		prefix := fmt.Sprintf("%d. ", i+1)
		indent := strings.Repeat(" ", len(prefix))

		text := rec.Text
		if rec.Command != "" {
			text = fmt.Sprintf("%s: `%s`", rec.Text, rec.Command)
		}

		style := styleAction
		if i == cursor {
			style = styleSelect
			prefix = "> " + prefix
			indent = "  " + indent
		}

		wrapped := wrapStyle.Render(text)
		wrappedLines := strings.Split(wrapped, "\n")
		for j, wl := range wrappedLines {
			if j == 0 {
				lines = append(lines, "  "+style.Render(prefix+wl))
			} else {
				lines = append(lines, "  "+style.Render(indent+wl))
			}
		}
	}
	return lines
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
