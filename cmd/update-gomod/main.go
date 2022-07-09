package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/masa213f/tools/pkg/util"
	"golang.org/x/mod/modfile"
)

var (
	ctrlModule        = "sigs.k8s.io/controller-runtime"
	ctrlModuleVersion string
)

var (
	k8sModules = []string{
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
	}
	k8sModulesMap     map[string]bool
	k8sModulesVersion string
)

func init() {
	k8sModulesMap = map[string]bool{}
	for _, i := range k8sModules {
		k8sModulesMap[i] = true
	}

	flag.StringVar(&ctrlModuleVersion, "c", "", "controller-runtime module version")
	flag.StringVar(&k8sModulesVersion, "k", "", "Kubernetes modules version")
}

func main() {
	flag.Parse()
	if flag.NArg() > 1 {
		fmt.Println("Usage: update-gomod [WORK_DIR]")
		os.Exit(1)
	}
	workDir := flag.Arg(0)
	fmt.Printf("WORK_DIR: %s\n", workDir)

	err := subMain(workDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func subMain(workDir string) error {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("failed to read go.mod; %v", err)
	}
	file, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return fmt.Errorf("failed to parse go.mod; %v", err)
	}

	job := [][]string{}
	requiredK8sModules := []string{}
	for _, r := range file.Require {
		if r.Indirect {
			continue
		}

		if ctrlModule == r.Mod.Path {
			if len(ctrlModuleVersion) == 0 {
				fmt.Printf("SKIP: %s\n", r.Mod.Path)
				continue
			}
			requiredK8sModules = append(requiredK8sModules, r.Mod.Path+"@"+ctrlModuleVersion)
			continue
		}

		if k8sModulesMap[r.Mod.Path] {
			if len(k8sModulesVersion) == 0 {
				fmt.Printf("SKIP: %s\n", r.Mod.Path)
				continue
			}
			requiredK8sModules = append(requiredK8sModules, r.Mod.Path+"@"+k8sModulesVersion)
			continue
		}

		job = append(job, []string{r.Mod.Path})
	}
	if len(requiredK8sModules) != 0 {
		job = append(job, requiredK8sModules)
	}

	for _, modules := range job {
		cmd := exec.Command("go", append([]string{"get", "-d"}, modules...)...)
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
