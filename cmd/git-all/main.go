package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/masa213f/tools/pkg/util"
)

var gitCmd = "git"

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: git-all TARGET_ROOT GIT_ARGS")
		os.Exit(1)
	}
	targetRoot, _ := filepath.Abs(os.Args[1])
	gitArgs := os.Args[2:]

	if c := os.Getenv("GIT_COMMAND"); c != "" {
		gitCmd = c
	}

	fmt.Printf("TARGET_ROOT: %s\n", targetRoot)
	fmt.Printf("GIT_COMMAND: %s\n", gitCmd)
	fmt.Printf("GIT_ARGS: %v\n", gitArgs)

	err := subMain(targetRoot, gitArgs)
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func subMain(targetRoot string, gitArgs []string) error {
	targets := []string{}
	err := filepath.Walk(targetRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".git" {
			targets = append(targets, filepath.Dir(path))
			// Skip other files in the directory
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to filepath.Walk: %w", err)
	}

	var wg sync.WaitGroup
	for _, t := range targets {
		targetDir := t
		wg.Add(1)
		go func() {
			defer wg.Done()
			util.ExecCmd(append([]string{gitCmd, "-C", targetDir}, gitArgs...)...)
		}()
	}
	wg.Wait()

	return nil
}
