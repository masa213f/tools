package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/masa213f/tools/pkg/util"
	"golang.org/x/mod/modfile"
)

type LockRule struct {
	Name  string
	Group []PackageGroup
}

type PackageGroup struct {
	Path []string
	Tag  string
}

var defaultRules = []LockRule{
	{
		Name: "k8s",
		Group: []PackageGroup{
			{
				Path: []string{
					"sigs.k8s.io/controller-runtime",
				},
				Tag: "v0.12.3",
			},
			{
				Path: []string{
					"k8s.io/api",
					"k8s.io/apiextensions-apiserver",
					"k8s.io/apimachinery",
					"k8s.io/apiserver",
					"k8s.io/client-go",
					"k8s.io/cli-runtime",
					"k8s.io/kubectl",
					"k8s.io/kubelet",
					"k8s.io/kube-proxy",
					"k8s.io/kube-scheduler",
				},
				Tag: "v0.24.2",
			},
		},
	},
	{
		Name: "cni",
		Group: []PackageGroup{
			{
				Path: []string{
					"github.com/containernetworking/cni",
					"github.com/containernetworking/plugins",
				},
				Tag: "v1.0.1",
			},
		},
	},
}

func main() {
	flag.Parse()
	if flag.NArg() > 1 {
		fmt.Println("Usage: update-gomod [WORK_DIR]")
		os.Exit(1)
	}
	workDir := flag.Arg(0)
	fmt.Printf("WORK_DIR: %s\n", workDir)

	err := subMain(workDir, defaultRules)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func subMain(workDir string, rules []LockRule) error {
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
	for _, rule := range defaultRules {
		for _, group := range rule.Group {
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
		fmt.Printf("Rule %s: %v\n", ruleName, packages)
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
