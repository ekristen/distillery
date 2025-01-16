package config

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigNewYAML(t *testing.T) {
	cases := []struct {
		path string
	}{
		{"testdata/base.yaml"},
		{"testdata/base.toml"},
	}

	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			cfg, err := New(c.path)
			assert.NoError(t, err)

			assert.Equal(t, "/home/test/.distillery", cfg.GetPath())
			assert.Equal(t, "/home/test/.cache/distillery", cfg.GetCachePath())
			assert.Equal(t, "/home/test/.distillery/opt", cfg.GetOptPath())
			assert.Equal(t, "/home/test/.cache/distillery/metadata", cfg.GetMetadataPath())
			assert.Equal(t, "/home/test/.cache/distillery/downloads", cfg.GetDownloadsPath())

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
			assert.Equal(t, "latest", cfg.GetAlias("dist").Version)
			assert.Equal(t, "3.29.3", cfg.GetAlias("aws-nuke").Version)
		})
	}
}

func TestDefaultAlias(t *testing.T) {
	cfg, err := New("")
	assert.NoError(t, err)

	assert.Equal(t, "latest", cfg.GetAlias("dist").Version)
}

func TestDefaultAliasAdd(t *testing.T) {
	cfg, err := New("testdata/default-aliases.yaml")
	assert.NoError(t, err)

	assert.Equal(t, "latest", cfg.GetAlias("dist").Version)
	assert.Equal(t, "latest", cfg.GetAlias("aws-nuke").Version)
}

func TestDefaultAliasNoOverride(t *testing.T) {
	cfg, err := New("testdata/default-aliases-no-override.yaml")
	assert.NoError(t, err)

	assert.Equal(t, "someother/project", cfg.GetAlias("dist").Name)
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
