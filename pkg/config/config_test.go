package config

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigNewYAML(t *testing.T) {
	cfg, err := New("testdata/base.yaml")
	assert.NoError(t, err)

	assert.Equal(t, "/home/test/.distillery", cfg.Path)
	assert.Equal(t, "/home/test/.cache", cfg.CachePath)

	aliases := &Aliases{
		"dist": &Alias{
			Name:    "ekristen/distillery",
			Version: "latest",
		},
		"aws-nuke": &Alias{
			Name:    "ekristen/aws-nuke",
			Version: "3.29.3",
		},
	}

	assert.EqualValues(t, aliases, cfg.Aliases)
}

func TestConfigNewTOML(t *testing.T) {
	cfg, err := New("testdata/base.toml")
	assert.NoError(t, err)

	assert.Equal(t, "/home/test/.distillery", cfg.Path)
	assert.Equal(t, "/home/test/.cache", cfg.CachePath)

	aliases := &Aliases{
		"dist": &Alias{
			Name:    "ekristen/distillery",
			Version: "latest",
		},
		"aws-nuke": &Alias{
			Name:    "ekristen/aws-nuke",
			Version: "3.29.3",
		},
	}

	assert.EqualValues(t, aliases, cfg.Aliases)
}

func TestProcessPath(t *testing.T) {
	homePath, _ := os.UserHomeDir()
	result := processPath("$HOME/.config/test")
	assert.Equal(t, path.Join(homePath, ".config/test"), result)

	result = processPath("/test/..")
	assert.Equal(t, "/", result)

	os.Setenv("TEST", "value")
	result = processPath("/$TEST/path")
	assert.Equal(t, "/value/path", result)

	result = processPath("/$NAENV/test")
	assert.Equal(t, "/test", result)

	cwd, _ := os.Getwd()
	result = processPath("test/path")
	assert.Equal(t, path.Join(cwd, "test/path"), result)

	result = processPath("/test//path")
	assert.Equal(t, "/test/path", result)
}
