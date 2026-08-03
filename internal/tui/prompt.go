package tui

import (
	"errors"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

var ErrPromptCancelled = errors.New("prompt cancelled")

const (
	promptCharLimit = 2000
	minPromptWidth  = 20
	promptSign      = "▸ "
)

func Prompt(in io.Reader, out io.Writer) (string, error) {
	program := tea.NewProgram(
		newPromptModel(newStyles(out)),
		tea.WithOutput(out),
		tea.WithInput(in),
	)

	final, err := program.Run()
	if errors.Is(err, tea.ErrInterrupted) || errors.Is(err, tea.ErrProgramKilled) {
		return "", ErrPromptCancelled
	}
	if err != nil {
		return "", err
	}

	m, ok := final.(promptModel)
	if !ok || m.cancelled {
		return "", ErrPromptCancelled
	}

	return strings.TrimSpace(m.input.Value()), nil
}

type promptModel struct {
	styles    styles
	input     textinput.Model
	cancelled bool
	submitted bool
}

func newPromptModel(s styles) promptModel {
	input := textinput.New()
	input.Prompt = promptSign
	input.Placeholder = "what should the agent do?"
	input.CharLimit = promptCharLimit
	input.PromptStyle = s.accent
	input.PlaceholderStyle = s.muted
	input.Width = fallbackCols - ansi.StringWidth(promptSign)
	input.Focus()

	return promptModel{styles: s, input: input}
}

func (m promptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m promptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.input.Width = max(msg.Width-ansi.StringWidth(promptSign)-1, minPromptWidth)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if strings.TrimSpace(m.input.Value()) == "" {
				return m, nil
			}
			m.submitted = true
			return m, tea.Quit

		case tea.KeyEsc, tea.KeyCtrlC:
			m.cancelled = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m promptModel) View() string {
	if m.submitted || m.cancelled {
		return ""
	}

	return m.input.View() + "\n" +
		m.styles.muted.Render("enter to run · esc to cancel") + "\n"
}
