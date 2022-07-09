package util

import (
	"fmt"
	"io"
	"net/http"
	"os"
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

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func DownloadFile(url string, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	size, err := io.Copy(file, resp.Body)
	fmt.Printf("downloaded file %s (size %d)\n", filePath, size)
	return err
}
