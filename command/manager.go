package command

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"shells/pkg"
	"shells/style"
	"shells/util"
	"slices"
	"strings"
)

type Manager struct {
	handlers []Handler
	frozen   bool
}

func NewManager() *Manager {

	manager := &Manager{}
	helpHandler := NewHandler("help", manager.help, "Show this help message", []Argument{})
	exitHandler := NewHandler("exit", exit, "Exit gonduit", []Argument{})
	shellHandler := MakeShellCommand()

	manager.RegisterCommand(helpHandler)
	manager.RegisterCommand(exitHandler)
	manager.RegisterCommand(shellHandler)

	return manager

}

func (cm *Manager) RegisterCommand(handler Handler) {

	if cm.frozen {
		panic("cannot register command after manager is frozen")
	}

	if err := cm.validateHandler(handler); err != nil {
		panic(fmt.Sprintf("could not register '%s' command: %v", handler.Name(), err))
	}

	cm.handlers = append(cm.handlers, handler)

}

func (cm *Manager) Freeze() {
	cm.frozen = true
}

func (cm *Manager) HandleConnection(conn net.Conn) error {

	defer conn.Close()
	reader := bufio.NewReader(conn)

	wd, err := os.Getwd()
	if err != nil {
		wd = "<unknown>"
	}

	url := style.Gray.Apply("(https://github.com/nyra7/gonduit)")

	// Welcome message with styling
	util.WriteConn(conn, fmt.Sprintf("\nWelcome to %s ", style.BoldWhite.Apply("gonduit")))
	util.WriteConn(conn, fmt.Sprintf("server version %s %s\n", style.Italic.Apply(pkg.Version+"+"+pkg.Commit), url))
	util.WriteConn(conn, fmt.Sprintf("Running at %s on %s\n", style.Cyan.Apply(wd), style.Magenta.Apply(conn.LocalAddr().String())))
	util.WriteConn(conn, fmt.Sprintf("Type %s for a list of commands.\n\n", style.BoldYellow.Apply("'help'")))

	for {

		prompt := style.BoldWhite.Apply("gonduit")
		util.WriteConn(conn, fmt.Sprintf("%s> ", prompt))

		// Read command line
		input, err := reader.ReadString('\n')

		if err != nil {

			// If EOF, return no error (closes connection)
			if err == io.EOF {
				return nil
			}

			return fmt.Errorf("error reading command: %v", err)

		}

		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		args := strings.Split(input, " ")
		command := args[0]
		args = args[1:]

		handler, err := cm.FindHandler(command)

		if err != nil {
			util.WriteError(conn, err.Error()+"\n")
			continue
		}

		res, err := handler.Executor()(conn, args)

		if err != nil {

			if err == io.EOF {
				util.WriteConn(conn, "Goodbye!\n")
				return nil
			}

			util.WriteError(conn, err.Error()+"\n")

			var e Error

			if errors.As(err, &e); e.code == ErrBadUsage {
				util.WriteConn(conn, formatDetailedHelp(handler))
			}

			continue

		}

		util.WriteSuccess(conn, res+"\n")

	}

}

func (cm *Manager) validateHandler(handler Handler) error {

	if handler == nil {
		return NewError(ErrNilHandler, "handler cannot be nil")
	}

	if strings.ToLower(handler.Name()) != handler.Name() {
		return NewError(ErrNotLowercaseName, "command names must be lowercase (got %s)", handler.Name())
	}

	for _, cmd := range cm.handlers {
		if cmd.Name() == handler.Name() {
			return NewError(ErrDuplicateHandler, "a handler with name %s already exists", cmd.Name())
		}
	}

	var seen []string

	for _, arg := range handler.Args() {

		if strings.ToLower(arg.Name()) != arg.Name() {
			return NewError(ErrNotLowercaseName, "argument names must be lowercase (got %s)", arg.Name())
		}

		if slices.Index(seen, arg.Name()) != -1 {
			return NewError(ErrDuplicateArgument, "argument %s is defined more than once", arg.Name())
		}

	}

	return nil

}

func (cm *Manager) FindHandler(name string) (Handler, error) {
	for _, handler := range cm.handlers {
		if handler.Name() == name {
			return handler, nil
		}
	}
	return nil, NewError(ErrUnknownHandler, "unknown command: %s. Type 'help' for available commands", name)
}
