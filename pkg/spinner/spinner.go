package spinner

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
)

const (
	stateSuccess = "success"
	stateFail    = "fail"
	stateOK      = "ok"
	stateDone    = "done"
)

var (
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

var appColors = []lipgloss.Color{
	lipgloss.Color("6"),  // cyan
	lipgloss.Color("5"),  // magenta
	lipgloss.Color("4"),  // blue
	lipgloss.Color("3"),  // yellow
	lipgloss.Color("2"),  // green
	lipgloss.Color("13"), // bright magenta
	lipgloss.Color("14"), // bright cyan
	lipgloss.Color("12"), // bright blue
}

// logMsg is sent to the bubbletea program from WriteLevel.
type logMsg struct {
	app   string
	msg   string
	state string // "", "success", "fail", "warn", "ok", "done"
}

// deferredQuit is sent after a short delay to allow queued messages to drain.
type deferredQuit struct{}

// completedLine is a finalized line of output (terminal state reached).
type completedLine struct {
	app   string
	msg   string
	state string
}

type appState struct {
	spinner spinner.Model
	msg     string
	done    bool
}

type model struct {
	apps      map[string]*appState
	order     []string
	appColors map[string]lipgloss.Style
	completed []completedLine
	quitting  bool
}

func newModel() model {
	return model{
		apps:      make(map[string]*appState),
		appColors: make(map[string]lipgloss.Style),
	}
}

func (m model) assignColor(app string) {
	if _, exists := m.appColors[app]; !exists {
		idx := len(m.appColors) % len(appColors)
		m.appColors[app] = lipgloss.NewStyle().Bold(true).Foreground(appColors[idx])
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func deferQuit() tea.Msg {
	time.Sleep(50 * time.Millisecond)
	return deferredQuit{}
}

func isTerminalState(state string) bool {
	return state == stateSuccess || state == stateFail || state == stateOK || state == stateDone
}

func (m *model) applyLogMsg(as *appState, msg logMsg) {
	if msg.state != "" {
		m.completed = append(m.completed, completedLine(msg))
		if isTerminalState(msg.state) {
			as.done = true
		}
	} else {
		as.msg = msg.msg
	}
}

func (m *model) handleNewApp(msg logMsg) (tea.Model, tea.Cmd) {
	s := spinner.New(spinner.WithSpinner(spinner.Points))
	as := &appState{spinner: s}
	m.apps[msg.app] = as
	m.order = append(m.order, msg.app)
	m.assignColor(msg.app)

	as.msg = msg.msg
	if msg.state != "" {
		m.completed = append(m.completed, completedLine(msg))
		if isTerminalState(msg.state) {
			as.done = true
		}
	}

	cmds := []tea.Cmd{as.spinner.Tick}
	if m.allDone() && !m.quitting {
		m.quitting = true
		cmds = append(cmds, deferQuit)
	}
	return m, tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case logMsg:
		as, exists := m.apps[msg.app]
		if !exists {
			return m.handleNewApp(msg)
		}

		m.applyLogMsg(as, msg)

		if m.allDone() && !m.quitting {
			m.quitting = true
			return m, deferQuit
		}

		return m, nil

	case deferredQuit:
		return m, tea.Quit

	case spinner.TickMsg:
		var cmds []tea.Cmd
		for _, key := range m.order {
			as := m.apps[key]
			if as.done {
				continue
			}
			var cmd tea.Cmd
			as.spinner, cmd = as.spinner.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	// Render completed lines first (frozen, no longer updated)
	for _, cl := range m.completed {
		prefix := m.appColors[cl.app].Render(cl.app) + ": "
		switch cl.state {
		case stateSuccess:
			fmt.Fprintf(&b, "%s%s%s\n", greenStyle.Render("✓ "), prefix, greenStyle.Render(cl.msg))
		case stateFail:
			fmt.Fprintf(&b, "%s%s%s\n", redStyle.Render("✗ "), prefix, cl.msg)
		case "warn":
			fmt.Fprintf(&b, "%s%s%s\n", yellowStyle.Render("! "), prefix, cl.msg)
		case stateOK:
			fmt.Fprintf(&b, "%s%s%s\n", greenStyle.Render("✓ "), prefix, cl.msg)
		case "hint":
			fmt.Fprintf(&b, "%s%s%s\n", cyanStyle.Render("? "), prefix, cl.msg)
		case stateDone:
			fmt.Fprintf(&b, "  %s%s\n", prefix, cl.msg)
		}
	}

	// Render active spinners
	for _, key := range m.order {
		as := m.apps[key]
		if as.done {
			continue
		}
		prefix := m.appColors[key].Render(key) + ": "
		fmt.Fprintf(&b, "%s %s%s\n", as.spinner.View(), prefix, as.msg)
	}

	return b.String()
}

func (m model) allDone() bool {
	if len(m.apps) == 0 {
		return false
	}
	for _, as := range m.apps {
		if !as.done {
			return false
		}
	}
	return true
}

// SpinnerWriter implements zerolog.LevelWriter and io.Writer for spinner output.
type SpinnerWriter struct {
	mu      sync.Mutex
	program *tea.Program
	started bool
	hasApps bool
	done    chan struct{}
}

// NewWriter creates a new SpinnerWriter.
func NewWriter() *SpinnerWriter {
	sw := &SpinnerWriter{
		done: make(chan struct{}),
	}
	m := newModel()
	sw.program = tea.NewProgram(m,
		tea.WithInput(nil),
		tea.WithOutput(os.Stderr),
	)
	go func() {
		defer close(sw.done)
		_, _ = sw.program.Run()
	}()
	sw.started = true
	return sw
}

// Write implements io.Writer.
func (sw *SpinnerWriter) Write(p []byte) (n int, err error) {
	return sw.WriteLevel(zerolog.InfoLevel, p)
}

// WriteLevel implements zerolog.LevelWriter.
func (sw *SpinnerWriter) WriteLevel(_ zerolog.Level, p []byte) (n int, err error) {
	var event map[string]interface{}
	if err := json.Unmarshal(p, &event); err != nil {
		return len(p), nil
	}

	msg, _ := event["message"].(string)
	app, _ := event["app"].(string)

	if app == "" {
		return len(p), nil
	}

	var state string
	switch {
	case event["success"] == true:
		state = stateSuccess
	case event["fail"] == true:
		state = stateFail
	case event["warn"] == true:
		state = "warn"
	case event["ok"] == true:
		state = stateOK
	case event["hint"] == true:
		state = "hint"
	case event["done"] == true:
		state = stateDone
	}

	sw.mu.Lock()
	sw.hasApps = true
	sw.mu.Unlock()

	sw.program.Send(logMsg{
		app:   app,
		msg:   msg,
		state: state,
	})

	return len(p), nil
}

// HasApps returns true if any log messages with an app field were sent.
func (sw *SpinnerWriter) HasApps() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.hasApps
}

// Stop signals the spinner program to quit and waits for it to exit.
func (sw *SpinnerWriter) Stop() {
	if sw.started {
		sw.program.Quit()
		<-sw.done
	}
}

// Wait blocks until the spinner program exits.
func (sw *SpinnerWriter) Wait() {
	if sw.started {
		<-sw.done
	}
}
