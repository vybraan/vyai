package agent

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RunBash(cmd string) (string, error) {
	c := exec.Command("bash", "-c", cmd)

	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out

	err := c.Run()
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

func GrepFile(pattern, include, path string) (string, error) {
	args := []string{"--recursive"}
	if pattern != "" {
		args = append(args, pattern)
	}
	if include != "" {
		args = append(args, "--include", include)
	}
	if path != "" {
		args = append(args, path)
	}

	out, err := exec.Command("grep", args...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func GlobFile(pattern, path string) (string, error) {
	args := []string{}
	if path != "" {
		args = append(args, path)
	}

	// Use filepath.Glob for simpler globbing
	matches, err := filepath.Glob(filepath.Join(path, pattern))
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
