package providers

import (
	"app/component/completion"
	"sort"
	"strings"
)

// FlagProvider completes flag names
type FlagProvider struct{}

func NewFlagProvider() *FlagProvider { return &FlagProvider{} }

func (p *FlagProvider) Fetch(ctx completion.Context) completion.Result {
	if ctx.Handler == nil {
		return completion.Result{}
	}

	var matches []string
	for _, flag := range ctx.Handler.Flags() {
		var token string
		if len(flag.Name()) > 1 {
			token = "--" + flag.Name()
		} else {
			token = "-" + flag.Name()
		}
		if strings.HasPrefix(token, ctx.Word) {
			matches = append(matches, token)
		}
	}
	sort.Strings(matches)
	return completion.Result{Completions: matches, DisplayNames: matches}
}

// ApplyTo replaces the trailing flag token with the chosen completion
func (p *FlagProvider) ApplyTo(rawInput string, chosen string) string {
	return completion.ReplaceLastToken(rawInput, chosen)
}
