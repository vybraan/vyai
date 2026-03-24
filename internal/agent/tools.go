package agent

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RunBash(cmd, workspace string) (string, error) {
	args, err := ParseCommand(cmd)
	if err != nil {
		return "", err
	}
	if !AllowCommand(args) {
		return "", errors.New("command is not allowed")
	}

	argv, err := prepareCommandArgs(args, workspace)
	if err != nil {
		return "", err
	}

	c := exec.Command(argv[0], argv[1:]...)
	c.Dir = workspace

	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out

	err = c.Run()
	return out.String(), err
}

func ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func ListDir(path string) (string, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	var entries []string
	for _, file := range files {
		entries = append(entries, file.Name())
	}
	return strings.Join(entries, "\n"), nil
}

func GrepFile(pattern, include, path, workspace string) (string, error) {
	safePath, err := ResolveWorkspacePath(workspace, path)
	if err != nil {
		return "", err
	}

	args := []string{"--recursive"}
	if pattern != "" {
		args = append(args, pattern)
	}
	if include != "" {
		args = append(args, "--include", include)
	}
	args = append(args, safePath)

	out, err := exec.Command("grep", args...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func GlobFile(pattern, path, workspace string) (string, error) {
	safePath, err := ResolveWorkspacePath(workspace, path)
	if err != nil {
		return "", err
	}

	// Use filepath.Glob for simpler globbing
	matches, err := filepath.Glob(filepath.Join(safePath, pattern))
	if err != nil {
		return "", err
	}
	return strings.Join(matches, "\n"), nil
}

func EditFile(path, oldString, newString string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	newContent := strings.Replace(string(content), oldString, newString, 1)
	if newContent == string(content) {
		return errors.New("old string not found or no change")
	}

	return os.WriteFile(path, []byte(newContent), 0644)
}

func prepareCommandArgs(args []string, workspace string) ([]string, error) {
	switch args[0] {
	case "ls", "cat", "gofmt", "goimports":
		return rewritePathArgs(args, workspace, args[0] == "cat")
	case "go":
		return validateGoArgs(args)
	default:
		return nil, errors.New("unsupported command")
	}
}

func rewritePathArgs(args []string, workspace string, requirePath bool) ([]string, error) {
	rewritten := []string{args[0]}
	pathCount := 0

	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") {
			rewritten = append(rewritten, arg)
			continue
		}

		pathCount++
		safePath, err := ResolveWorkspacePath(workspace, arg)
		if err != nil {
			return nil, err
		}
		rewritten = append(rewritten, safePath)
	}

	if requirePath && pathCount == 0 {
		return nil, errors.New("command requires at least one path")
	}

	if !requirePath && pathCount == 0 {
		rewritten = append(rewritten, workspace)
	}

	return rewritten, nil
}

func validateGoArgs(args []string) ([]string, error) {
	if len(args) < 2 {
		return nil, errors.New("go subcommand is required")
	}

	for _, arg := range args[2:] {
		if filepath.IsAbs(arg) {
			return nil, errors.New("absolute paths are not allowed")
		}
	}

	return args, nil
}
