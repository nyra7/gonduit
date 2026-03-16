package command

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"shared/pkg"
	"shared/util"
	"strconv"
	"strings"
)

type Context struct {
	input   string
	args    []any
	flags   map[string]any
	mgr     *Manager
	handler Handler
	ctx     context.Context
}

// ContextError represents errors during context creation
type ContextError int

const (
	ErrMissingRequiredArg ContextError = iota
	ErrMissingRequiredFlag
	ErrInvalidArgType
	ErrInvalidFlagType
	ErrTooManyArgs
	ErrUnknownFlag
	ErrMissingFlagValue
)

func (e ContextError) Error() string {
	switch {
	case errors.Is(e, ErrMissingRequiredArg):
		return "missing required argument"
	case errors.Is(e, ErrMissingRequiredFlag):
		return "missing required flag"
	case errors.Is(e, ErrInvalidArgType):
		return "invalid argument type"
	case errors.Is(e, ErrInvalidFlagType):
		return "invalid flag type"
	case errors.Is(e, ErrTooManyArgs):
		return "too many arguments provided"
	case errors.Is(e, ErrUnknownFlag):
		return "unknown flag"
	case errors.Is(e, ErrMissingFlagValue):
		return "missing flag value"
	default:
		return "unknown context error"
	}
}

func NewContextError(code ContextError, message string) error {
	return fmt.Errorf("%w: %s", code, message)
}

func NewContext(m *Manager, input string, ctx context.Context) (*Context, Handler, error) {

	parts, err := util.QuotedSplit(input)

	if err != nil {
		return nil, nil, err
	}

	if len(parts) == 0 {
		return nil, nil, fmt.Errorf("empty command")
	}

	command := parts[0]
	parts = parts[1:]

	handler, findErr := m.FindHandler(command)

	if findErr != nil {
		return nil, nil, findErr
	}

	// Check if help flag is present (fast path, skip validation)
	if hasHelpFlag(parts, handler) {

		// Create minimal context just for help display
		return &Context{
			input:   input,
			args:    []any{},
			flags:   map[string]any{"help": true},
			mgr:     m,
			handler: handler,
			ctx:     ctx,
		}, handler, nil

	}

	// Parse flags and arguments
	flags, args, err := parse(parts, handler)
	if err != nil {
		return nil, nil, err
	}

	// Validate against cmd
	if err = validate(args, flags, handler); err != nil {
		return nil, nil, err
	}

	return &Context{
		input:   input,
		args:    args,
		flags:   flags,
		mgr:     m,
		handler: handler,
		ctx:     ctx,
	}, handler, nil

}

// hasHelpFlag quickly checks if --help or -h is present without full validation
func hasHelpFlag(parts []string, handler Handler) bool {

	for _, part := range parts {
		if part == "--help" {
			return true
		}
		// Check for -h shorthand
		if part == "-h" {
			if ch, ok := handler.(*commandHandler); ok {
				// Check if 'h' is a valid shorthand for 'help'
				if fullName, exists := ch.shorthands["h"]; exists && fullName == "help" {
					return true
				}
			}
			// Also check if there's a single-char 'h' flag that maps to help
			if flag, err := handler.GetFlag("h"); err == nil && flag.Name() == "h" {
				// Single char flag 'h' exists, check if it's defined as help
				return true
			}
		}
	}

	return false

}

// parse extracts flags and positional arguments from parts
func parse(parts []string, cmd Handler) (map[string]any, []any, error) {

	flags := make(map[string]any)
	args := make([]any, 0, len(parts))

	for i := 0; i < len(parts); i++ {

		part := parts[i]

		// Check if this is a variadic argument
		if isVariadicArgument(len(args), cmd) {

			// Consume all remaining parts as variadic
			variadicValues := parts[i:]
			args = append(args, variadicValues)
			break

		}

		// Try to read next part as a flag
		flagName, err := readFlag(part, cmd)

		if err != nil {
			return nil, nil, err
		}

		if flagName != "" {

			// Check if flag is known
			if err = checkFlagExists(flagName, part, cmd); err != nil {
				return nil, nil, err
			}

			// Parse flag value
			flagValue, consumed, parseErr := parseFlagValue(flagName, parts[i+1:], cmd)
			if parseErr != nil {
				return nil, nil, parseErr
			}

			flags[flagName] = flagValue
			i += consumed

			continue

		}

		// Check if too many arguments
		if err = checkArgumentCount(len(args), cmd); err != nil {
			return nil, nil, err
		}

		// Parse positional argument
		argValue := parseValue(part, getArgumentType(len(args), cmd))
		args = append(args, argValue)

	}

	return flags, args, nil
}

