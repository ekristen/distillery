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
	mu       sync.Mutex
	multi    pterm.MultiPrinter
	inactive int
	spinner  map[string]*pterm.SpinnerPrinter
}

// NewSpinnerWriter creates a new SpinnerWriter.
func NewWriter() *SpinnerWriter {
	return &SpinnerWriter{
		multi:   pterm.DefaultMultiPrinter,
		spinner: map[string]*pterm.SpinnerPrinter{},
	}
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

	if app == "" {
		return 0, nil
	}

	if !sw.multi.IsActive {
		_, _ = sw.multi.Start()
	}

	var appSpinner *pterm.SpinnerPrinter
	if app != "" {
		appSpinner = sw.spinner[app]
		if appSpinner == nil {
			var err error
			appSpinner, err = pterm.DefaultSpinner.WithWriter(sw.multi.NewWriter()).Start(app + ": " + msg)
			if err != nil {
				return 0, err
			}
			sw.spinner[app] = appSpinner
		}
	}

	// Bold the app name if present
	var prefix string
	if app != "" {
		prefix = pterm.Style{pterm.Bold}.Sprint(app) + ": "
	}
	fullMsg := prefix + msg

	appSpinner.UpdateText(fullMsg)

	// Handle completion states
	switch {
	case event["success"] == true:
		appSpinner.Success(fullMsg)
	case event["fail"] == true:
		appSpinner.Fail(fullMsg)
	case event["warn"] == true:
		appSpinner.Warning(fullMsg)
	case event["ok"] == true:
		appSpinner.InfoPrinter = &pterm.PrefixPrinter{
			MessageStyle: &pterm.Style{pterm.FgLightGreen},
			Prefix: pterm.Prefix{
				Style: &pterm.Style{pterm.FgBlack, pterm.BgLightGreen},
				Text:  "OK",
			},
		}
		appSpinner.Info(fullMsg)
	case event["done"] == true:
		_ = appSpinner.Stop()
	}

	if !appSpinner.IsActive {
		sw.inactive++
	}

	if sw.multi.IsActive && sw.inactive == len(sw.spinner) {
		_, _ = sw.multi.Stop()
	}

	return len(p), nil
}
