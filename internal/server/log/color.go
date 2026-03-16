package log

import "fmt"

// ConsoleColor represents common console colors using ANSI escape codes
type ConsoleColor string

const (
	Black   ConsoleColor = "\033[30m"
	Red     ConsoleColor = "\033[31m"
	Green   ConsoleColor = "\033[32m"
	Yellow  ConsoleColor = "\033[33m"
	Blue    ConsoleColor = "\033[34m"
	Magenta ConsoleColor = "\033[35m"
	Cyan    ConsoleColor = "\033[36m"
	White   ConsoleColor = "\033[37m"
	Reset   ConsoleColor = "\033[0m"
)

// Styled returns a string with the color applied
func (c ConsoleColor) Styled(text string) string {
	return fmt.Sprintf("%s%s%s", c, text, Reset)
}
