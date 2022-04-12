package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

func main() {
	if len(os.Args) > 2 {
		fmt.Println("Usage: gomod-upgrade [WORK_DIR]")
		os.Exit(1)
	}
	if len(os.Args) == 2 {
		workDir, _ := filepath.Abs(os.Args[1])
		fmt.Printf("WORK_DIR: %s\n", workDir)
		os.Chdir(workDir)
	}

	data, err := os.ReadFile("go.mod")
	if err != nil {
		fmt.Printf("ioutil.ReadFile: %v", err)
		os.Exit(1)
	}
	file, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		fmt.Printf("modfile.Parse: %v", err)
		os.Exit(1)
	}

	for _, r := range file.Require {
		if r.Indirect {
			continue
		}
		if strings.HasPrefix(r.Mod.Path, "k8s.io/") {
			fmt.Printf("skip: %s\n", r.Mod.Path)
			continue
		}
		if strings.HasPrefix(r.Mod.Path, "sigs.k8s.io/") {
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
		fmt.Printf("Error: cmd: %v, out: %s, err: %v\n", cmd, out, err)
		return err
	}
	if len(out) != 0 {
		fmt.Println(string(out))
	}
	return nil
}
