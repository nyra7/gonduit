package completion

// Result holds the completions produced by a Provider
type Result struct {
	// Completions contains the full string options to be inserted into the input line
	Completions []string

	// DisplayNames contains the short labels to be shown in the UI hint bar
	DisplayNames []string
}

// Provider is the interface that a completion source must satisfy
type Provider interface {
	// Fetch returns all candidates that matches ctx.Word
	Fetch(ctx Context) Result

	// ApplyTo rewrites rawInput so that its trailing token is replaced with the chosen completion string
	ApplyTo(rawInput string, completion string) string
}
