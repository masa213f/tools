package util

import (
	"fmt"
	"os/exec"
	"strings"
)

func ExecCmd(cmd ...string) error {
	stdoutStderr, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()

	var b strings.Builder
	fmt.Fprintf(&b, "RUN: %s\n", strings.Join(cmd, " "))
	if err != nil {
		fmt.Fprintf(&b, "ERROR: %v\n", err)
	}
	if len(stdoutStderr) != 0 {
		fmt.Fprintln(&b, string(stdoutStderr))
	}
	fmt.Print(b.String())

	if err != nil {
		return fmt.Errorf("failed to exec %s; %w", cmd, err)
	}
	return nil
}
