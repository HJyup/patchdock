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
	// gutter is the left margin every line of a run is printed against
	gutter = "  "
	// subIndent nests the lines belonging to a block's own title
	subIndent    = gutter + "  "
	childIndent  = gutter + "    "
	fallbackCols = 80
)

const (
	sucessSign = "✔"
	rejectSign = "✖"
	retrySign  = "↻"
	arrowSign  = "→"
)

// RunInfo names the run before any stage opens. It is re-rendered on resize
// rather than flattened once, so the task line truncates to the real width
type RunInfo struct {
	Repo   string
	RunID  string
	Task   string
	LogDir string
}

// Result is the closing account of a run
type Result struct {
	Accepted  bool
	Duration  time.Duration
	Branch    string
	Files     int
	Additions int
	Deletions int
	LogDir    string
}

// Progress is the whole surface the rest of patchdock sees. Every method is
// safe to call from any goroutine: on the live path they become messages to the
// bubbletea program, which is the only thing that writes to the terminal
type Progress struct {
	out     io.Writer
	styles  styles
	live    bool
	timeout time.Duration
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
// fallback so much as the correct renderer for a non-terminal.
func New(out io.Writer, timeout time.Duration) *Progress {
	styles := newStyles(out)
	live := usable(out)

	p := &Progress{
		out:      out,
		styles:   styles,
		live:     live,
		timeout:  timeout,
		finished: make(chan struct{}),
	}

	if !live {
		close(p.finished)
		return p
	}

	// Input stays closed and signals stay ours: this program renders, it does
	// not interact, and Ctrl-C must reach the process rather than be swallowed
	p.program = tea.NewProgram(
		newModel(styles, timeout),
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
// identifiable while it is still going — which repo it belongs to, and where
// its logs are, up front rather than only once it has finished
func (p *Progress) Header(info RunInfo) {
	p.mu.Lock()
	live := p.running
	p.mu.Unlock()

	if live {
		p.program.Send(headerMsg{info: info})
		return
	}

	for _, line := range headerLines(info, p.styles, fallbackCols) {
		fmt.Fprintf(p.out, "%s\n", line)
	}
}

// Start opens a step, closing the previous one first: a stage ending is not
// reported separately, so the next stage beginning is the signal. Any activity
// the previous step reported is dropped
func (p *Progress) Start(label string) {
	p.commit("", p.styles.green.Render(sucessSign))

	p.mu.Lock()
	p.label, p.started, p.active = label, time.Now(), true
	live := p.running
	p.mu.Unlock()

	if live {
		p.program.Send(startMsg{label: label})
		return
	}
	fmt.Fprintf(p.out, "%s%v %s\n", gutter, arrowSign, strings.TrimRight(label, " "))
}

// Note commits a recessive line under the step that just closed — the planner's
// strategy, a reviewer's verdict — so the reason for what happens next survives
// in scrollback rather than flashing past on the activity line
func (p *Progress) Note(text string) {
	if text == "" {
		return
	}

	p.mu.Lock()
	live := p.running
	p.mu.Unlock()

	if live {
		p.program.Send(noteMsg{text: text})
		return
	}
	fmt.Fprintf(p.out, "%s%s\n", childIndent, p.styles.muted.Render(text))
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

	mark := p.styles.green.Render(sucessSign)
	if err != nil {
		mark = p.styles.red.Render(rejectSign)
	}
	p.commit(note, mark)
}

// FinishRetry closes a step that ran cleanly but whose verdict sends the
// pipeline round again. A tick here would read as "all good"
func (p *Progress) FinishRetry(note string) {
	p.commit(note, p.styles.amber.Render(retrySign))
}

// Summary prints the closing lines of a run. It runs after Close, once the
// program has released the terminal, so it writes straight to the output
func (p *Progress) Summary(res Result) {
	fmt.Fprintf(p.out, "\n")
	for _, line := range summaryLines(res, p.styles) {
		fmt.Fprintf(p.out, "%s\n", line)
	}
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
		p.commit("interrupted", p.styles.red.Render(rejectSign))

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

	fmt.Fprintf(p.out, "%s%s %s %s  %s\n",
		gutter, mark, label, p.styles.noteCell(note, noteWidth),
		p.styles.muted.Render(short(elapsed)))
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
