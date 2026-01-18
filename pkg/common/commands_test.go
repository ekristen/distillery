package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestCommands(t *testing.T) {
	cmd := &cli.Command{
		Name: "test",
	}
	RegisterCommand(cmd)
	commands := GetCommands()

	assert.Len(t, commands, 1)
}
