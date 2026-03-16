//go:build windows

package pty

import (
	"context"
	"errors"
	"fmt"
	"os"
	"shared/util"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type WinPty struct {
	consoleHandle     windows.Handle
	consoleRead       *os.File
	consoleWrite      *os.File
	startupInfo       *windows.StartupInfoEx
	processInfo       *windows.ProcessInformation
	attrListContainer *windows.ProcThreadAttributeListContainer
}

func (w *WinPty) Start(_ context.Context, cmd string, args ...string) error {

	// Set up pseudoconsole with actual terminal size
	hpc, conIn, conOut, inputRead, outputWrite, err := setupPty()

	if err != nil {
		return fmt.Errorf("failed to setup pseudo console: %v", err)
	}

	// Prepare startup information
	si, err := w.initStartupInfo(hpc)
	if err != nil {
		return fmt.Errorf("failed to prepare startup info: %v", err)
	}

	pi, err := w.createProcess(si, fmt.Sprintf("%s %s", cmd, strings.Join(args, " ")))
	if err != nil {
		return fmt.Errorf("failed to create process: %v", err)
	}

	// Cleanup the handles that were given to the pseudoconsole
	_ = windows.CloseHandle(inputRead)
	_ = windows.CloseHandle(outputWrite)

	w.startupInfo = si
	w.consoleHandle = hpc
	w.consoleRead = conOut
	w.consoleWrite = conIn
	w.processInfo = pi

	return nil

}

func (w *WinPty) Close() error {

	if w.consoleHandle != 0 {
		windows.ClosePseudoConsole(w.consoleHandle)
		w.consoleHandle = 0
	}

	if w.attrListContainer != nil {
		w.attrListContainer.Delete()
		w.startupInfo = nil
		w.attrListContainer = nil
	}

	if w.processInfo != nil {
		_ = windows.CloseHandle(w.processInfo.Process)
		_ = windows.CloseHandle(w.processInfo.Thread)
		w.processInfo = nil
	}

	return nil
}

func (w *WinPty) Resize(size util.TerminalSize) error {
	return windows.ResizePseudoConsole(w.consoleHandle, windows.Coord{X: int16(size.Columns), Y: int16(size.Rows)})
}

func (w *WinPty) Read(b []byte) (n int, err error) {
	return w.consoleRead.Read(b)
}

func (w *WinPty) Write(b []byte) (n int, err error) {
	return w.consoleWrite.Write(b)
}

func (w *WinPty) Wait() error {

	var exitCode uint32

	if _, err := windows.WaitForSingleObject(w.processInfo.Process, syscall.INFINITE); err != nil {
		return err
	}

	if err := windows.GetExitCodeProcess(w.processInfo.Process, &exitCode); err != nil {
		return err
	}

	return &ExitError{ExitCode: int(exitCode)}

}

func (w *WinPty) SetEcho(mode bool) error {
	return errors.New("unimplemented")
}

func (w *WinPty) initStartupInfo(hpc windows.Handle) (*windows.StartupInfoEx, error) {

	si := &windows.StartupInfoEx{}
	si.StartupInfo.Cb = uint32(unsafe.Sizeof(*si))

	container, err := windows.NewProcThreadAttributeList(1)

	if err != nil {
		return nil, fmt.Errorf("failed to create proc thread attribute list: %w", err)
	}

	err = container.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, unsafe.Pointer(hpc), unsafe.Sizeof(hpc))

	if err != nil {
		container.Delete()
		return nil, fmt.Errorf("failed to update proc thread attribute list: %w", err)
	}

	si.ProcThreadAttributeList = container.List()
	w.attrListContainer = container

	return si, nil

}

func (w *WinPty) createProcess(si *windows.StartupInfoEx, commandLine string) (*windows.ProcessInformation, error) {
	pi := &windows.ProcessInformation{}

	// Convert command line to UTF16
	cmdLine, err := windows.UTF16PtrFromString(commandLine)
	if err != nil {
		return nil, fmt.Errorf("failed to convert command line: %w", err)
	}

	// Create process
	err = windows.CreateProcess(
		nil,
		cmdLine,
		nil,
		nil,
		false,
		windows.EXTENDED_STARTUPINFO_PRESENT,
		nil,
		nil,
		&si.StartupInfo,
		pi,
	)

	if err != nil {
		return nil, fmt.Errorf("CreateProcess failed: %w", err)
	}

	return pi, nil
}

// CreatePseudoConsole wraps the Windows CreatePseudoConsole API
func createPty(size windows.Coord, hInput, hOutput windows.Handle) (windows.Handle, error) {
	var hpc windows.Handle

	if err := windows.CreatePseudoConsole(size, hInput, hOutput, 0, &hpc); err != nil {
		return 0, err
	}

	return hpc, nil
}

// SetUpPseudoConsole creates the pipes and pseudoconsole
func setupPty() (windows.Handle, *os.File, *os.File, windows.Handle, windows.Handle, error) {

	var inputRead, inputWrite windows.Handle
	var outputRead, outputWrite windows.Handle

	// Create input pipe
	err := windows.CreatePipe(&inputRead, &inputWrite, nil, 0)
	if err != nil {
		return 0, nil, nil, 0, 0, fmt.Errorf("failed to create input pipe: %w", err)
	}

	// Create output pipe
	err = windows.CreatePipe(&outputRead, &outputWrite, nil, 0)
	if err != nil {
		_ = windows.CloseHandle(inputRead)
		_ = windows.CloseHandle(inputWrite)
		return 0, nil, nil, 0, 0, fmt.Errorf("failed to create output pipe: %w", err)
	}

	// Create pseudoconsole
	size := windows.Coord{X: 80, Y: 16}
	hpc, err := createPty(size, inputRead, outputWrite)
	if err != nil {
		_ = windows.CloseHandle(inputRead)
		_ = windows.CloseHandle(inputWrite)
		_ = windows.CloseHandle(outputRead)
		_ = windows.CloseHandle(outputWrite)
		return 0, nil, nil, 0, 0, fmt.Errorf("failed to create pseudoconsole: %w", err)
	}

	// Convert handles to *os.File for easier I/O
	conIn := os.NewFile(uintptr(inputWrite), "conin")
	conOut := os.NewFile(uintptr(outputRead), "conout")

	return hpc, conIn, conOut, inputRead, outputWrite, nil

}

func NewPty() Pty {
	return &WinPty{}
}
