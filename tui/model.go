package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gi4nks/tecla/gitinfo"
	"github.com/gi4nks/tecla/internal/config"
	"github.com/gi4nks/tecla/internal/runner"
	"github.com/gi4nks/tecla/scanner"
	"github.com/atotto/clipboard"
)

type mode int

type sortMode int

type entry struct {
	Repo  gitinfo.RepoInfo
	Path  string
}

type scanResultMsg struct {
	Repos      []gitinfo.RepoInfo
	Dirs       []string
	ScanErrors []error
	Err        error
}

type globalConfigMsg struct {
	Config gitinfo.GlobalConfig
}

const (

	modeMain mode = iota

	modeFilter

	modeErrors

	modeDetail

	modeInput

)



const (

	sortByName sortMode = iota

	sortByDirty

	sortByWorkspace

)



type model struct {
	opts           Options
	repos          []gitinfo.RepoInfo
	globalConfig   gitinfo.GlobalConfig
	dirs           []string
	entries        []entry
	scanErrors     []error
	visible        []int
	selected       map[string]bool
	cursor         int
	sortMode       sortMode
	errorCursor    int
	recCursor      int
	commandResult  string
	pendingCommand string
	inputPlaceholder string
	filterInput    textinput.Model
	detailViewport viewport.Model
	filter         string
	mode           mode
	parentMode     mode
	spinner        spinner.Model
	loading        bool
	message        string
	width          int
	height         int
}

func newModel(opts Options) model {
	spin := spinner.New()
	spin.Spinner = spinner.Spinner{Frames: []string{"-", "\\", "|", "/"}, FPS: 120 * time.Millisecond}
	input := textinput.New()
	input.Placeholder = "filter..."
	input.CharLimit = 64
	input.Prompt = "/ "

	m := model{
		opts:           opts,
		sortMode:       sortByName,
		filterInput:    input,
		detailViewport: viewport.New(0, 0),
		mode:           modeMain,
		spinner:        spin,
		loading:        true,
		selected:       make(map[string]bool),
	}
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, scanCmd(m.opts), globalConfigCmd(m.opts))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	case globalConfigMsg:
		m.globalConfig = msg.Config
		return m, nil
	case scanResultMsg:
		m.loading = false
		if msg.Err != nil {
			m.message = fmt.Sprintf("Scan failed: %v", msg.Err)
		} else {
			m.message = fmt.Sprintf("Scan complete (%d repos)", len(msg.Repos))
		}
		m.repos = msg.Repos
		m.dirs = msg.Dirs
		m.scanErrors = msg.ScanErrors
		m.applySortFilter()
		return m, nil
	case clipboardMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Clipboard error: %v", msg.err)
		} else {
			m.message = "Copied to clipboard"
		}
		return m, nil
	case ignoreMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Ignore error: %v", msg.err)
		} else {
			m.message = fmt.Sprintf("Ignored: %s", msg.path)
			// Trigger rescan to hide the ignored repo
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, scanCmd(m.opts))
		}
		return m, nil
	case commandFinishedMsg:
		m.loading = false
		if msg.err != nil {
			m.message = fmt.Sprintf("Error: %v", msg.err)
			m.commandResult = fmt.Sprintf("Error: %v\n%s", msg.err, msg.output)
		} else {
			m.message = "Command executed successfully"
			m.commandResult = msg.output
			// Refresh
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, scanCmd(m.opts))
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.mode {
	case modeFilter:
		if key == "enter" {
			m.filter = strings.TrimSpace(m.filterInput.Value())
			m.filterInput.Blur()
			m.mode = modeMain
			m.applySortFilter()
			return m, nil
		}
		if key == "esc" {
			m.filterInput.Blur()
			m.mode = modeMain
			return m, nil
		}
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return m, cmd
	case modeInput:
		return m.handleInputKey(msg)
	case modeErrors:
		return m.handleErrorsKey(key)
	case modeDetail:
		return m.handleDetailKey(msg)
	default:
		return m.handleMainKey(key)
	}
}

