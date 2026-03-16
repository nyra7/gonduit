package providers

import (
	"app/command"
	"app/component/completion"
)

// FileSystemProvider provides completion sources for files and directories
type FileSystemProvider struct {
	file      completion.Provider
	directory completion.Provider
}

func NewFileSystemProvider(file, directory completion.Provider) *FileSystemProvider {
	return &FileSystemProvider{file: file, directory: directory}
}

func (p *FileSystemProvider) Fetch(ctx completion.Context) completion.Result {
	provider := p.resolve(ctx)
	if provider == nil {
		return completion.Result{}
	}
	return provider.Fetch(ctx)
}

func (p *FileSystemProvider) ApplyTo(rawInput string, chosen string) string {
	return completion.ReplaceLastToken(rawInput, chosen)
}

// resolve picks the concrete provider for the given context
func (p *FileSystemProvider) resolve(ctx completion.Context) completion.Provider {
	argType := expectedArgumentType(ctx)
	switch argType {
	case command.ArgTypeFile:
		return p.file
	case command.ArgTypeDirectory:
		return p.directory
	default:
		return nil
	}
}

// expectedArgumentType reads the handler's metadata to determine what type of value is expected at the cursor position
func expectedArgumentType(ctx completion.Context) command.ArgumentType {
	if ctx.Handler == nil {
		return ""
	}

	switch ctx.Type {
	case completion.TokenFlagValue:
		flag, err := ctx.Handler.GetFlag(ctx.FlagName)
		if err != nil {
			return ""
		}
		return flag.Type()

	case completion.TokenPositional:
		args := ctx.Handler.Args()
		if ctx.PositionalIndex < len(args) {
			return args[ctx.PositionalIndex].Type()
		}
	default:
		break
	}

	return ""

}
