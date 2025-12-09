package agent

import (
	"bytes"
	"os/exec"
)

func RunBash(cmd string) (string, error) {
	c := exec.Command("bash", "-c", cmd)

	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out

	err := c.Run()
	return out.String(), err
}
