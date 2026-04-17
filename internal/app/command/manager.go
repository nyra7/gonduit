package command

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
)

var alphaNumDashRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

type Manager struct {
	handlers        []Handler
	frozen          bool
	termCols        int
	termRows        int
	runningCommands atomic.Int32
}

func NewManager() *Manager {
	mgr := &Manager{}
	mgr.RegisterCommand(MakeHelpCommand())
	return mgr
}

// Exec executes a command without cancellation
func (m *Manager) Exec(input string) (string, error) {
	return m.ExecWithContext(input, context.Background())
}

func (m *Manager) ExecWithContext(input string, ctx context.Context) (string, error) {

	m.runningCommands.Add(1)
	defer m.runningCommands.Add(-1)

	// Check if context is already canceled before executing
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	cmdCtx, handler, err := NewContext(m, input, ctx)

	if err != nil {
		return "", err
	}

	// Check if help flag is set (already validated in NewContext)
	if cmdCtx.IsFlagSet("help") {
		cols, _ := m.TermSize()
		return formatDetailedHelp(handler, cols), nil
	}

	result, err := handler.Executor()(cmdCtx)

	if err != nil {
		return "", err
	}

	return result, nil

}

func (m *Manager) RegisterCommand(handler Handler) {

	if m.frozen {
		panic("cannot register command after manager is frozen")
	}

	if err := m.validateHandler(handler); err != nil {
		panic(fmt.Sprintf("could not register '%s' command: %v", handler.Name(), err))
	}

	m.handlers = append(m.handlers, handler)

}

func (m *Manager) Freeze() {
	m.frozen = true
}

func (m *Manager) Handlers() []Handler {
	return append([]Handler(nil), m.handlers...)
}

func (m *Manager) NumRunningCommands() int {
	return int(m.runningCommands.Load())
}

func (m *Manager) IsRunning() bool {
	return m.NumRunningCommands() > 0
}

func (m *Manager) validateHandler(handler Handler) error {

	if handler == nil {
		return NewError(ErrNilHandler, "handler cannot be nil")
	}

	if strings.ToLower(handler.Name()) != handler.Name() {
		return NewError(ErrInvalidHandlerName, "command names must be lowercase (got %s)", handler.Name())
	}

	if !alphaNumDashRegex.MatchString(handler.Name()) {
		return NewError(ErrInvalidHandlerName, "command names can only contain alphanumeric characters or dashes (got %s)", handler.Name())
	}

	for _, cmd := range m.handlers {
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

func (m *Manager) FindHandler(name string) (Handler, error) {
	for _, handler := range m.handlers {
		if handler.Name() == name {
			return handler, nil
		}
	}
	return nil, NewError(ErrUnknownHandler, "unknown command: %s. Type 'help' for available commands", name)
}

func (m *Manager) SetDimensions(cols, rows int) {
	m.termCols = cols
	m.termRows = rows
}

func (m *Manager) TermSize() (int, int) {
	return m.termCols, m.termRows
}
