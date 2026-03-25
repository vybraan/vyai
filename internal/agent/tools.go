package agent

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RunBash executes a parsed shell-style command inside the specified workspace.
// 
// It parses and validates the provided command string, enforces allowed commands,
// rewrites or validates arguments to ensure they are safe within the workspace when
// applicable, and runs the resolved executable with the workspace as the working
// directory. Standard output and standard error are captured together; the function
// returns the combined output and any execution error encountered.
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

// ReadFile reads the file at the given path and returns its contents as a string.
// It returns the file contents, or an error if the file could not be read.
func ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ListDir reads the directory named by path and returns a newline-separated list of entry names.
// It returns a non-nil error if the directory cannot be read.
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

// GrepFile runs the `grep` command recursively against the path resolved relative to the workspace,
// optionally filtering by `pattern` and `--include`, and returns grep's combined output or an error.
 // The returned string contains both stdout and stderr from the grep invocation; an error is returned if path resolution or command execution fails.
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

// GlobFile returns newline-separated filesystem paths that match the glob pattern under the workspace-resolved path.
// It resolves the provided path against the workspace and returns an error if path resolution or globbing fails.
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

// EditFile replaces the first occurrence of oldString with newString in the file at path.
// It returns an error if the file cannot be read or written, or if oldString is not found
// (i.e., the file content would be unchanged).
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

// prepareCommandArgs rewrites or validates command-line arguments for supported commands
// using the provided workspace as the base for any path arguments.
//
// For "ls", "cat", "gofmt", and "goimports" it resolves and rewrites non-flag arguments as
// workspace-scoped paths (and requires at least one path for "cat"). For "go" it validates
// the go subcommand arguments (ensuring a subcommand is present and disallowing absolute paths).
// Returns the rewritten/validated argument slice or an error for unsupported or invalid input.
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

// rewritePathArgs rewrites command arguments by resolving non-flag path arguments against the workspace.
// It preserves arguments that start with "-" as flags and replaces other arguments with ResolveWorkspacePath(workspace, arg).
// If requirePath is true and no non-flag path arguments are present, it returns an error.
// If requirePath is false and no non-flag path arguments are present, it appends the workspace path as the default target.
// Returns the rewritten argument slice, or an error from ResolveWorkspacePath or when a required path is missing.
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

// validateGoArgs validates arguments intended for the `go` command.
// It requires a subcommand (at least one argument after "go") and rejects any
// absolute file system paths in subsequent arguments. On success it returns the
// original argument slice; on failure it returns a descriptive error.
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
