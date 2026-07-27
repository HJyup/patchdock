// Package tui renders a run's progress to the terminal. On a terminal a
// bubbletea program owns a small live region below the cursor and commits each
// finished step to scrollback; anywhere else the same events become plain
// lines, so redirected output stays readable and free of escape sequences
package tui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	// detailIndent lines the activity line up under the step's label
	detailIndent = "  "
	fallbackCols = 80
)

// Progress is the whole surface the rest of patchdock sees. Every method is
// safe to call from any goroutine: on the live path they become messages to the
// bubbletea program, which is the only thing that writes to the terminal
type Progress struct {
	out     io.Writer
	styles  styles
	live    bool
	program *tea.Program

	mu      sync.Mutex
	running bool
	label   string // plain path only; the model tracks its own
	started time.Time
	active  bool

	finished  chan struct{}
	closeOnce sync.Once
}

// New picks the renderer that suits out. A tea.Program writing to a pipe
// produces repaint sequences rather than text, so the plain path is not a
// fallback so much as the correct renderer for a non-terminal
func New(out io.Writer) *Progress {
	styles := newStyles(out)
	live := usable(out)

	p := &Progress{
		out:      out,
		styles:   styles,
		live:     live,
		finished: make(chan struct{}),
	}

	if !live {
		close(p.finished)
		return p
	}

	// Input stays closed and signals stay ours: this program renders, it does
	// not interact, and Ctrl-C must reach the process rather than be swallowed
	p.program = tea.NewProgram(
		newModel(styles),
		tea.WithOutput(out),
		tea.WithInput(nil),
		tea.WithoutSignalHandler(),
	)
	p.running = true

	go func() {
		defer close(p.finished)
		_, _ = p.program.Run()
	}()

	return p
}

// Header names the run before any step opens, so a long pipeline is
// identifiable while it is still going
func (p *Progress) Header(title, detail string) {
	line := "▸ " + p.styles.title.Render(title)
	if detail != "" {
		line += "  " + detail
	}

	p.mu.Lock()
	live := p.running
	p.mu.Unlock()

	if live {
		p.program.Send(headerMsg{text: line})
		return
	}
	fmt.Fprintf(p.out, "%s\n", line)
}

// Start opens a step. Any activity the previous step reported is dropped
func (p *Progress) Start(label string) {
	p.mu.Lock()
	p.label, p.started, p.active = label, time.Now(), true
	live := p.running
	p.mu.Unlock()

	if live {
		p.program.Send(startMsg{label: label})
		return
	}
	fmt.Fprintf(p.out, "→ %s\n", strings.TrimRight(label, " "))
}

// Detail replaces the activity line under the active step. It is dropped off a
// terminal: activity belongs to the moment, and stdout.log holds the durable
// version
func (p *Progress) Detail(activity string) {
	p.mu.Lock()
	live := p.running && p.active
	p.mu.Unlock()

	if live {
		p.program.Send(detailMsg{activity: activity})
	}
}

// Finish closes the active step: a tick, or a cross when err is set
func (p *Progress) Finish(note string, err error) {
	// A failed stage has no meaningful result to report, and whatever the caller
	// read off a zero value would be misleading next to a cross
	if err != nil {
		note = ""
	}

	mark := p.styles.green.Render("✔")
	if err != nil {
		mark = p.styles.red.Render("✖")
	}
	p.commit(note, mark)
}

// FinishRetry closes a step that ran cleanly but whose verdict sends the
// pipeline round again. A tick here would read as "all good"
func (p *Progress) FinishRetry(note string) {
	p.commit(note, p.styles.amber.Render("↻"))
}

// Summary prints the closing lines of a run. It runs after Close, once the
// program has released the terminal, so it writes straight to the output
func (p *Progress) Summary(headline, detail string) {
	fmt.Fprintf(p.out, "\n%s\n%s\n",
		p.styles.title.Render(headline), p.styles.muted.Render(detail))
}

// Muted renders text in the same recessive grey as elapsed times, so a caller
// can fold an aside into a label without it competing for attention
func (p *Progress) Muted(text string) string {
	return p.styles.muted.Render(text)
}

// Close commits an unfinished step and shuts the program down, restoring the
// terminal. Safe to call more than once
func (p *Progress) Close() {
	p.closeOnce.Do(func() {
		p.commit("interrupted", p.styles.red.Render("✖"))

		p.mu.Lock()
		program, running := p.program, p.running
		p.running = false
		p.mu.Unlock()

		if running {
			program.Quit()
			<-p.finished
		}
	})
}

func (p *Progress) commit(note, mark string) {
	p.mu.Lock()
	if !p.active {
		p.mu.Unlock()
		return
	}
	p.active = false
	live, label, elapsed := p.running, p.label, time.Since(p.started)
	p.mu.Unlock()

	if live {
		p.program.Send(finishMsg{note: note, mark: mark})
		return
	}

	fmt.Fprintf(p.out, "%s %s %s  %s\n",
		mark, label, p.styles.noteCell(note, noteWidth), p.styles.muted.Render(short(elapsed)))
}

func short(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
}

func Interactive(in io.Reader, out io.Writer) bool {
	return usable(in) && usable(out)
}

func usable(stream any) bool {
	return isTerminal(stream) && os.Getenv("TERM") != "dumb"
}

func isTerminal(stream any) bool {
	file, ok := stream.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