// validate checks that all requirements are met
func validate(args []any, flags map[string]any, cmd Handler) error {

	if cmd == nil {
		return nil
	}

	if err := validateArguments(args, cmd); err != nil {
		return err
	}

	if err := validateFlags(flags, cmd); err != nil {
		return err
	}

	if err := validateHandlerArgs(cmd.Args()); err != nil {
		return err
	}

	return nil
}

// checkFlagExists verifies that a flag is defined
func checkFlagExists(flagName string, part string, cmd Handler) error {

	if cmd == nil {
		return nil
	}

	if _, err := cmd.GetFlag(flagName); err != nil {
		return NewContextError(ErrUnknownFlag, part)
	}

	return nil

}

// checkArgumentCount verifies we haven't exceeded max arguments
func checkArgumentCount(currentCount int, cmd Handler) error {
	if cmd == nil || len(cmd.Args()) == 0 {
		return nil
	}

	if currentCount >= cmd.NumArgs() {
		return NewContextError(ErrTooManyArgs, fmt.Sprintf("expected %d, got at least %d", cmd.NumArgs(), currentCount+1))
	}

	return nil
}

// parseFlagValue extracts and parses a flag value from remaining parts
func parseFlagValue(flagName string, remaining []string, cmd Handler) (any, int, error) {
	flagType := getFlagType(flagName, cmd)

	// Boolean flags don't consume a value
	if flagType == ArgTypeBool {
		return true, 0, nil
	}

	// Non-boolean flags need a value
	if len(remaining) == 0 || strings.HasPrefix(remaining[0], "--") || (strings.HasPrefix(remaining[0], "-") && len(remaining[0]) > 1) {
		return nil, 0, NewContextError(ErrMissingFlagValue, flagName)
	}

	value := parseValue(remaining[0], flagType)

	// Validate type if cmd exists
	if cmd != nil && !validateType(value, flagType) {
		return nil, 0, NewContextError(ErrInvalidFlagType, fmt.Sprintf("%s: expected %s, got %T", flagName, flagType, value))
	}

	return value, 1, nil
}

// parseValue converts a string to the appropriate type
func parseValue(s string, argType ArgumentType) any {

	switch argType {
	case ArgTypeInt:
		if i, err := strconv.Atoi(s); err == nil {
			return i
		}
	case ArgTypeFloat:
		if f, err := strconv.ParseFloat(s, 32); err == nil {
			return float32(f)
		}
	case ArgTypeBool:
		if b, err := strconv.ParseBool(s); err == nil {
			return b
		}
	case ArgTypeString, ArgTypeFile, ArgTypeDirectory:
		return s
	}

	// Auto-detect type when no cmd
	if argType == "" {
		if i, err := strconv.Atoi(s); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(s, 32); err == nil {
			return float32(f)
		}
		if b, err := strconv.ParseBool(s); err == nil {
			return b
		}
	}

	return s
}

// validateArguments checks argument requirements and types
func validateArguments(args []any, cmd Handler) error {
	if cmd == nil || len(cmd.Args()) == 0 {
		return nil
	}

	for idx, argDef := range cmd.Args() {
		if idx >= len(args) {
			if argDef.Required() {
				return NewContextError(ErrMissingRequiredArg, argDef.Name())
			}
		} else {
			// For variadic arguments, validate as []string
			if argDef.Type() == ArgTypeVariadic {
				if _, ok := args[idx].([]string); !ok {
					return NewContextError(ErrInvalidArgType, fmt.Sprintf("%s: expected []string for variadic, got %T", argDef.Name(), args[idx]))
				}
			} else if !validateType(args[idx], argDef.Type()) {
				return NewContextError(ErrInvalidArgType, fmt.Sprintf("%s: expected %s, got %T", argDef.Name(), argDef.Type(), args[idx]))
			}
		}
	}

	return nil
}

