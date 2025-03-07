package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/masa213f/tools/pkg/util"
	"golang.org/x/mod/modfile"
)

type Config struct {
	GoVersion string `json:"go-version"`
	LockRule  []Rule `json:"lock"`
}

type Rule struct {
	Name    string        `json:"name"`
	Modules []ModuleGroup `json:"modules"`
}

type ModuleGroup struct {
	Path []string `json:"path"`
	Tag  string   `json:"tag"`
}

//go:embed config.yaml
var defaultConfigBytes []byte

var configFilePath string

func init() {
	const usage = `Usage: update-gomod [<options>] [<work-dir>]

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
	workDir := flag.Arg(0)

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

	fmt.Printf("WORK_DIR: %s\n", workDir)

	modules, err := getDirectDependencies(filepath.Join(workDir, "go.mod"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get direct dependencies: %v\n", err)
		os.Exit(1)
	}

	grouped := grouping(&config, modules)
	err = update(workDir, config.GoVersion, grouped)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func getDirectDependencies(modFilePath string) ([]string, error) {
	data, err := os.ReadFile(modFilePath)
	if err != nil {
		return nil, err
	}
	file, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, err
	}

	modules := []string{}
	for _, r := range file.Require {
		if r.Indirect {
			continue
		}
		modules = append(modules, r.Mod.Path)
	}
	return modules, nil
}

func grouping(config *Config, modules []string) [][]string {
	type lockedTag struct {
		ruleName string
		tag      string
	}
	lockedModules := map[string]lockedTag{}
	for _, rule := range config.LockRule {
		for _, group := range rule.Modules {
			for _, mod := range group.Path {
				lockedModules[mod] = lockedTag{ruleName: rule.Name, tag: group.Tag}
			}
		}
	}

	grouped := [][]string{}
	locked := map[string][]string{}
	for _, mod := range modules {
		if l, ok := lockedModules[mod]; ok {
			locked[l.ruleName] = append(locked[l.ruleName], mod+"@"+l.tag)
			continue
		}
		grouped = append(grouped, []string{mod})
	}
	for ruleName, mods := range locked {
		fmt.Printf("LOCK RULE: %s %v\n", ruleName, mods)
		grouped = append(grouped, mods)
	}
	return grouped
}

func update(workDir string, goVersion string, groupedModules [][]string) error {
	cmd := exec.Command("go", "mod", "edit", "-go", goVersion)
	cmd.Dir = workDir
	err := util.Run(cmd)
	if err != nil {
		return err
	}

	for _, modules := range groupedModules {
		cmd := exec.Command("go", append([]string{"get"}, modules...)...)
		cmd.Dir = workDir
		err := util.Run(cmd)
		if err != nil {
			return err
		}
	}

	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = workDir
	return util.Run(cmd)
}
