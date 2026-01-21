package command

import (
	"errors"
	"fmt"
	"gonduit/style"
	"gonduit/util"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/creack/pty"
)

func MakeShellCommand() Handler {

	shells, _ := util.GetValidShells()
	desc := fmt.Sprintf("The path of the shell to execute (default: %s)", shells[0])

	args := []Argument{
		NewArgument("path", desc, ArgTypeString, false),
	}

	flags := []Argument{
		NewArgument("list", "List all valid shells", ArgTypeBool, false),
	}

	return NewHandler("shell", shell, "Spawns a new shell", args, flags)

}

func shell(ctx *Context) error {

	var err error

	shells, err := util.GetValidShells()

	if err != nil {
		return err
	}

	target := shells[0]

	list := ctx.BoolFlag("list")

	if list {
		util.WriteConn(ctx.Conn, fmt.Sprintf("Valid shells:\n%s\n", style.BoldWhite.Apply(strings.Join(shells, "\n"))))
		return nil
	}

	requested, err := ctx.Next()

	if err == nil {

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
			return NewError(ErrUnknownHandler, "invalid shell '%s'. Use --list to show available shells", requested)
		}

		target = shells[index]

	}

	util.WriteInfo(ctx.Conn, fmt.Sprintf("Spawning %s...\n", target))

	if runtime.GOOS == "windows" {
		return spawnWindows(ctx.Conn, target)
	}

	return spawnUnix(ctx.Conn, target)

}

func spawnWindows(conn net.Conn, shellPath string) error {

	cmd := exec.Command(shellPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("error creating stdin pipe: %v", err)
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %v", err)
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stderr pipe: %v", err)
	}
	defer stderr.Close()

	// Start the shell
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("error spawning shell: %v", err)
	}

	go func() {
		_, _ = io.Copy(conn, stdout)
	}()

	go func() {
		_, _ = io.Copy(conn, stderr)
	}()

	go func() {
		_, _ = io.Copy(stdin, conn)
	}()

	return waitForCommand(conn, cmd)

}

func spawnUnix(conn net.Conn, shell string) error {

	cmd := exec.Command(shell)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)

	if err != nil {
		log.Println("pty error:", err)
		return fmt.Errorf("error spawning shell: %v\n", err)
	}

	defer ptmx.Close()

	// Copy data both ways
	go func() {
		_, _ = io.Copy(ptmx, conn)
	}()
	_, _ = io.Copy(conn, ptmx)

	return waitForCommand(conn, cmd)

}

func waitForCommand(conn net.Conn, cmd *exec.Cmd) error {

	err := cmd.Wait()
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
		util.WriteSuccess(conn, "shell exited with code 0\n")
		return nil
	}

	return fmt.Errorf("shell exited with code %d", exitCode)

}
