package agent

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
)

// SecureJoin returns the cleaned path obtained by joining base and rel while ensuring
// the result is either exactly base or a path contained within base. If the computed
// target escapes the base directory (is neither base nor a subpath of base), it
// returns an error "path escape blocked".
func SecureJoin(base, rel string) (string, error) {
	base = filepath.Clean(base)
	target := filepath.Clean(filepath.Join(base, rel))

	if target != base && !strings.HasPrefix(target, base+string(filepath.Separator)) {
		return "", errors.New("path escape blocked")
	}

	return target, nil
}

// ResolveWorkspacePath resolves a workspace-relative path `rel` against `base` and returns a cleaned, validated path.
// If `rel` is empty or ".", ResolveWorkspacePath returns the cleaned `base`.
// If `rel` is an absolute path, it returns an error ("absolute paths are not allowed").
// Otherwise it returns the cleaned join of `base` and `rel`, or an error if the resulting path escapes `base`.
func ResolveWorkspacePath(base, rel string) (string, error) {
	if rel == "" || rel == "." {
		return filepath.Clean(base), nil
	}
	if filepath.IsAbs(rel) {
		return "", errors.New("absolute paths are not allowed")
	}

	return SecureJoin(base, rel)
}

// ParseCommand splits cmd into whitespace-separated fields and rejects empty input
// or fields that contain shell metacharacters.
//
// If cmd is empty or contains only whitespace, it returns an error.
// If any field contains any of the characters & ; | < > ` $ ( ) newline or carriage return,
// it returns an error. On success it returns the slice of fields.
func ParseCommand(cmd string) ([]string, error) {
	fields := strings.Fields(strings.TrimSpace(cmd))
	if len(fields) == 0 {
		return nil, errors.New("empty command")
	}

	for _, field := range fields {
		if strings.ContainsAny(field, "&;|<>`$()\n\r") {
			return nil, errors.New("shell metacharacters are not allowed")
		}
	}

	return fields, nil
}

// AllowCommand reports whether the parsed command arguments are permitted by the server's allowlist.
// It returns true for the commands "ls", "cat", "gofmt", and "goimports". For "go" it returns true only
// when a subcommand is present and is one of "build", "test", or "vet". It returns false for empty input
// or any other command.
func AllowCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}

	switch args[0] {
	case "ls", "cat", "gofmt", "goimports":
		return true
	case "go":
		return len(args) > 1 && slices.Contains([]string{"build", "test", "vet"}, args[1])
	default:
		return false
	}
}
