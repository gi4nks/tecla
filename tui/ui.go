package tui

import (
	"path/filepath"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gi4nks/tecla/gitinfo"
)

type Options struct {
	Roots                 []string
	IncludeHidden         bool
	Exclude               []string
	DefaultIgnoredDirs    []string
	MaxDepth              int
	Workers               int
	Timeout               time.Duration
	StaleThresholdDays    int
	CustomRecommendations []gitinfo.CustomRecommendation
}

func Run(opts Options) error {
	if len(opts.Roots) == 0 {
		opts.Roots = []string{"."}
	}
	for i, root := range opts.Roots {
		absRoot, err := filepath.Abs(root)
		if err == nil {
			opts.Roots[i] = absRoot
		}
	}
	if opts.Timeout == 0 {
		opts.Timeout = 3 * time.Second
	}
	if opts.Workers <= 0 {
		opts.Workers = runtime.NumCPU()
	}

	program := tea.NewProgram(newModel(opts), tea.WithAltScreen())
	_, err := program.Run()
	return err
}
