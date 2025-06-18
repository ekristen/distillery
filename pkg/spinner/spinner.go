package spinner

import (
	"encoding/json"
	"sync"

	"github.com/pterm/pterm"
	"github.com/rs/zerolog"
)

// SpinnerWriter implements zerolog.LevelWriter and io.Writer for spinner output.
// Each SpinnerWriter manages its own spinner instance.
type SpinnerWriter struct {
	spinner    *pterm.SpinnerPrinter
	mu         sync.Mutex
	active     bool
	prevMsgLen int
}

// NewSpinnerWriter creates a new SpinnerWriter.
func NewWriter() *SpinnerWriter {
	return &SpinnerWriter{}
}

// Write implements io.Writer.
func (sw *SpinnerWriter) Write(p []byte) (n int, err error) {
	return sw.WriteLevel(zerolog.InfoLevel, p)
}

// WriteLevel implements zerolog.LevelWriter.
func (sw *SpinnerWriter) WriteLevel(level zerolog.Level, p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	// Parse zerolog JSON message
	var event map[string]interface{}
	_ = json.Unmarshal(p, &event)

	msg, _ := event["message"].(string)
	app, _ := event["app"].(string)

	// Bold the app name if present
	var prefix string
	if app != "" {
		prefix = pterm.Style{pterm.Bold}.Sprint(app) + ": "
	}
	fullMsg := prefix + msg

	// Pad the message to clear any trailing characters from previous longer messages
	displayMsg := fullMsg
	if sw.prevMsgLen > len(fullMsg) {
		padding := make([]rune, sw.prevMsgLen-len(fullMsg))
		for i := range padding {
			padding[i] = ' '
		}
		displayMsg = fullMsg + string(padding)
	}
	sw.prevMsgLen = len(fullMsg)

	if !sw.active {
		var err error
		sw.spinner, err = pterm.DefaultSpinner.Start(displayMsg)
		if err != nil {
			return 0, nil
		}
		sw.active = true
	} else {
		sw.spinner.UpdateText(displayMsg)
	}

	// Handle completion states
	switch {
	case event["success"] == true:
		sw.spinner.Success(fullMsg)
		sw.active = false
		sw.prevMsgLen = 0
	case event["fail"] == true:
		sw.spinner.Fail(fullMsg)
		sw.active = false
		sw.prevMsgLen = 0
	case event["warn"] == true:
		sw.spinner.Warning(fullMsg)
		sw.active = false
		sw.prevMsgLen = 0
	case event["done"] == true:
		_ = sw.spinner.Stop()
		sw.active = false
		sw.prevMsgLen = 0
	}

	return len(p), nil
}
