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
	LockRule []Rule `json:"lock"`
}

type Rule struct {
	Name     string         `json:"name"`
	Packages []PackageGroup `json:"packages"`
}

type PackageGroup struct {
	Path []string `json:"path"`
	Tag  string   `json:"tag"`
}

//go:embed config.yaml
var defaultConfigBytes []byte

func main() {
	flag.Parse()
	if flag.NArg() > 1 {
		fmt.Println("Usage: update-gomod [WORK_DIR]")
		os.Exit(1)
	}
	workDir := flag.Arg(0)
	fmt.Printf("WORK_DIR: %s\n", workDir)

	var deufaltConfig Config
	err := yaml.Unmarshal(defaultConfigBytes, &deufaltConfig)
	if err != nil {
		os.Exit(1)
	}

	err = subMain(workDir, &deufaltConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func subMain(workDir string, config *Config) error {
	data, err := os.ReadFile(filepath.Join(workDir, "go.mod"))
	if err != nil {
		return fmt.Errorf("failed to read go.mod; %v", err)
	}
	file, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return fmt.Errorf("failed to parse go.mod; %v", err)
	}

	packageRuleName := map[string]string{}
	packageTag := map[string]string{}
	for _, rule := range config.LockRule {
		for _, group := range rule.Packages {
			for _, pkg := range group.Path {
				packageRuleName[pkg] = rule.Name
				packageTag[pkg] = group.Tag
			}
		}
	}

	jobs := [][]string{}
	locked := map[string][]string{}
	for _, r := range file.Require {
		if r.Indirect {
			continue
		}

		if rule := packageRuleName[r.Mod.Path]; rule != "" {
			tag := packageTag[r.Mod.Path]
			locked[rule] = append(locked[rule], r.Mod.Path+"@"+tag)
			continue
		}

		jobs = append(jobs, []string{r.Mod.Path})
	}
	for ruleName, packages := range locked {
		fmt.Printf("LOCK RULE: %s %v\n", ruleName, packages)
		jobs = append(jobs, packages)
	}

	for _, packages := range jobs {
		cmd := exec.Command("go", append([]string{"get", "-d"}, packages...)...)
		cmd.Dir = workDir
		err := util.Run(cmd)
		if err != nil {
			return err
		}
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = workDir
	return util.Run(cmd)
}
