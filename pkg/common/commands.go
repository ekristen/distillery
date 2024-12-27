package common

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var commands = make(map[string][]*cli.Command, 0)

// RegisterCommand -- allows you to register a command under the main group
func RegisterCommand(command *cli.Command) {
	logrus.Debugln("Registering", command.Name, "command...")
	commands["_main_"] = append(commands["_main_"], command)
}

// GetCommands -- retrieves all commands assigned to the main group
func GetCommands() []*cli.Command {
	return commands["_main_"]
}

func GetCommand(name string) *cli.Command {
	for _, command := range commands["_main_"] {
		if command.Name == name {
			return command
		}
	}
	return nil
}
