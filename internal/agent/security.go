package agent

import (
	"errors"
	"path/filepath"
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

func AllowCommand(cmd string) bool {
	allowed := []string{"ls", "cat", "go", "git", "npm", "grep", "glob", "mv", "rm", "goimports", "gofmt", "go build", "go test", "go vet"}
	for _, a := range allowed {
		if strings.HasPrefix(cmd, a+" ") || cmd == a {
			return true
		}
	}
	return false
}
