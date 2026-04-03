package distfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Command represents a parsed command from the Distfile.
type Command struct {
	Action string
	Args   []string
}

// Parse parses the given Distfile and returns a list of commands.
func Parse(filePath string) ([]Command, error) {
	absPath, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return nil, fmt.Errorf("invalid distfile path: %w", err)
	}
	baseDir := filepath.Dir(absPath)
	return parseWithTracking(absPath, baseDir, make(map[string]struct{}))
}

func parseWithTracking(filePath, baseDir string, seen map[string]struct{}) ([]Command, error) {
	if _, processed := seen[filePath]; processed {
		return nil, fmt.Errorf("circular inclusion detected: %s", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	seen[filePath] = struct{}{}
	currentDir := filepath.Dir(filePath)

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
			includePath, err := resolveIncludePath(args[0], currentDir, baseDir)
			if err != nil {
				return nil, fmt.Errorf("invalid file inclusion %q: %w", args[0], err)
			}
			subCommands, err := parseWithTracking(includePath, baseDir, seen)
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

	return commands, nil
}

// resolveIncludePath resolves and validates an included file path. Relative paths
// are resolved against currentDir (the directory of the file containing the directive).
// The resolved path must not escape baseDir (the root Distfile's directory).
func resolveIncludePath(raw, currentDir, baseDir string) (string, error) {
	cleaned := filepath.Clean(raw)
	if !filepath.IsAbs(cleaned) {
		cleaned = filepath.Join(currentDir, cleaned)
	}
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}

	// Ensure the resolved path is within the base directory
	if !strings.HasPrefix(abs, baseDir+string(filepath.Separator)) && abs != baseDir {
		return "", fmt.Errorf("path %q escapes base directory %q", raw, baseDir)
	}

	return abs, nil
}
