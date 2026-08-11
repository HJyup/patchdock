package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/HJyup/patchdock/internal/daemon/api"
)

const (
	promptCharLimit = 2000
	minPromptWidth  = 20
	promptSign      = "▸ "
	noteLimit       = 8
)

// StreamFunc feeds the dashboard. It should block, calling onSnapshot for
// every snapshot, until ctx is cancelled or the stream breaks
type StreamFunc func(ctx context.Context, onSnapshot func(api.Snapshot) error) error

// SubmitFunc queues one task and returns its run id
type SubmitFunc func(ctx context.Context, prompt string) (string, error)

type CancelFunc func(ctx context.Context, runID string) error

type AppOptions struct {
	Repo         string
	Submit       SubmitFunc
	Cancel       CancelFunc
	Stream       StreamFunc
	StartOnWatch bool
}

func App(ctx context.Context, in io.Reader, out io.Writer, opts AppOptions) error {
	if !Interactive(in, out) {
		return errors.New(`dock needs a terminal — pass the prompt inline instead: dock "…"`)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	program := tea.NewProgram(
		newAppModel(ctx, newStyles(out), opts),
		tea.WithInput(in),
		tea.WithOutput(out),
		tea.WithAltScreen(),
	)

	streamErr := make(chan error, 1)
	go func() {
		err := opts.Stream(ctx, func(snap api.Snapshot) error {
			program.Send(snapshotMsg{snap: snap})
			return nil
		})

		if ctx.Err() != nil {
			streamErr <- nil
			return
		}
		if err == nil {
			err = errors.New("daemon closed the stream")
		}

		streamErr <- err
		program.Send(streamFailedMsg{})
	}()

	_, runErr := program.Run()
	cancel()

	if err := <-streamErr; err != nil {
		return fmt.Errorf("watch runs: %w", err)
	}
	if runErr != nil && !errors.Is(runErr, tea.ErrInterrupted) && !errors.Is(runErr, tea.ErrProgramKilled) {
		return runErr
	}
	return nil
}

type (
	snapshotMsg     struct{ snap api.Snapshot }
	streamFailedMsg struct{}
	queuedMsg       struct{ id, title string }
	submitFailedMsg struct{ err error }
	cancelledMsg    struct{ id string }
	cancelFailedMsg struct {
		id  string
		err error
	}
)

type appMode int

const (
	modePrompt appMode = iota
	modeWatch
)

type note struct {
	id    string
	title string
	err   error
}

type appModel struct {
	ctx    context.Context
	styles styles
	submit SubmitFunc
	cancel CancelFunc
	repo   string

	mode  appMode
	input textinput.Model
	notes []note

	watch watchModel

	width  int
	height int
}

func newAppModel(ctx context.Context, s styles, opts AppOptions) appModel {
	input := textinput.New()
	input.Prompt = promptSign
	input.Placeholder = "what should the agent do?"
	input.CharLimit = promptCharLimit
	input.PromptStyle = s.accent
	input.PlaceholderStyle = s.muted
	input.Width = fallbackCols
	input.Focus()

	mode := modePrompt
	if opts.StartOnWatch {
		mode = modeWatch
	}

	footer := "tab new task · q quit"
	if opts.Cancel != nil {
		footer = "tab new task · c cancel · q quit"
	}

	return appModel{
		ctx:    ctx,
		styles: s,
		submit: opts.Submit,
		cancel: opts.Cancel,
		repo:   tildePath(opts.Repo),
		mode:   mode,
		input:  input,
		watch:  newWatchModel(s, footer),
		width:  fallbackCols,
	}
}

func (m appModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.watch.spin.Tick)
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
			m.input.Width = max(m.width-len(gutter)-ansi.StringWidth(promptSign)-1, minPromptWidth)
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		m.watch, _ = m.watch.update(msg)
		return m, nil

	case tea.KeyMsg:
		if m.mode == modeWatch {
			return m.watchKeys(msg)
		}
		return m.promptKeys(msg)

	case queuedMsg:
		return m.noted(note{id: msg.id, title: msg.title}), nil

	case submitFailedMsg:
		return m.noted(note{err: msg.err}), nil

	case cancelledMsg:
		m.watch = m.watch.withNotice("")
		return m, nil

	case cancelFailedMsg:
		m.watch = m.watch.withNotice(fmt.Sprintf("cancel %s: %v", msg.id, msg.err))
		return m, nil

	case streamFailedMsg:
		return m, tea.Quit

	case snapshotMsg, spinner.TickMsg:
		var cmd tea.Cmd
		m.watch, cmd = m.watch.update(msg)
		return m, cmd
	}

	return m, nil
}

