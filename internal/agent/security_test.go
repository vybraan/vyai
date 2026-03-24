package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCommandRejectsShellMetacharacters(t *testing.T) {
	t.Parallel()

	_, err := ParseCommand("ls; rm -rf .")
	if err == nil {
		t.Fatal("expected shell metacharacters to be rejected")
	}
}

func TestAllowCommandRejectsDestructiveCommands(t *testing.T) {
	t.Parallel()

	if AllowCommand([]string{"rm", "-rf", "."}) {
		t.Fatal("expected rm to be blocked")
	}
}

func TestRunBashRejectsAbsolutePaths(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	_, err := RunBash("cat /etc/passwd", workspace)
	if err == nil {
		t.Fatal("expected absolute path access to be blocked")
	}
}

func TestGrepFileRejectsPathEscape(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	parentFile := filepath.Join(filepath.Dir(workspace), "outside.txt")
	if err := os.WriteFile(parentFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("write parent file: %v", err)
	}

	_, err := GrepFile("secret", "*.txt", "../", workspace)
	if err == nil {
		t.Fatal("expected grep path escape to be blocked")
	}
}

func TestRunBashReadsFilesInsideWorkspace(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	path := filepath.Join(workspace, "note.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	out, err := RunBash("cat note.txt", workspace)
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if strings.TrimSpace(out) != "hello" {
		t.Fatalf("unexpected output: %q", out)
	}
}
