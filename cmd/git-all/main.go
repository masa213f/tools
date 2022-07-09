package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/masa213f/tools/pkg/util"
)

var options struct {
	targetRoots []string
	command     string
	commandArgs []string
}

const usage = `Usage: git-all [ [ -c COMMAND ] TARGET_ROOT...  -- ] COMMAND_ARGS`

func main() {
	myArgs := []string{}
	options.commandArgs = os.Args[1:]
	for i, a := range os.Args {
		if a == "--" {
			myArgs = os.Args[1:i]
			options.commandArgs = os.Args[i+1:]
			break
		}
	}

	flagSet := flag.NewFlagSet("", flag.ExitOnError)
	flagSet.StringVar(&options.command, "c", "git", "command")
	err := flagSet.Parse(myArgs)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	options.targetRoots = flagSet.Args()

	fmt.Printf("TARGET_ROOTS: %v\n", options.targetRoots)
	fmt.Printf("COMMAND: %s\n", options.command)
	fmt.Printf("COMMAND_ARGS: %v\n", options.commandArgs)
	fmt.Println()

	err = subMain(options.targetRoots, options.command, options.commandArgs)
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func findWorkingDirs(rootDir string) ([]string, error) {
	dirs := []string{}
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".git" {
			abs, err := filepath.Abs(filepath.Dir(path))
			if err != nil {
				return err
			}
			dirs = append(dirs, abs)
			// Skip other files in the directory
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to filepath.Walk: %w", err)
	}
	return dirs, nil
}

func subMain(rootDirs []string, cmd string, args []string) error {
	targets := []string{}
	tmp := map[string]struct{}{}
	for _, r := range rootDirs {
		dirs, err := findWorkingDirs(r)
		if err != nil {
			return err
		}
		for _, d := range dirs {
			tmp[d] = struct{}{}
		}
	}
	for k := range tmp {
		targets = append(targets, k)
	}

	var wg sync.WaitGroup
	for _, t := range targets {
		targetDir := t
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command(cmd, args...)
			cmd.Dir = targetDir
			util.Run(cmd)
		}()
	}
	wg.Wait()

	return nil
}