func (m model) handleMainKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "enter":
		if len(m.visible) > 0 {
			m.mode = modeDetail
			m.detailViewport.YOffset = 0
			m.recCursor = 0
			m.commandResult = ""
		}
		return m, nil
	case "j", "down":
		m.cursor = minInt(m.cursor+1, maxInt(len(m.visible)-1, 0))
		return m, nil
	case "k", "up":
		m.cursor = maxInt(m.cursor-1, 0)
		return m, nil
	case " ", "m":
		if len(m.visible) > 0 {
			entry := m.entries[m.visible[m.cursor]]
			if m.selected[entry.Path] {
				delete(m.selected, entry.Path)
			} else {
				m.selected[entry.Path] = true
			}
		}
		return m, nil
	case "b":
		if len(m.selected) > 0 {
			m.mode = modeInput
			m.parentMode = modeMain
			m.pendingCommand = ""
			m.inputPlaceholder = "batch command"
			m.filterInput.SetValue("")
			m.filterInput.Placeholder = "git pull --rebase"
			m.filterInput.Focus()
		} else {
			m.message = "No repositories selected for batch operation"
		}
		return m, nil
	case "/":
		m.mode = modeFilter
		m.filterInput.SetValue(m.filter)
		m.filterInput.CursorEnd()
		m.filterInput.Focus()
		return m, nil
	case "s":
		m.sortMode = (m.sortMode + 1) % 3
		m.applySortFilter()
		return m, nil
	case "r":
		m.loading = true
		m.message = "Rescanning..."
		return m, tea.Batch(m.spinner.Tick, scanCmd(m.opts))
	case "i":
		if len(m.visible) > 0 {
			entry := m.entries[m.visible[m.cursor]]
			return m, ignoreRepoCmd(entry.Path)
		}
		return m, nil
	case "f":
		m.loading = true
		m.message = "Fetching all repositories..."
		var paths []string
		for _, repo := range m.repos {
			if repo.IsRepo && repo.Remote != "" {
				paths = append(paths, repo.Path)
			}
		}
		return m, tea.Batch(m.spinner.Tick, runBatchCommandCmd(paths, "git fetch --all"))
	case "e":
		m.mode = modeErrors
		m.errorCursor = 0
		return m, nil
	}
	return m, nil
}

func (m model) handleInputKey(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		key := keyMsg.String()
		if key == "enter" {
			val := strings.TrimSpace(m.filterInput.Value())
			if val != "" {
				if m.parentMode == modeMain {
					m.filterInput.Blur()
					m.mode = modeMain
					m.loading = true
					m.message = fmt.Sprintf("Executing batch: %s", val)
					var paths []string
					for p := range m.selected {
						paths = append(paths, p)
					}
					return m, tea.Batch(m.spinner.Tick, runBatchCommandCmd(paths, val))
				}

				fullCmd := strings.Replace(m.pendingCommand, m.inputPlaceholder, val, 1)
				m.filterInput.Blur()
				
				// Check if there are more placeholders
				if nextPlaceholder := findPlaceholder(fullCmd); nextPlaceholder != "" {
					m.pendingCommand = fullCmd
					m.inputPlaceholder = nextPlaceholder
					m.filterInput.SetValue("")
					m.filterInput.Placeholder = nextPlaceholder
					m.filterInput.Focus()
					return m, nil
				}

				m.mode = m.parentMode
				m.loading = true
				m.message = fmt.Sprintf("Executing: %s", fullCmd)
				entry := m.entries[m.visible[m.cursor]]
				return m, tea.Batch(m.spinner.Tick, runCommandCmd(entry.Path, fullCmd))
			}
			return m, nil
		}
		if key == "esc" {
			m.filterInput.Blur()
			m.mode = m.parentMode
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	return m, cmd
}

func findPlaceholder(cmd string) string {
	start := strings.Index(cmd, "<")
	if start == -1 {
		return ""
	}
	end := strings.Index(cmd[start:], ">")
	if end == -1 {
		return ""
	}
	return cmd[start : start+end+1]
}

func (m model) handleDetailKey(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		key := keyMsg.String()
		entry := m.entries[m.visible[m.cursor]]
		recs := entry.Repo.Recommendations

		switch key {
		case "esc", "q":
			m.mode = modeMain
			return m, nil
		case "tab":
			if len(recs) > 0 {
				m.recCursor = (m.recCursor + 1) % len(recs)
			}
			return m, nil
		case "a":
			if len(recs) > 0 && m.recCursor < len(recs) {
				rec := recs[m.recCursor]
				if rec.Command != "" {
					if placeholder := findPlaceholder(rec.Command); placeholder != "" {
						m.mode = modeInput
						m.parentMode = modeDetail
						m.pendingCommand = rec.Command
						m.inputPlaceholder = placeholder
						m.filterInput.SetValue("")
						m.filterInput.Placeholder = placeholder
						m.filterInput.Focus()
						return m, nil
					}

					m.loading = true
					m.message = fmt.Sprintf("Executing: %s", rec.Command)
					return m, tea.Batch(m.spinner.Tick, runCommandCmd(entry.Path, rec.Command))
				}
			}
			return m, nil
		case "c":
			if len(recs) > 0 && m.recCursor < len(recs) {
				rec := recs[m.recCursor]
				if rec.Command != "" {
					return m, copyToClipboardCmd(rec.Command)
				}
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.detailViewport, cmd = m.detailViewport.Update(msg)
	return m, cmd
}

type clipboardMsg struct {
	err error
}

func copyToClipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		err := clipboard.WriteAll(text)
		return clipboardMsg{err: err}
	}
}

type ignoreMsg struct {
	path string
	err  error
}

func ignoreRepoCmd(path string) tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return ignoreMsg{err: err}
		}
		if cfg.AddIgnore(path) {
			if err := config.Save(cfg); err != nil {
				return ignoreMsg{err: err}
			}
			return ignoreMsg{path: path}
		}
		return ignoreMsg{path: path} // already ignored
	}
}

