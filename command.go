package autohand

import (
	"fmt"
	"strings"
)

// FormatSlashCommand validates and formats a CLI slash command.
func FormatSlashCommand(command string, args ...string) (string, error) {
	command = strings.TrimSpace(command)
	if !strings.HasPrefix(command, "/") || strings.ContainsAny(command, " \t\r\n") {
		return "", fmt.Errorf("invalid slash command %q", command)
	}
	clean := make([]string, 0, len(args))
	for _, arg := range args {
		if arg = strings.TrimSpace(arg); arg != "" {
			clean = append(clean, arg)
		}
	}
	if len(clean) == 0 {
		return command, nil
	}
	return command + " " + strings.Join(clean, " "), nil
}
