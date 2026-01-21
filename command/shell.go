package command

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"shells/style"
	"shells/util"
	"slices"
	"strings"
	"time"

	"github.com/creack/pty"
)

func MakeShellCommand() Handler {

	shells, _ := util.GetValidShells()
	desc := fmt.Sprintf("The path of the shell to execute (default: %s)", shells[0])

	args := []Argument{
		NewArgument("path", desc, true, false),
		NewArgument("list", "List all valid shells", false, false),
	}

	return NewHandler("shell", shell, "Spawns a new shell", args)

}

func shell(conn net.Conn, args []string) (string, error) {

	if len(args) > 2 {
		return "", NewError(ErrBadUsage, "bad usage: shell %s", strings.Join(args, " "))
	}

	var err error

	shells, err := util.GetValidShells()

	if err != nil {
		return "", err
	}

	target := shells[0]

	listIndex := slices.Index(args, "--list")

	if listIndex != -1 {
		util.WriteConn(conn, fmt.Sprintf("Valid shells:\n%s", style.BoldWhite.Apply(strings.Join(shells, "\n"))))
		return "", nil
	}

	if len(args) == 1 {

		index := slices.IndexFunc(shells, func(s string) bool {

			if runtime.GOOS == "windows" {
				slc := strings.Split(s, "\\")
				if len(slc) == 0 {
					return false
				}
				basename := slc[len(slc)-1]
				basenameNoExt, _ := strings.CutSuffix(basename, ".exe")
				return s == args[0] || basename == args[0] || basenameNoExt == args[0]
			}

			slc := strings.Split(s, "/")
			return s == args[0] || slc[len(slc)-1] == args[0]
		})

		if index == -1 {
			return "", NewError(ErrUnknownHandler, "invalid shell '%s'", args[0])
		}

		target = shells[index]

	}

	util.WriteInfo(conn, fmt.Sprintf("Spawning %s...\n", target))

	if runtime.GOOS == "windows" {
		return spawnWindows(conn, target)
	}

	return spawnUnix(conn, target)

}

// TODO: combine common functionality with spawnUnix

func spawnWindows(conn net.Conn, shellPath string) (string, error) {
	cmd := exec.Command(shellPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("error creating stdin pipe: %v", err)
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("error creating stdout pipe: %v", err)
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("error creating stderr pipe: %v", err)
	}
	defer stderr.Close()

	// Start the shell
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("error spawning shell: %v", err)
	}

	go func() {
		_, _ = io.Copy(conn, stdout)
	}()

	go func() {
		_, _ = io.Copy(conn, stderr)
	}()

	// Handle connection input -> stdin
	go func() {
		defer stdin.Close()
		_, _ = io.Copy(stdin, conn)
	}()

	err = cmd.Wait()
	exitCode := 0

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}

	_ = conn.SetReadDeadline(time.Now())
	_, _ = io.Copy(io.Discard, conn)
	_ = conn.SetReadDeadline(time.Time{})

	if exitCode == 0 {
		return fmt.Sprintf("shell exited with code %d", exitCode), nil
	}

	return "", fmt.Errorf("shell exited with code %d", exitCode)

}

func spawnUnix(conn net.Conn, shell string) (string, error) {

	cmd := exec.Command(shell, "-i")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TERM=vt100")

	ptmx, err := pty.Start(cmd)

	if err != nil {
		log.Println("pty error:", err)
		return "", fmt.Errorf("error spawning shell: %v\n", err)
	}

	defer ptmx.Close()

	// Copy data both ways
	go func() {
		_, _ = io.Copy(ptmx, conn)
	}()
	_, _ = io.Copy(conn, ptmx)

	err = cmd.Wait()

	exitCode := 0

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}

	_ = conn.SetReadDeadline(time.Now())
	_, _ = io.Copy(io.Discard, conn)
	_ = conn.SetReadDeadline(time.Time{})

	if exitCode == 0 {
		return fmt.Sprintf("shell exited with code %d", exitCode), nil
	}

	return "", fmt.Errorf("shell exited with code %d", exitCode)

}
