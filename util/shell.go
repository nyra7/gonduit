package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

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
		return getWindowsShells(), nil
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
	defer file.Close()

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

func getWindowsShells() []string {
	candidates := []string{
		"pwsh.exe",
		"powershell.exe",
		"cmd.exe",
		"bash.exe",
		"zsh.exe",
		"sh.exe",
	}

	var shells []string
	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			shells = append(shells, path)
		}
	}

	return shells
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
