package component

const maxHistorySize = 1000

// History represents a command history manager with navigation and state control functionality
type History struct {
	history      []string
	historyIndex int
	tempInput    string
}

func NewHistory() *History {
	return &History{}
}

// Add adds a command to history, avoiding duplicates at the end
func (h *History) Add(cmd string) {
	if cmd == "" {
		return
	}

	// Do not add if it's already the last item
	if len(h.history) > 0 && h.history[len(h.history)-1] == cmd {
		return
	}

	h.history = append(h.history, cmd)

	// Limit history size
	if len(h.history) > maxHistorySize {
		h.history = h.history[1:]
	}
}

// NavigateUp moves backward through command history
func (h *History) NavigateUp(inputValue string) string {

	if len(h.history) == 0 {
		return inputValue
	}

	// Save current input if we're at the bottom
	if h.historyIndex == -1 {
		h.tempInput = inputValue
		h.historyIndex = len(h.history) - 1
	} else if h.historyIndex > 0 {
		h.historyIndex--
	}

	return h.history[h.historyIndex]
}

// NavigateDown moves forward through command history
func (h *History) NavigateDown(inputValue string) string {

	if h.historyIndex == -1 {
		return inputValue
	}

	if h.historyIndex < len(h.history)-1 {
		h.historyIndex++
		return h.history[h.historyIndex]
	}

	// Back to current input
	h.historyIndex = -1
	h.tempInput = ""
	return h.tempInput

}

// Reset resets the history navigation state
func (h *History) Reset() {
	h.historyIndex = -1
	h.tempInput = ""
}

// Clear clears the history
func (h *History) Clear() {
	h.history = []string{}
	h.historyIndex = -1
}
