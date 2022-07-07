package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/masa213f/tools/pkg/util"
)

var kustomizeCmd = "kustomize"

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: kustomize-build-all TARGET_DIR OUTPUT_DIR")
		os.Exit(1)
	}
	targetRoot, _ := filepath.Abs(os.Args[1])
	outputRoot, _ := filepath.Abs(os.Args[2])

	if c := os.Getenv("KUSTOMIZE_COMMAND"); c != "" {
		kustomizeCmd = c
	}

	fmt.Printf("TARGET_DIR: %s\n", targetRoot)
	fmt.Printf("OUTPUT_DIR: %s\n", outputRoot)
	fmt.Printf("KUSTOMIZE_COMMAND: %s\n", kustomizeCmd)

	err := subMain(targetRoot, outputRoot)
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func subMain(targetRoot, outputRoot string) error {
	targets := []string{}
	err := filepath.Walk(targetRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !isKustomizationFile(info.Name()) {
			return nil
		}
		targets = append(targets, filepath.Dir(path))
		// Skip other files in the directory
		return filepath.SkipDir
	})
	if err != nil {
		return fmt.Errorf("failed to filepath.Walk: %w", err)
	}

	job := map[string]string{}
	for _, targetDir := range targets {
		outputDir := strings.Replace(targetDir, targetRoot, outputRoot, 1)
		err = os.MkdirAll(outputDir, 0775)
		if err != nil {
			return fmt.Errorf("failed to create output directory. path: %s, err: %w", outputDir, err)
		}
		job[targetDir] = filepath.Join(outputDir, "build.yaml")
	}

	var wg sync.WaitGroup
	for targetDir, outputFile := range job {
		targetDir := targetDir
		outputFile := outputFile
		wg.Add(1)
		go func() {
			defer wg.Done()
			util.ExecCmd(kustomizeCmd, "build", "--enable-helm", targetDir, "-o", outputFile)
		}()
	}
	wg.Wait()

	return nil
}

func isKustomizationFile(name string) bool {
	return name == "kustomization.yaml" || name == "kustomization.yml" || name == "Kustomization"
}
