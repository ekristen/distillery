package console

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
)

// appColors are distinct colors assigned to apps for visual separation in interleaved output.
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

var (
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

// Writer implements zerolog.LevelWriter and io.Writer for clean console output.
// It produces human-friendly output without timestamps, callers, or field dumps.
type Writer struct {
	mu       sync.Mutex
	out      io.Writer
	appIndex map[string]int
}

// NewWriter creates a new console Writer.
func NewWriter() *Writer {
	return &Writer{
		out:      os.Stderr,
		appIndex: make(map[string]int),
	}
}

func (w *Writer) appStyle(app string) lipgloss.Style {
	idx, exists := w.appIndex[app]
	if !exists {
		idx = len(w.appIndex) % len(appColors)
		w.appIndex[app] = idx
	}
	return lipgloss.NewStyle().Bold(true).Foreground(appColors[idx])
}

// Write implements io.Writer.
func (w *Writer) Write(p []byte) (n int, err error) {
	return w.WriteLevel(zerolog.InfoLevel, p)
}

// WriteLevel implements zerolog.LevelWriter.
func (w *Writer) WriteLevel(level zerolog.Level, p []byte) (n int, err error) {
	var event map[string]interface{}
	if err := json.Unmarshal(p, &event); err != nil {
		return w.out.Write(p)
	}

	msg, _ := event["message"].(string)
	app, _ := event["app"].(string)

	if msg == "" {
		return len(p), nil
	}

	w.mu.Lock()
	var prefix string
	if app != "" {
		prefix = w.appStyle(app).Render(app) + ": "
	}
	w.mu.Unlock()

	var line string
	switch {
	case event["success"] == true:
		line = greenStyle.Render("✓ ") + prefix + greenStyle.Render(msg)
	case event["fail"] == true:
		line = redStyle.Render("✗ ") + prefix + msg
	case event["warn"] == true:
		line = yellowStyle.Render("! ") + prefix + msg
	case event["hint"] == true:
		line = cyanStyle.Render("? ") + prefix + msg
	case event["ok"] == true:
		line = greenStyle.Render("✓ ") + prefix + msg
	case level == zerolog.WarnLevel:
		line = yellowStyle.Render("! ") + prefix + msg
	case level == zerolog.ErrorLevel:
		line = redStyle.Render("✗ ") + prefix + msg
	default:
		line = "  " + prefix + msg
	}

	fmt.Fprintln(w.out, line)

	return len(p), nil
}
