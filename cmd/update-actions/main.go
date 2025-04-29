package main

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Allow []AllowedAction `json:"allow"`
	Deny  []DeniedAction  `json:"deny"`
}

type AllowedAction struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
	Hash string `json:"hash"`
}

type DeniedAction struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

//go:embed config.yaml
var defaultConfigBytes []byte

var configFilePath string

var (
	findActionRe    = regexp.MustCompile(`.*\suses:\s*(\S+)\s*`)
	replaceActionRe = regexp.MustCompile(`(.*\suses:).*`)
)

func findAction(line string) string {
	matches := findActionRe.FindStringSubmatch(line)
	if len(matches) == 0 {
		return ""
	}
	// fmt.Printf("DEBUG[0]: **%s**\n", matches[0])
	// fmt.Printf("DEBUG[1]: **%s**\n", matches[1])
	return matches[1]
}

func replaceActionTag(line, name, tag string) string {
	repl := fmt.Sprintf("${1} %s@%s", name, tag) // uses: {action}@{tag}
	return replaceActionRe.ReplaceAllString(line, repl)
}

func replaceActionHash(line, name, hash, tag string) string {
	repl := fmt.Sprintf("${1} %s@%s # %s", name, hash, tag) // uses: {action}@{hash} # {tag}
	return replaceActionRe.ReplaceAllString(line, repl)
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

func isYamlFile(name string) bool {
	return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
}

func getWorkflowFiles(dir string) ([]string, error) {
	files := []string{}
	rootDir := filepath.Join(dir, ".github")
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
		files = append(files, abs)
		return nil
	})
	return files, err
}

func init() {
	const usage = `Usage: update-actions [<options>] [<target-dir>]

Options:
  -c <config>   path of config file
  -h            display this help and exit
`
	flag.Usage = func() { fmt.Fprintf(flag.CommandLine.Output(), usage) }
	flag.StringVar(&configFilePath, "c", "", "")
}

func main() {
	flag.Parse()
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(1)
	}
	targetDir := "."
	if flag.NArg() == 1 {
		targetDir = flag.Arg(0)
	}

	rawConfig := defaultConfigBytes
	if configFilePath != "" {
		dat, err := os.ReadFile(configFilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to read config file: %v\n", err)
			os.Exit(1)
		}
		rawConfig = dat
	}

	var config Config
	err := yaml.Unmarshal(rawConfig, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to unmarshal config yaml: %v\n", err)
		os.Exit(1)
	}

	allowedActions := map[string]*AllowedAction{}
	deniedActions := map[string]*DeniedAction{}
	for i := range config.Allow {
		allowedActions[config.Allow[i].Name] = &config.Allow[i]
	}
	for i := range config.Deny {
		deniedActions[config.Deny[i].Name] = &config.Deny[i]
	}

	workflowFiles, err := getWorkflowFiles(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "filepath.Walk: %v", err)
		os.Exit(1)
	}

	gotError := false
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

			if strings.HasPrefix(action, "./") {
				fmt.Printf("%4d: Skip. local action: %s\n", i, action)
				continue
			}

			// This tool is for the public actions only.
			// Public Actions are in the following format.
			// - `{owner}/{repo}@{ref}`
			// - `{owner}/{repo}/{path}@{ref}`
			// ref: https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#jobsjob_idstepsuses
			split := strings.SplitN(action, "@", 2)
			if len(split) != 2 {
				fmt.Printf("%4d: Error. unknown format: %s\n", i, action)
				gotError = true
				continue
			}
			name := split[0]
			currentVersion := split[1]

			if denied, ok := deniedActions[name]; ok {
				fmt.Printf("%4d: Error. denied (%s): %s\n", i, denied.Reason, action)
				gotError = true
				continue
			}

			allowed, ok := allowedActions[name]
			if !ok {
				fmt.Printf("%4d: Error. unknown action: %s\n", i, action)
				gotError = true
				continue
			}
			if allowed.Hash != "" {
				if allowed.Hash == currentVersion {
					fmt.Printf("%4d: OK. %s\n", i, action)
					continue
				}
				fmt.Printf("%4d: Replace. %s -> %s\n", i, action, allowed.Hash)
				contents[i] = replaceActionHash(line, name, allowed.Hash, allowed.Tag)
			} else {
				if allowed.Tag == currentVersion {
					fmt.Printf("%4d: OK. %s\n", i, action)
					continue
				}
				fmt.Printf("%4d: Replace. %s -> %s\n", i, action, allowed.Tag)
				contents[i] = replaceActionTag(line, name, allowed.Tag)
			}
		}

		err = os.WriteFile(path, []byte(strings.Join(contents, "\n")+"\n"), os.ModePerm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "os.WriteFile: %v", err)
			os.Exit(1)
		}
	}

	if gotError {
		os.Exit(1)
	}
}
