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
	"actions/setup-go":                   "v3",
	"actions/setup-python":               "v4",
	"actions/upload-artifact":            "v3",
	"azure/setup-helm":                   "v3",
	"google-github-actions/auth":         "v0.8.0",
	"google-github-actions/setup-gcloud": "v0.6.0",
	"goreleaser/goreleaser-action":       "68acf3b1adf004ac9c2f0a4259e85c5f66e99bef", // v3.0.0
	"helm/chart-testing-action":          "v2.2.1",
	"helm/kind-action":                   "v1.3.0",
	"rajatjindal/krew-release-bot":       "92da038bbf995803124a8e50ebd438b2f37bbbb0", // 0.0.43
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
