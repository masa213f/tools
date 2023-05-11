package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Key: name, Value: tag
var permittedActions = map[string]string{
	"actions/cache":                      "v3",
	"actions/checkout":                   "v3",
	"actions/download-artifact":          "v3",
	"actions/setup-go":                   "v4",
	"actions/setup-python":               "v4",
	"actions/upload-artifact":            "v3",
	"azure/setup-helm":                   "v3",
	"docker/build-push-action":           "v4",
	"docker/login-action":                "v2",
	"docker/metadata-action":             "v4",
	"docker/setup-buildx-action":         "v2",
	"docker/setup-qemu-action":           "v2",
	"google-github-actions/auth":         "v1",
	"google-github-actions/setup-gcloud": "v1",
	"goreleaser/goreleaser-action":       "f82d6c1c344bcacabba2c841718984797f664a6b", // v4.2.0
	"helm/chart-testing-action":          "v2.4.0",
	"helm/kind-action":                   "v1.5.0",
	"rajatjindal/krew-release-bot":       "df3eb197549e3568be8b4767eec31c5e8e8e6ad8", // 0.0.46
}

// Key: name, Value: reason
var prohibitedActions = map[string]string{
	"actions/create-release":       "archived",
	"actions/upload-release-asset": "archived",
}

var (
	findActionRe    = regexp.MustCompile(`.*uses:\s*(.+)\s*`)
	replaceActionRe = regexp.MustCompile(`(.*uses:[^@]+@)\S+(\s*)`)
)

func findAction(line string) string {
	matches := findActionRe.FindStringSubmatch(line)
	if len(matches) == 0 {
		return ""
	}
	return matches[1]
}

func replaceActionTag(line, action string) string {
	return replaceActionRe.ReplaceAllString(line, "${1}"+action+"${2}")
}

func readFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	contents := []string{}
	sc := bufio.NewScanner(file)
	for sc.Scan() {
		contents = append(contents, sc.Text())
	}
	err = sc.Err()
	if err != nil {
		return nil, err
	}
	return contents, nil
}

func main() {
	flag.Parse()
	if flag.NArg() > 1 {
		fmt.Println("Usage: update-actions [TARGET_DIR]")
		os.Exit(1)
	}
	targetDir := "."
	if flag.NArg() == 1 {
		targetDir = flag.Arg(0)
	}

	workflowFiles := []string{}
	rootDir := filepath.Join(targetDir, ".github")
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !isYamlFile(info.Name()) {
			return nil
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		workflowFiles = append(workflowFiles, abs)
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "filepath.Walk: %v", err)
		os.Exit(1)
	}

	for _, path := range workflowFiles {
		fmt.Println(path)
		contents, err := readFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "readFile: %v", err)
			return
		}

		for i, line := range contents {
			action := findAction(line)
			if action == "" {
				continue
			}

			split := strings.SplitN(action, "@", 2)
			if len(split) != 2 {
				fmt.Printf("%4d, Error. unknown format: %s\n", i, action)
				continue
			}
			name := split[0]
			currentVersion := split[1]

			if reason, ok := prohibitedActions[name]; ok {
				fmt.Printf("%4d, Error. prohibited (%s): %s\n", i, reason, action)
				continue
			}

			requiredVersion := permittedActions[name]
			if requiredVersion == "" {
				fmt.Printf("%4d, Error. unknown action: %s\n", i, action)
				continue
			}
			if requiredVersion == currentVersion {
				fmt.Printf("%4d, OK. %s\n", i, action)
				continue
			}

			fmt.Printf("%4d, Replace. %s -> %s\n", i, action, requiredVersion)
			contents[i] = replaceActionTag(line, requiredVersion)
		}

		err = os.WriteFile(path, []byte(strings.Join(contents, "\n")+"\n"), os.ModePerm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "os.WriteFile: %v", err)
			return
		}
	}
}

func isYamlFile(name string) bool {
	return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
}