// validateFlags checks flag requirements
func validateFlags(flags map[string]any, cmd Handler) error {

	if cmd == nil {
		return nil
	}

	for _, flagDef := range cmd.Flags() {
		if flagDef.Required() {
			if _, exists := flags[flagDef.Name()]; !exists {
				return NewContextError(ErrMissingRequiredFlag, flagDef.Name())
			}
		}
	}

	return nil

}

// validateHandlerArgs checks that variadic arguments are only at the end
func validateHandlerArgs(args []Argument) error {
	variadicFound := false

	for _, arg := range args {
		if variadicFound && arg.Type() != ArgTypeVariadic {
			return fmt.Errorf("non-variadic argument %s cannot follow variadic argument", arg.Name())
		}
		if arg.Type() == ArgTypeVariadic {
			variadicFound = true
		}
	}

	return nil
}

// validateType checks if a value matches the expected type
func validateType(value any, expected ArgumentType) bool {
	switch expected {
	case ArgTypeInt:
		_, ok := value.(int)
		return ok
	case ArgTypeFloat:
		_, ok := value.(float32)
		return ok
	case ArgTypeBool:
		_, ok := value.(bool)
		return ok
	case ArgTypeString, ArgTypeFile, ArgTypeDirectory:
		_, ok := value.(string)
		return ok
	case ArgTypeVariadic:
		_, ok := value.([]string)
		return ok
	default:
		return true
	}
}

// getFlagType returns the type for a flag from cmd
func getFlagType(flagName string, cmd Handler) ArgumentType {
	if cmd == nil {
		return ""
	}

	if flagDef, err := cmd.GetFlag(flagName); err == nil {
		return flagDef.Type()
	}

	return ""
}

// getArgumentType returns the type for an argument from cmd
func getArgumentType(index int, cmd Handler) ArgumentType {
	if cmd == nil || index >= cmd.NumArgs() {
		return ""
	}

	return cmd.Args()[index].Type()
}

// isVariadicArgument checks if the argument at index is variadic
func isVariadicArgument(index int, cmd Handler) bool {
	if cmd == nil || len(cmd.Args()) == 0 || index >= len(cmd.Args()) {
		return false
	}
	return cmd.Args()[index].Type() == ArgTypeVariadic
}

// Variadic returns all remaining arguments as a []string
func (c *Context) Variadic(idx int) []string {

	var zero []string

	if len(c.args) <= idx {
		return zero
	}

	object := c.args[idx]

	if strs, ok := object.([]string); ok {
		return strs
	}

	panic(fmt.Sprintf("expected []string for variadic, got: %T", object))

}

func (c *Context) Manager() *Manager {
	return c.mgr
}

func (c *Context) IsFlagSet(flag string) bool {
	searchName := flag
	if c.handler != nil {
		if ch, ok := c.handler.(*commandHandler); ok {
			if fullName, isShorthand := ch.shorthands[flag]; isShorthand {
				searchName = fullName
			}
		}
	}
	return c.flags[searchName] != nil
}

func (c *Context) IsArgumentSet(index int) bool {
	return index < len(c.args)
}

func (c *Context) Flag(name string) string {
	value, err := findFlag[string](name, c)

	if err != nil && pkg.IsDebug() {
		panic(err)
	}

	return value
}

func (c *Context) IntFlag(name string) int {
	value, err := findFlag[int](name, c)

	if err != nil && pkg.IsDebug() {
		panic(err)
	}

	return value
}

func (c *Context) FloatFlag(name string) float32 {
	value, err := findFlag[float32](name, c)

	if err != nil && pkg.IsDebug() {
		panic(err)
	}

	return value
}

func (c *Context) BoolFlag(name string) bool {
	value, err := findFlag[bool](name, c)

	if err != nil && pkg.IsDebug() {
		panic(err)
	}

	return value
}