func (m appModel) promptKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		prompt := strings.TrimSpace(m.input.Value())
		if prompt == "" {
			return m, nil
		}

		m.input.Reset()
		return m, m.submitCmd(prompt)

	case "tab":
		m.mode = modeWatch
		return m, nil

	case "esc", "ctrl+c":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m appModel) watchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.watch = m.watch.withNotice("") // any key clears a stale failure

	if m.watch.selecting {
		return m.cancelKeys(msg)
	}

	switch msg.String() {
	case "tab", "n":
		m.mode = modePrompt
		return m, textinput.Blink

	case "c":
		if m.cancel != nil {
			m.watch = m.watch.startSelecting()
		}
		return m, nil

	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	}

	return m, nil
}

func (m appModel) cancelKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.watch = m.watch.move(-1)

	case "down", "j":
		m.watch = m.watch.move(1)

	case "enter":
		run, ok := m.watch.selected()
		m.watch = m.watch.stopSelecting()
		if !ok {
			return m, nil
		}
		return m, m.cancelCmd(run.ID)

	case "c", "esc", "q":
		m.watch = m.watch.stopSelecting()

	case "ctrl+c":
		return m, tea.Quit
	}

	return m, nil
}

func (m appModel) submitCmd(prompt string) tea.Cmd {
	return func() tea.Msg {
		id, err := m.submit(m.ctx, prompt)
		if err != nil {
			return submitFailedMsg{err: err}
		}
		return queuedMsg{id: id, title: prompt}
	}
}

func (m appModel) cancelCmd(runID string) tea.Cmd {
	return func() tea.Msg {
		if err := m.cancel(m.ctx, runID); err != nil {
			return cancelFailedMsg{id: runID, err: err}
		}
		return cancelledMsg{id: runID}
	}
}

func (m appModel) noted(n note) appModel {
	m.notes = append(m.notes, n)
	if len(m.notes) > noteLimit {
		m.notes = m.notes[len(m.notes)-noteLimit:]
	}
	return m
}

func (m appModel) View() string {
	if m.mode == modeWatch {
		return m.watch.View()
	}
	return m.promptView()
}

func (m appModel) promptView() string {
	var b strings.Builder

	left := gutter + m.styles.title.Render("patchdock")
	right := m.styles.muted.Render(m.repo)
	gap := max(m.width-ansi.StringWidth(left)-ansi.StringWidth(right)-len(gutter), 2)
	fmt.Fprintf(&b, "\n%s%s%s\n", left, strings.Repeat(" ", gap), right)

	fmt.Fprintf(&b, "\n%s%s\n", gutter, m.input.View())

	if len(m.notes) > 0 {
		b.WriteString("\n")
		for _, n := range m.notes {
			fmt.Fprintf(&b, "%s%s\n", subIndent, m.noteLine(n))
		}
	}

	footer := gutter + m.styles.muted.Render("enter queue · tab watch · esc quit")
	return pinFooter(b.String(), footer, m.height)
}

func (m appModel) noteLine(n note) string {
	room := max(m.width-len(subIndent)-2, minDetail)

	if n.err != nil {
		text := ansi.Truncate(oneLine(n.err.Error()), room-2, "…")
		return m.styles.red.Render(rejectSign) + " " + m.styles.muted.Render(text)
	}

	line := m.styles.green.Render(sucessSign) + " " +
		m.styles.muted.Render("queued") + " " + n.id
	if title := oneLine(n.title); title != "" {
		width := max(room-ansi.StringWidth("  queued "+n.id), minDetail)
		line += "  " + m.styles.muted.Render(ansi.Truncate(title, width, "…"))
	}

	return line
}
