package agent

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
)

func SecureJoin(base, rel string) (string, error) {
	base = filepath.Clean(base)
	target := filepath.Clean(filepath.Join(base, rel))

	if target != base && !strings.HasPrefix(target, base+string(filepath.Separator)) {
		return "", errors.New("path escape blocked")
	}

	return target, nil
}

func ResolveWorkspacePath(base, rel string) (string, error) {
	if rel == "" || rel == "." {
		return filepath.Clean(base), nil
	}
	if filepath.IsAbs(rel) {
		return "", errors.New("absolute paths are not allowed")
	}

	return SecureJoin(base, rel)
}

func ParseCommand(cmd string) ([]string, error) {
	fields := strings.Fields(strings.TrimSpace(cmd))
	if len(fields) == 0 {
		return nil, errors.New("empty command")
	}

	for _, field := range fields {
		if strings.ContainsAny(field, "&;|<>`$()\\\n\r'\"*?[]{}!#~") {
			return nil, errors.New("shell metacharacters are not allowed")
		}
	}

	return fields, nil
}

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