type commandFinishedMsg struct {
	output string
	err    error
}

func runCommandCmd(path, command string) tea.Cmd {
	return func() tea.Msg {
		output, err := runner.GlobalRunner.RunShell(context.Background(), path, 30*time.Second, command)
		return commandFinishedMsg{
			output: output,
			err:    err,
		}
	}
}

func runBatchCommandCmd(paths []string, command string) tea.Cmd {
	return func() tea.Msg {
		var allOutput strings.Builder
		var lastErr error
		for _, path := range paths {
			out, err := runner.GlobalRunner.RunShell(context.Background(), path, 30*time.Second, command)
			
			allOutput.WriteString(fmt.Sprintf("[%s]\n%s\n", path, out))
			if err != nil {
				allOutput.WriteString(fmt.Sprintf("Error: %v\n", err))
				lastErr = err
			}
		}
		return commandFinishedMsg{
			output: allOutput.String(),
			err:    lastErr,
		}
	}
}

func (m model) handleErrorsKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.mode = modeMain
		return m, nil
	case "j", "down":
		m.errorCursor = minInt(m.errorCursor+1, maxInt(len(m.scanErrors)-1, 0))
		return m, nil
	case "k", "up":
		m.errorCursor = maxInt(m.errorCursor-1, 0)
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	switch m.mode {
	case modeErrors:
		return m.errorsView()
	case modeDetail:
		return m.detailModeView()
	case modeInput:
		if m.parentMode == modeDetail {
			return m.detailModeView()
		}
		return m.mainView()
	default:
		return m.mainView()
	}
}

func globalConfigCmd(opts Options) tea.Cmd {
	return func() tea.Msg {
		cfg := gitinfo.GetGlobalConfig(context.Background(), opts.Timeout)
		return globalConfigMsg{Config: cfg}
	}
}

func scanCmd(opts Options) tea.Cmd {
	return func() tea.Msg {
		repos, dirs, scanErrs := scanner.ScanAll(scanner.Options{
			Roots:              opts.Roots,
			IncludeHidden:      opts.IncludeHidden,
			ExcludePatterns:    opts.Exclude,
			DefaultIgnoredDirs: opts.DefaultIgnoredDirs,
			MaxDepth:           opts.MaxDepth,
		})
		sort.Strings(repos)
		sort.Strings(dirs)
		infos := gitinfo.Collect(context.Background(), repos, gitinfo.Options{
			Timeout:               opts.Timeout,
			Workers:               opts.Workers,
			StaleThresholdDays:    opts.StaleThresholdDays,
			CustomRecommendations: opts.CustomRecommendations,
		}, nil)
		sort.Slice(infos, func(i, j int) bool {
			return infos[i].Path < infos[j].Path
		})
		return scanResultMsg{Repos: infos, Dirs: dirs, ScanErrors: scanErrs}
	}
}

func (m *model) applySortFilter() {
	entries := buildEntries(m.repos)
	order := sortedEntries(entries, m.sortMode)
	query := strings.ToLower(strings.TrimSpace(m.filter))
	var visible []int
	for _, idx := range order {
		entry := entries[idx]
		if query == "" || entryMatches(entry, query) {
			visible = append(visible, idx)
		}
	}
	m.entries = entries
	m.visible = visible
	if len(m.visible) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
}

func buildEntries(repos []gitinfo.RepoInfo) []entry {
	entries := make([]entry, 0, len(repos))
	for _, repo := range repos {
		entries = append(entries, entry{Repo: repo, Path: repo.Path})
	}
	return entries
}

func entryMatches(entry entry, query string) bool {
	if strings.Contains(strings.ToLower(entry.Path), query) {
		return true
	}
	if strings.Contains(strings.ToLower(entry.Repo.Branch), query) {
		return true
	}
	if strings.Contains(strings.ToLower(entry.Repo.Remote), query) {
		return true
	}
	if strings.Contains(strings.ToLower(entry.Repo.Upstream), query) {
		return true
	}
	return false
}

func sortedEntries(entries []entry, mode sortMode) []int {
	indices := make([]int, len(entries))
	for i := range entries {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		a := entries[indices[i]]
		b := entries[indices[j]]
		switch mode {
		case sortByDirty:
			dirtyA := entryDirtyScore(a)
			dirtyB := entryDirtyScore(b)
			if dirtyA != dirtyB {
				return dirtyA > dirtyB
			}
		case sortByWorkspace:
			if a.Repo.Workspace != b.Repo.Workspace {
				return a.Repo.Workspace < b.Repo.Workspace
			}
		}
		return a.Path < b.Path
	})
	return indices
}

func entryDirtyScore(entry entry) int {
	if entry.Repo.Error != "" {
		return 3
	}
	if entry.Repo.Status.Clean {
		return 0
	}
	return 2
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}