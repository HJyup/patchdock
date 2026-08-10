package tui

import (
	"errors"
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

func ConfirmWatch(in io.Reader, out io.Writer, runID string) (bool, error) {
	program := tea.NewProgram(
		newConfirmModel(newStyles(out), runID),
		tea.WithInput(in),
		tea.WithOutput(out),
	)

	final, err := program.Run()
	if errors.Is(err, tea.ErrInterrupted) || errors.Is(err, tea.ErrProgramKilled) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	m, ok := final.(confirmModel)
	return ok && m.watch, nil
}

type confirmModel struct {
	styles  styles
	runID   string
	watch   bool
	decided bool
}

func newConfirmModel(s styles, runID string) confirmModel {
	return confirmModel{styles: s, runID: runID}
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "enter", "w":
		m.watch, m.decided = true, true
		return m, tea.Quit

	case "esc", "q", "ctrl+c":
		m.decided = true
		return m, tea.Quit
	}

	return m, nil
}

func (m confirmModel) View() string {
	if m.decided {
		return ""
	}

	return gutter + m.styles.green.Render(sucessSign) + " queued " + m.styles.title.Render(m.runID) + "\n" +
		subIndent + m.styles.muted.Render("enter to watch · esc to exit") + "\n"
}
