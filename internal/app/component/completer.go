package component

import (
	"app/command"
	"app/component/completion"
	"path/filepath"
	"strings"
)

type CycleDirection int

const (
	CycleForward  CycleDirection = 1
	CycleBackward CycleDirection = -1
)

// Completer is responsible for managing command-line completions, cycling through results, and providing UI hints
type Completer struct {
	manager      *command.Manager
	providers    map[completion.TokenType]completion.Provider
	active       bool
	lastInput    string
	activeCtx    completion.Context
	activeResult completion.Result
	index        int
}

func NewCompleter(m *command.Manager, providers map[completion.TokenType]completion.Provider) *Completer {
	return &Completer{
		manager:   m,
		providers: providers,
	}
}

func (c *Completer) Complete(input string, dir CycleDirection) string {
	if !c.active {
		ctx := completion.Tokenize(input, c.manager)
		p, ok := c.providers[ctx.Type]
		if !ok {
			return input
		}

		result := p.Fetch(ctx)
		if len(result.Completions) == 0 {
			return input
		}

		c.active = true
		c.lastInput = input
		c.activeCtx = ctx
		c.activeResult = result

		if dir == CycleBackward {
			c.index = len(result.Completions) - 1
		} else {
			c.index = 0
		}
	} else {
		n := len(c.activeResult.Completions)
		c.index = (c.index + int(dir) + n) % n
	}

	return c.applyActive()
}

// AcceptAndContinue accepts the currently highlighted completion
func (c *Completer) AcceptAndContinue(input string) (string, bool) {
	if !c.active || len(c.activeResult.Completions) == 0 {
		return input, false
	}

	chosen := c.activeResult.Completions[c.index]
	result := c.applyActive()
	c.Reset()

	isDir := strings.HasSuffix(strings.Trim(chosen, `"`), string(filepath.Separator))
	return result, isDir
}

// Reset discards all cycling state
func (c *Completer) Reset() {
	c.active = false
	c.index = 0
	c.lastInput = ""
	c.activeCtx = completion.Context{}
	c.activeResult = completion.Result{}
}

// DisplayNames returns the completion labels shown in the UI hint bar
func (c *Completer) DisplayNames() []string {
	if len(c.activeResult.DisplayNames) > 0 {
		return append([]string{}, c.activeResult.DisplayNames...)
	}
	return append([]string{}, c.activeResult.Completions...)
}

func (c *Completer) IsActive() bool { return c.active }
func (c *Completer) Index() int     { return c.index }
func (c *Completer) NumItems() int  { return len(c.activeResult.Completions) }

func (c *Completer) applyActive() string {
	p := c.providers[c.activeCtx.Type]
	chosen := c.activeResult.Completions[c.index]
	return p.ApplyTo(c.lastInput, chosen)
}
