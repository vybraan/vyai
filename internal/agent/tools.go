package agent

import (
	"bytes"
	"errors"
	"fmt"
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
	if pattern == "" {
		return "", errors.New("grep pattern is required")
	}

	safePath, err := ResolveWorkspacePath(workspace, path)
	if err != nil {
		return "", err
	}

	args := []string{"--recursive"}
	if include != "" {
		args = append(args, "--include", include)
	}
	args = append(args, pattern, safePath)

	out, err := exec.Command("grep", args...).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func GlobFile(pattern, path, workspace string) (string, error) {
	if pattern == "" {
		return "", errors.New("glob pattern is required")
	}
	if filepath.IsAbs(pattern) {
		return "", errors.New("absolute patterns are not allowed")
	}

	safePath, err := ResolveWorkspacePath(workspace, path)
	if err != nil {
		return "", err
	}

	cleanPattern := filepath.Clean(pattern)
	if cleanPattern == ".." ||
		strings.HasPrefix(cleanPattern, ".."+string(os.PathSeparator)) ||
		strings.Contains(cleanPattern, string(os.PathSeparator)+".."+string(os.PathSeparator)) ||
		strings.HasSuffix(cleanPattern, string(os.PathSeparator)+"..") {
		return "", errors.New("glob pattern escapes the workspace")
	}

	globPattern := filepath.Join(safePath, cleanPattern)
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return "", err
	}

	filtered := make([]string, 0, len(matches))
	for _, match := range matches {
		rel, err := filepath.Rel(workspace, match)
		if err != nil {
			return "", err
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return "", errors.New("glob match escapes the workspace")
		}
		filtered = append(filtered, match)
	}
	return strings.Join(filtered, "\n"), nil
}

func EditFile(path, oldString, newString string) error {
	if oldString == "" {
		return errors.New("old string is required")
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	newContent := strings.Replace(string(content), oldString, newString, 1)
	if newContent == string(content) {
		return errors.New("old string not found or no change")
	}

	return os.WriteFile(path, []byte(newContent), fileInfo.Mode())
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
