package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/modfile"
)

const (
	ctrlModule = "sigs.k8s.io/controller-runtime"
)

var k8sModules = []string{
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

func main() {
	flag.Parse()
	if flag.NArg() > 1 {
		fmt.Println("Usage: gomod-upgrade [WORK_DIR]")
		os.Exit(1)
	}
	if workDir := flag.Arg(0); workDir != "" {
		os.Chdir(workDir)
		path, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "os.Getwd: %v", err)
			os.Exit(1)
		}
		fmt.Printf("WORK_DIR: %s\n", path)
	}

	data, err := os.ReadFile("go.mod")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ioutil.ReadFile: %v", err)
		os.Exit(1)
	}
	file, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "modfile.Parse: %v", err)
		os.Exit(1)
	}

	ignoreMap := map[string]bool{}
	for _, i := range k8sModules {
		ignoreMap[i] = true
	}
	ignoreMap[ctrlModule] = true

	for _, r := range file.Require {
		if r.Indirect {
			continue
		}
		if ignoreMap[r.Mod.Path] {
			fmt.Printf("skip: %s\n", r.Mod.Path)
			continue
		}

		execCmd("go", "get", "-d", r.Mod.Path)
	}

	execCmd("go", "mod", "tidy")
}

func execCmd(cmd ...string) error {
	fmt.Println("run: " + strings.Join(cmd, " "))
	out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cmd: %v, out: %s, err: %v\n", cmd, out, err)
		return err
	}
	if len(out) != 0 {
		fmt.Println(string(out))
	}
	return nil
}
