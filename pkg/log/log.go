package log

import (
	"github.com/rs/zerolog"
)

// LevelFromString parses a string into a zerolog.Level, defaulting to InfoLevel.
func LevelFromString(level string) zerolog.Level {
	l, err := zerolog.ParseLevel(level)
	if err != nil {
		return zerolog.InfoLevel
	}
	return l
}
