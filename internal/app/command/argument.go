package command

type ArgumentType string

var (
	ArgTypeString    ArgumentType = "string"
	ArgTypeInt       ArgumentType = "int"
	ArgTypeFloat     ArgumentType = "float"
	ArgTypeBool      ArgumentType = "bool"
	ArgTypeFile      ArgumentType = "file"
	ArgTypeDirectory ArgumentType = "directory"
	ArgTypeVariadic  ArgumentType = "variadic"
)

type Argument interface {
	Name() string
	Description() string
	Type() ArgumentType
	Required() bool
	DefaultValue() (any, bool)
}

type commandArgument struct {
	name         string
	description  string
	argType      ArgumentType
	required     bool
	defaultValue any
	hasDefault   bool
}

func NewArgument(name, description string, argType ArgumentType, required bool, defaultValue ...any) Argument {
	arg := &commandArgument{
		name:        name,
		description: description,
		argType:     argType,
		required:    required,
	}
	if len(defaultValue) > 0 {
		arg.defaultValue = defaultValue[0]
		arg.hasDefault = true
	}
	return arg
}

func (ca *commandArgument) Name() string              { return ca.name }
func (ca *commandArgument) Description() string       { return ca.description }
func (ca *commandArgument) Type() ArgumentType        { return ca.argType }
func (ca *commandArgument) Required() bool            { return ca.required }
func (ca *commandArgument) DefaultValue() (any, bool) { return ca.defaultValue, ca.hasDefault }