func (c *Context) Argument(index int) string {

	value, err := argWithDefault[string](index, c)

	if err != nil && pkg.IsDebug() {
		panic(err)
	}

	return value

}

func (c *Context) IntArgument(index int) int {
	value, err := argWithDefault[int](index, c)

	if err != nil && pkg.IsDebug() {
		panic(err)
	}

	return value
}

func (c *Context) FloatArgument(index int) float32 {
	value, err := argWithDefault[float32](index, c)

	if err != nil && pkg.IsDebug() {
		panic(err)
	}

	return value
}

func (c *Context) BoolArgument(index int) bool {
	value, err := argWithDefault[bool](index, c)

	if err != nil && pkg.IsDebug() {
		panic(err)
	}

	return value
}

func (c *Context) IsCancelled() bool {
	if c.ctx == nil {
		return false
	}
	select {
	case <-c.ctx.Done():
		return true
	default:
		return false
	}
}

func (c *Context) CancelError() error {
	if c.ctx == nil {
		return nil
	}
	return c.ctx.Err()
}

func (c *Context) ControlContext() context.Context {
	return c.ctx
}

func readFlag(part string, cmd Handler) (string, error) {

	// Long flag (double dash)
	if strings.HasPrefix(part, "--") {

		flagName := strings.TrimPrefix(part, "--")

		if len(flagName) <= 1 {
			return "", NewContextError(ErrUnknownFlag, part)
		}

		return strings.TrimPrefix(part, "--"), nil
	}

	// Short flag (single dash)
	if strings.HasPrefix(part, "-") && len(part) > 1 {

		shorthand := strings.TrimPrefix(part, "-")

		// Resolve shorthand to full flag name
		if ch, ok := cmd.(*commandHandler); ok {
			if fullName, exists := ch.shorthands[shorthand]; exists {
				return fullName, nil
			}
		}

		return "", NewContextError(ErrUnknownFlag, part)

	}

	return "", nil

}

func findFlag[T any](name string, c *Context) (T, error) {

	var zero T

	// Resolve shorthand to full name if needed
	searchName := name
	if c.handler != nil {
		if ch, ok := c.handler.(*commandHandler); ok {
			if fullName, isShorthand := ch.shorthands[name]; isShorthand {
				searchName = fullName
			}
		}
	}

	// Check if flag was provided
	if arg, exists := c.flags[searchName]; exists {
		if t, ok := arg.(T); ok {
			return t, nil
		}
		return zero, fmt.Errorf("expected %s argument, got: %T", reflect.TypeFor[T]().String(), arg)
	}

	// Check for default value
	if c.handler != nil {
		for _, flagDef := range c.handler.Flags() {
			if flagDef.Name() != searchName {
				continue
			}

			if def, ok := flagDef.DefaultValue(); ok {
				if t, ok2 := def.(T); ok2 {
					return t, nil
				}
				return zero, fmt.Errorf("default for %s has wrong type: %T", name, def)
			}
			if !flagDef.Required() {
				return zero, nil
			}
			break
		}
	}

	return zero, fmt.Errorf("flag %s not found", name)

}

func argWithDefault[T any](index int, c *Context) (T, error) {
	var zero T

	if c == nil {
		return zero, fmt.Errorf("missing context")
	}

	if index < len(c.args) {
		if t, ok := c.args[index].(T); ok {
			return t, nil
		}
		return zero, fmt.Errorf("expected %s argument, got: %T", reflect.TypeFor[T]().String(), c.args[index])
	}

	if c.handler == nil {
		return zero, fmt.Errorf("argument %d not found", index)
	}

	args := c.handler.Args()
	if index >= len(args) {
		return zero, fmt.Errorf("argument %d not found", index)
	}

	if def, ok := args[index].DefaultValue(); ok {
		if t, ok2 := def.(T); ok2 {
			return t, nil
		}
		return zero, fmt.Errorf("default for %s has wrong type: %T", args[index].Name(), def)
	}

	if !args[index].Required() {
		return zero, nil
	}

	return zero, fmt.Errorf("argument %d not found", index)
}
