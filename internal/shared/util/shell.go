package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"shared/proto"
	"slices"
	"sort"
	"strings"
)

type TerminalSize struct {
	Rows    int32
	Columns int32
}

func (s TerminalSize) ToProto() *proto.TerminalSize {
	return &proto.TerminalSize{
		Rows:    s.Rows,
		Columns: s.Columns,
	}
}

func TerminalSizeFromProto(size *proto.TerminalSize) TerminalSize {
	return TerminalSize{
		Rows:    size.Rows,
		Columns: size.Columns,
	}
}

func FindBestShell() (string, error) {
	shells, err := GetValidShells()
	if err != nil {
		return "", err
	}
	return shells[0], nil
}

func GetValidShells() ([]string, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsShells()
	default:
		return getUnixShells()
	}
}

func getUnixShells() ([]string, error) {

	// Modern -> classic priority
	priority := []string{"zsh", "fish", "bash", "dash", "ksh", "tcsh", "csh", "ash", "sh"}

	shells, err := readEtcShells()

	if err != nil || len(shells) == 0 {
		shells = findShells(priority)
	}

	if len(shells) == 0 {
		return nil, fmt.Errorf("no valid shells found on system")
	}

	// Sort by priority
	sort.SliceStable(shells, func(i, j int) bool {
		pi := indexOfShell(priority, shells[i])
		pj := indexOfShell(priority, shells[j])
		return pi < pj
	})

	return shells, nil
}

func readEtcShells() ([]string, error) {
	file, err := os.Open("/etc/shells")
	if err != nil {
		return nil, err
	}
	defer CloseFile(file)

	var shells []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "/") {
			shells = append(shells, line)
		}
	}

	return shells, scanner.Err()
}

func FindShell(requested string) (string, error) {

	shells, err := GetValidShells()

	if err != nil {
		return "", err
	}

	index := slices.IndexFunc(shells, func(s string) bool {

		if runtime.GOOS == "windows" {
			slc := strings.Split(s, "\\")
			if len(slc) == 0 {
				return false
			}
			basename := slc[len(slc)-1]
			basenameNoExt, _ := strings.CutSuffix(basename, ".exe")
			return s == requested || basename == requested || basenameNoExt == requested
		}

		slc := strings.Split(s, "/")
		return s == requested || slc[len(slc)-1] == requested
	})

	if index == -1 {
		return "", fmt.Errorf("invalid shell '%s'", requested)
	}

	return shells[index], nil

}

func getWindowsShells() ([]string, error) {
	priority := []string{
		"pwsh.exe",
		"powershell.exe",
		"cmd.exe",
		"git-bash.exe",
	}

	shells := findShells(priority)

	if len(shells) == 0 {
		return nil, fmt.Errorf("no valid shells found on system")
	}

	// Sort by priority
	sort.SliceStable(shells, func(i, j int) bool {
		pi := indexOfShell(priority, shells[i])
		pj := indexOfShell(priority, shells[j])
		return pi < pj
	})

	return shells, nil
}

func findShells(list []string) []string {
	var shells []string
	for _, name := range list {
		if path, err := exec.LookPath(name); err == nil {
			shells = append(shells, path)
		}
	}
	return shells
}

func indexOfShell(priority []string, path string) int {
	base := filepath.Base(path)
	for i, p := range priority {
		if base == p {
			return i
		}
	}
	return len(priority)
}
