package providers

import (
	"app/command"
	"app/component/completion"
	"sort"
	"strings"
)

// CommandProvider completes command names from the registry
type CommandProvider struct {
	manager *command.Manager
}

func NewCommandProvider(m *command.Manager) *CommandProvider {
	return &CommandProvider{manager: m}
}

func (p *CommandProvider) Fetch(ctx completion.Context) completion.Result {
	var matches []string
	for _, h := range p.manager.Handlers() {
		if strings.HasPrefix(h.Name(), ctx.Word) {
			matches = append(matches, h.Name())
		}
	}
	sort.Strings(matches)
	return completion.Result{Completions: matches, DisplayNames: matches}
}

func (p *CommandProvider) ApplyTo(_ string, chosen string) string {
	return chosen
}
