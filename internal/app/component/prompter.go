package component

import (
	"app/style"
	"context"
	"errors"
	"sync/atomic"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

var (
	// ErrPromptActive is returned when a prompt is already active
	ErrPromptActive = errors.New("prompt already active")

	// ErrPromptCancelled is returned when a prompt is canceled
	ErrPromptCancelled = errors.New("prompt cancelled")
)

// Request represents a prompt request
type Request struct {
	Prompt   string
	Secure   bool
	Response chan Response
}

// Response represents a prompt response
type Response struct {
	Value string
	Err   error
}

// PromptRequestMsg is a Bubble Tea message for prompt requests
type PromptRequestMsg struct {
	Req Request
}

// Prompter handles prompting for user input in the application
type Prompter struct {
	requests chan Request
	active   *Request
	input    textinput.Model

	// Pre-configured styles
	normalStyle textinput.Styles
	promptStyle textinput.Styles

	// Saved state when prompting
	savedValue       string
	savedPlaceholder string
	savedEchoMode    textinput.EchoMode
	savedEchoChar    rune

	closed atomic.Bool
}

// NewPrompter creates a new prompter with pre-configured styles
func NewPrompter() *Prompter {

	normalStyle := textinput.Styles{
		Focused: textinput.StyleState{
			Prompt:      style.Primary,
			Text:        style.Text,
			Placeholder: style.Muted,
		},
	}

	promptStyle := textinput.Styles{
		Focused: textinput.StyleState{
			Prompt:      style.Primary,
			Text:        style.Text,
			Placeholder: style.Primary,
		},
	}

	ti := textinput.New()
	ti.Focus()
	ti.SetStyles(normalStyle)
	ti.Prompt = "> "
	ti.Placeholder = "Enter a command..."
	ti.EchoMode = textinput.EchoNormal

	return &Prompter{
		requests:         make(chan Request, 1),
		input:            ti,
		normalStyle:      normalStyle,
		promptStyle:      promptStyle,
		savedValue:       "",
		savedPlaceholder: ti.Placeholder,
		savedEchoMode:    ti.EchoMode,
	}

}

// PromptInput requests input from the user
func (m *Prompter) PromptInput(ctx context.Context, prompt string, secure bool) (string, error) {

	req := Request{
		Prompt:   prompt,
		Secure:   secure,
		Response: make(chan Response, 1),
	}

	select {
	case m.requests <- req:
	case <-ctx.Done():
		return "", ctx.Err()
	}

	select {
	case resp := <-req.Response:
		return resp.Value, resp.Err
	case <-ctx.Done():
		return "", ctx.Err()
	}

}

// WaitForRequest returns a Bubble Tea command that waits for a prompt request
func (m *Prompter) WaitForRequest() tea.Cmd {
	return func() tea.Msg {
		req, ok := <-m.requests
		if !ok {
			// Channel closed, return nil to stop processing
			return nil
		}
		return PromptRequestMsg{Req: req}
	}
}

// IsActive returns true if a prompt is currently active
func (m *Prompter) IsActive() bool {
	return m.active != nil
}

// HandleRequest processes a prompt request and configures the text input
func (m *Prompter) HandleRequest(req Request) tea.Cmd {

	if m.closed.Load() || (req.Prompt == "" && req.Response == nil) {
		// Ignore requests after close or zero-value requests
		return nil
	}

	if m.active != nil {
		req.Response <- Response{Err: ErrPromptActive}
		return m.WaitForRequest()
	}

	m.active = &req

	// Save current state
	m.savedValue = m.input.Value()
	m.savedPlaceholder = m.input.Placeholder
	m.savedEchoMode = m.input.EchoMode
	m.savedEchoChar = m.input.EchoCharacter

	// Configure for prompt mode
	m.input.SetValue("")
	m.input.SetStyles(m.promptStyle)
	m.input.Placeholder = req.Prompt
	if req.Secure {
		m.input.EchoMode = textinput.EchoPassword
		m.input.EchoCharacter = '*'
	} else {
		m.input.EchoMode = textinput.EchoNormal
	}

	return m.WaitForRequest()

}

// Submit submits the prompt response with the given value
func (m *Prompter) Submit(value string) {
	m.respond(value, nil)
}

// Cancel cancels the active prompt
func (m *Prompter) Cancel() {
	m.respond("", ErrPromptCancelled)
}

// Restore restores the saved text input state
func (m *Prompter) Restore() {
	m.input.SetValue(m.savedValue)
	m.input.SetStyles(m.normalStyle)
	m.input.Placeholder = m.savedPlaceholder
	m.input.EchoMode = m.savedEchoMode
	m.input.EchoCharacter = m.savedEchoChar
}

func (m *Prompter) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

func (m *Prompter) Value() string {
	return m.input.Value()
}

func (m *Prompter) SetValue(value string) {
	m.input.SetValue(value)
}

func (m *Prompter) Clear() {
	m.input.SetValue("")
}

func (m *Prompter) CursorEnd() {
	m.input.CursorEnd()
}

func (m *Prompter) View() string {
	return m.input.View()
}

// SetWidth sets the width of the text input
func (m *Prompter) SetWidth(width int) {
	m.input.SetWidth(width)
}

// Close closes the prompter and cleans up resources
func (m *Prompter) Close() {
	m.closed.Store(true)
	if m.requests != nil {
		close(m.requests)
	}
	m.active = nil
}

// respond sends a response to the active prompt
func (m *Prompter) respond(value string, err error) {
	if m.active != nil {
		m.active.Response <- Response{Value: value, Err: err}
		m.active = nil
	}
}
