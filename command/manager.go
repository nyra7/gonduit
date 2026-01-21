package command

import (
	"bufio"
	"errors"
	"fmt"
	"gonduit/pkg"
	"gonduit/style"
	"gonduit/util"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
)

var alphaNumDashRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

type Manager struct {
	handlers []Handler
	frozen   bool
}

func NewManager() *Manager {

	manager := &Manager{}

	manager.RegisterCommand(MakeHelpCommand(manager))
	manager.RegisterCommand(MakeExitCommand())
	manager.RegisterCommand(MakeShellCommand())
	manager.RegisterCommand(MakeExecHandler())

	return manager

}

func (cm *Manager) RegisterCommand(handler Handler) {

	if cm.frozen {
		log.Fatalf("cannot register command after manager is frozen")
	}

	if err := cm.validateHandler(handler); err != nil {
		log.Fatalf("could not register '%s' command: %v", handler.Name(), err)
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
	util.WriteConn(conn, fmt.Sprintf("server version %s %s\n", style.Italic.Apply(pkg.Version), url))
	util.WriteConn(conn, fmt.Sprintf("Running at %s on %s\n", style.Cyan.Apply(wd), style.Magenta.Apply(conn.LocalAddr().String())))
	util.WriteConn(conn, fmt.Sprintf("Type %s for a list of commands.\n\n", style.BoldYellow.Apply("'help'")))

	for {

		prompt := style.BoldWhite.Apply("gonduit")
		util.WriteConn(conn, fmt.Sprintf("%s> ", prompt))

		// Read command line
		input, readErr := reader.ReadString('\n')

		if readErr != nil {

			// If EOF, return no error (closes connection)
			if readErr == io.EOF {
				return nil
			}

			return fmt.Errorf("error reading command: %v", readErr)

		}

		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		args := strings.Split(input, " ")
		command := args[0]
		args = args[1:]

		handler, findErr := cm.FindHandler(command)

		if findErr != nil {
			util.WriteError(conn, findErr.Error()+"\n")
			continue
		}

		ctx, ctxErr := NewContext(input, handler, conn)

		if ctxErr != nil {
			util.WriteError(conn, ctxErr.Error()+"\n")
			util.WriteConn(conn, formatDetailedHelp(handler))
			continue
		}

		err = handler.Executor()(ctx)

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

	}

}

func (cm *Manager) validateHandler(handler Handler) error {

	if handler == nil {
		return NewError(ErrNilHandler, "handler cannot be nil")
	}

	if strings.ToLower(handler.Name()) != handler.Name() {
		return NewError(ErrInvalidHandlerName, "command names must be lowercase (got %s)", handler.Name())
	}

	if !alphaNumDashRegex.MatchString(handler.Name()) {
		return NewError(ErrInvalidHandlerName, "command names can only contain alphanumeric characters or dashes (got %s)", handler.Name())
	}

	for _, cmd := range cm.handlers {
		if cmd.Name() == handler.Name() {
			return NewError(ErrDuplicateHandler, "a command with name %s already exists", cmd.Name())
		}
	}

	seen := make(map[string]bool)
	variadic := false
	allArgs := append(handler.Args(), handler.Flags()...)

	for _, arg := range handler.Args() {

		if variadic {
			return fmt.Errorf("argument '%s' cannot follow variadic argument", arg.Name())
		}

		if arg.Type() == ArgTypeVariadic {
			variadic = true
		}

	}

	for _, flag := range handler.Flags() {
		if flag.Type() == ArgTypeVariadic {
			return fmt.Errorf("%s: flags cannot be variadic", flag.Name())
		}
	}

	for _, arg := range allArgs {

		if strings.ToLower(arg.Name()) != arg.Name() {
			return NewError(ErrInvalidHandlerName, "argument names must be lowercase (got %s)", arg.Name())
		}

		if !alphaNumDashRegex.MatchString(arg.Name()) {
			return NewError(ErrInvalidHandlerName, "argument names can only contain alphanumeric characters or dashes (got %s)", handler.Name())
		}

		if seen[arg.Name()] {
			return NewError(ErrDuplicateArgument, "argument %s is defined more than once", arg.Name())
		}

		seen[arg.Name()] = true

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
