package command

import (
	"fmt"
	"gonduit/util"
	"net"
	"reflect"
	"strconv"
	"strings"
)

type Context struct {
	Raw     string
	Command string
	args    []any
	flags   map[string]any
	Conn    net.Conn
	argIdx  int
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
)

func (e ContextError) Error() string {
	switch e {
	case ErrMissingRequiredArg:
		return "missing required argument"
	case ErrMissingRequiredFlag:
		return "missing required flag"
	case ErrInvalidArgType:
		return "invalid argument type"
	case ErrInvalidFlagType:
		return "invalid flag type"
	case ErrTooManyArgs:
		return "too many arguments provided"
	case ErrUnknownFlag:
		return "unknown flag"
	default:
		return "unknown context error"
	}
}

func NewContextError(code ContextError, message string) error {
	return fmt.Errorf("%w: %s", code, message)
}

func NewContext(raw string, cmd Handler, conn net.Conn) (*Context, error) {
	parts, err := util.SplitBash(raw)
	if err != nil {
		return nil, err
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	command := parts[0]
	parts = parts[1:]

	// Parse flags and arguments
	flags, args, err := parse(parts, cmd)
	if err != nil {
		return nil, err
	}

	// Validate against cmd
	if err = validate(args, flags, cmd); err != nil {
		return nil, err
	}

	return &Context{
		Raw:     raw,
		Command: command,
		args:    args,
		flags:   flags,
		Conn:    conn,
	}, nil
}

// parse extracts flags and positional arguments from parts
func parse(parts []string, cmd Handler) (map[string]any, []any, error) {
	flags := make(map[string]any)
	args := make([]any, 0, len(parts))

	for i := 0; i < len(parts); {
		part := parts[i]

		if strings.HasPrefix(part, "--") {
			flagName := strings.TrimPrefix(part, "--")

			// Check if flag is known
			if err := checkFlagExists(flagName, cmd); err != nil {
				return nil, nil, err
			}

			// Parse flag value
			flagValue, consumed, err := parseFlagValue(flagName, parts[i+1:], cmd)
			if err != nil {
				return nil, nil, err
			}

			flags[flagName] = flagValue
			i += consumed + 1
		} else {
			// Check if this is a variadic argument
			if isVariadicArgument(len(args), cmd) {
				// Consume all remaining parts as variadic
				variadicValues := parts[i:]
				args = append(args, variadicValues)
				break
			}

			// Check if too many arguments
			if err := checkArgumentCount(len(args), cmd); err != nil {
				return nil, nil, err
			}

			// Parse positional argument
			argValue := parseValue(part, getArgumentType(len(args), cmd))
			args = append(args, argValue)
			i++
		}
	}

	return flags, args, nil
}

// validate checks that all requirements are met
func validate(args []any, flags map[string]any, cmd Handler) error {
	if cmd == nil {
		return nil
	}

	// Validate arguments
	if err := validateArguments(args, cmd); err != nil {
		return err
	}

	// Validate flags
	if err := validateFlags(flags, cmd); err != nil {
		return err
	}

	if err := validateHandlerArgs(cmd.Args()); err != nil {
		return err
	}

	return nil
}

// checkFlagExists verifies that a flag is defined
func checkFlagExists(flagName string, cmd Handler) error {
	if cmd == nil || cmd.Flags == nil {
		return nil
	}

	if _, err := cmd.GetFlag(flagName); err != nil {
		return NewContextError(ErrUnknownFlag, flagName)
	}

	return nil
}

// checkArgumentCount verifies we haven't exceeded max arguments
func checkArgumentCount(currentCount int, cmd Handler) error {
	if cmd == nil || cmd.Args == nil {
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
	if len(remaining) == 0 || strings.HasPrefix(remaining[0], "--") {
		return nil, 0, NewContextError(ErrInvalidFlagType, fmt.Sprintf("%s: missing value", flagName))
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
	case ArgTypeString:
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
	if cmd == nil || cmd.Args == nil {
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
	if cmd == nil || cmd.Flags == nil {
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
	case ArgTypeString:
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
	if cmd == nil || cmd.Flags == nil {
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
	if cmd == nil || cmd.Args == nil || index >= len(cmd.Args()) {
		return false
	}
	return cmd.Args()[index].Type() == ArgTypeVariadic
}

func (c *Context) Next() (string, error) {
	return next[string](c)
}

// NextVariadic returns all remaining arguments as a []string
func (c *Context) NextVariadic() ([]string, error) {
	var zero []string

	if c.argIdx >= len(c.args) {
		return zero, fmt.Errorf("exhausted")
	}

	object := c.args[c.argIdx]
	c.argIdx++

	if strs, ok := object.([]string); ok {
		return strs, nil
	}

	return zero, fmt.Errorf("expected []string for variadic, got: %T", object)
}

func (c *Context) Flag(name string) (string, error) { return findFlag[string](name, c) }

func (c *Context) NextInt() (int, error) {
	return next[int](c)
}

func (c *Context) IntFlag(name string) (int, error) { return findFlag[int](name, c) }

func (c *Context) NextFloat() (float32, error) {
	return next[float32](c)
}

func (c *Context) FloatFlag(name string) (int, error) { return findFlag[int](name, c) }

func (c *Context) BoolFlag(name string) bool {
	b, err := findFlag[bool](name, c)
	return b && err == nil
}

func next[T any](c *Context) (T, error) {

	var zero T

	if c.argIdx >= len(c.args) {
		return zero, fmt.Errorf("exhausted")
	}

	object := c.args[c.argIdx]

	c.argIdx++

	if t, ok := object.(T); ok {
		return t, nil
	}

	return zero, fmt.Errorf("expected %s argument, got: %T", reflect.TypeFor[T]().String(), object)

}

func findFlag[T any](name string, c *Context) (T, error) {

	var zero T

	for flagName, arg := range c.flags {
		if flagName == name {
			if t, ok := arg.(T); ok {
				return t, nil
			}
			return zero, fmt.Errorf("expected %s argument, got: %T", reflect.TypeFor[T]().String(), flagName)
		}
	}

	return zero, fmt.Errorf("flag %s not found", name)

}
