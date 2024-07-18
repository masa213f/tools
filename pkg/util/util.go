package util

import (
	"fmt"
	"os/exec"
	"strings"
)

func Run(c *exec.Cmd) error {
	stdoutStderr, err := c.CombinedOutput()

	var b strings.Builder
	fmt.Fprintf(&b, "RUN: %s\n", strings.Join(c.Args, " "))
	if len(c.Dir) != 0 {
		fmt.Fprintf(&b, "IN: %s\n", c.Dir)
	}
	if err != nil {
		fmt.Fprintf(&b, "ERROR: %v\n", err)
	}
	if len(stdoutStderr) != 0 {
		fmt.Fprintln(&b, string(stdoutStderr))
	} else {
		fmt.Fprintln(&b, "")
	}
	fmt.Print(b.String())

	if err != nil {
		return fmt.Errorf("failed to exec %s; %w", c.Args, err)
	}
	return nil
}
