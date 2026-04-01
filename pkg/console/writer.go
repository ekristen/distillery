package console

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/pterm/pterm"
	"github.com/rs/zerolog"
)

// Writer implements zerolog.LevelWriter and io.Writer for clean console output.
// It produces human-friendly output without timestamps, callers, or field dumps.
type Writer struct {
	out io.Writer
}

// NewWriter creates a new console Writer.
func NewWriter() *Writer {
	return &Writer{out: os.Stderr}
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

	var prefix string
	if app != "" {
		prefix = pterm.Bold.Sprint(app) + ": "
	}

	var line string
	switch {
	case event["success"] == true:
		line = pterm.FgGreen.Sprint("✓ ") + prefix + msg
	case event["fail"] == true:
		line = pterm.FgRed.Sprint("✗ ") + prefix + msg
	case event["warn"] == true:
		line = pterm.FgYellow.Sprint("! ") + prefix + msg
	case event["ok"] == true:
		line = pterm.FgGreen.Sprint("✓ ") + prefix + msg
	case level == zerolog.WarnLevel:
		line = pterm.FgYellow.Sprint("! ") + prefix + msg
	case level == zerolog.ErrorLevel:
		line = pterm.FgRed.Sprint("✗ ") + prefix + msg
	default:
		line = "  " + prefix + msg
	}

	fmt.Fprintln(w.out, line)

	return len(p), nil
}
