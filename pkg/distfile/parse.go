package distfile

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Command represents a parsed command from the Distfile.
type Command struct {
	Action string
	Args   []string
}

var processedFiles = make(map[string]struct{})

// Parse parses the given Distfile and returns a list of commands.
func Parse(filePath string) ([]Command, error) {
	if _, processed := processedFiles[filePath]; processed {
		return nil, fmt.Errorf("circular inclusion detected: %s", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	processedFiles[filePath] = struct{}{}

	var commands []Command
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		action := parts[0]
		args := parts[1:]

		switch action {
		case "distill", "install", "dist":
			commands = append(commands, Command{Action: "install", Args: args})
		case "distfile", "file":
			if len(args) != 1 {
				return nil, fmt.Errorf("file command requires exactly one argument")
			}
			subCommands, err := Parse(args[0])
			if err != nil {
				return nil, err
			}
			commands = append(commands, subCommands...)
		default:
			return nil, fmt.Errorf("unknown command: %s", action)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	processedFiles = make(map[string]struct{})

	return commands, nil
}
