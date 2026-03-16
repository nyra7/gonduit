package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"shared/pkg"
	"strings"
	"sync"
	"time"
)

type Level int

var (
	LevelDebug   Level = 0
	LevelVerbose Level = 1
	LevelInfo    Level = 2
)

var replacer = strings.NewReplacer(
	"\n", "\\n",
	"\r", "\\r",
	"\t", "\\t",
	"\b", "\\b",
	"\f", "\\f",
	"\v", "\\v",
)

// One lock for all logging output (file + console) to keep lines from interleaving
var logMu sync.Mutex
var file *os.File
var level = LevelDebug
var consoleOutput = true

// InitLogger Initializes the logger
func InitLogger(logLevel Level, fileName string) {

	if fileName != "" {

		// Save the path for the app log
		filePath := filepath.Clean(logPath(fileName))

		// Create a variable to store the error
		var err error

		// Try to create directories for the log file. Panic on failure
		if err = os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
			panic(err)
		}

		// Try to open the log file, panic on failure
		if file, err = os.OpenFile(filePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0o600); err != nil {
			panic(err)
		}

	}

	level = logLevel

}

// SilenceConsole Silences the console output
func SilenceConsole() {
	consoleOutput = false
}

// Infof logs an info message with format string
func Infof(message string, a ...any) {

	if level > LevelInfo {
		return
	}

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Print the message to the console
	consolePrint(fmt.Sprintf(message, a...), "INFO", Cyan, timestamp, os.Stdout)

	// Log the message with INFO severity
	log(fmt.Sprintf(message, a...), "INFO", timestamp)

}

// Info logs an info message
func Info(a ...any) {

	if level > LevelInfo {
		return
	}

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Format the message
	message := fmt.Sprint(a...)

	// Print the message to the console
	consolePrint(message, "INFO", Cyan, timestamp, os.Stdout)

	// Log the message with INFO severity
	log(message, "INFO", timestamp)

}

// Successf logs a success message with format string
func Successf(message string, a ...any) {

	if level > LevelInfo {
		return
	}

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Print the message to the console
	consolePrint(fmt.Sprintf(message, a...), "SUCCESS", Green, timestamp, os.Stdout)

	// Log the message with WARNING severity
	log(fmt.Sprintf(message, a...), "SUCCESS", timestamp)

}

// Success logs a success message
func Success(a ...any) {

	if level > LevelInfo {
		return
	}

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Format the message
	message := fmt.Sprint(a...)

	// Print the message to the console
	consolePrint(message, "SUCCESS", Green, timestamp, os.Stdout)

	// Log the message with WARNING severity
	log(message, "SUCCESS", timestamp)

}

// Warnf logs a warning message with format string
func Warnf(message string, a ...any) {

	if level > LevelInfo {
		return
	}

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Print the message to the console
	consolePrint(fmt.Sprintf(message, a...), "WARNING", Yellow, timestamp, os.Stdout)

	// Log the message with WARNING severity
	log(fmt.Sprintf(message, a...), "WARNING", timestamp)

}

// Warn logs a warning message
func Warn(a ...any) {

	if level > LevelInfo {
		return
	}

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Format the message
	message := fmt.Sprint(a...)

	// Print the message to the console
	consolePrint(message, "WARNING", Yellow, timestamp, os.Stdout)

	// Log the message with WARNING severity
	log(message, "WARNING", timestamp)

}

// Errorf logs an error message with format string
func Errorf(message string, a ...any) {

	if level > LevelInfo {
		return
	}

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Print the message to the console
	consolePrint(fmt.Sprintf(message, a...), "ERROR", Red, timestamp, os.Stderr)

	// Log the message with ERROR severity
	log(fmt.Sprintf(message, a...), "ERROR", timestamp)

}

// Error logs an error message
func Error(a ...any) {

	if level > LevelInfo {
		return
	}

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Format the message
	message := fmt.Sprint(a...)

	// Print the message to the console
	consolePrint(message, "ERROR", Red, timestamp, os.Stderr)

	// Log the message with ERROR severity
	log(message, "ERROR", timestamp)

}

// Verbosef logs a verbose message with format string
func Verbosef(message string, a ...any) {

	// Do not log if level is greater than verbose
	if level > LevelVerbose {
		return
	}

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Print the message to the console
	consolePrint(fmt.Sprintf(message, a...), "VERBOSE", Magenta, timestamp, os.Stdout)

	// Log the message with VERBOSE severity
	log(fmt.Sprintf(message, a...), "VERBOSE", timestamp)

}

// Verbose logs a verbose message
func Verbose(a ...any) {

	// Do not log if level is greater than verbose
	if level > LevelVerbose {
		return
	}

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Format the message
	message := fmt.Sprint(a...)

	// Print the message to the console
	consolePrint(message, "VERBOSE", Magenta, timestamp, os.Stdout)

	// Log the message with VERBOSE severity
	log(message, "VERBOSE", timestamp)

}

// Debugf logs a debug message with format string
func Debugf(message string, a ...any) {

	// Do not log if level is greater than debug
	if level > LevelDebug {
		return
	}

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Print the message to the console
	consolePrint(fmt.Sprintf(message, a...), "DEBUG", White, timestamp, os.Stdout)

	// Log the message with DEBUG severity
	log(fmt.Sprintf(message, a...), "DEBUG", timestamp)

}

// Debug logs a debug message
func Debug(a ...any) {

	// Do not log if level is greater than debug
	if level > LevelDebug {
		return
	}

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Format the message
	message := fmt.Sprint(a...)

	// Print the message to the console
	consolePrint(message, "DEBUG", White, timestamp, os.Stdout)

	// Log the message with DEBUG severity
	log(message, "DEBUG", timestamp)

}

// Fatalf logs a fatal error with format string and panics
func Fatalf(message string, a ...any) {

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Print the message to the console
	consolePrint(fmt.Sprintf(message, a...), "FATAL", Red, timestamp, os.Stderr)

	// Log the message with FATAL severity
	log(fmt.Sprintf(message, a...), "FATAL", timestamp)

	// Trigger a breakpoint if debug is enabled
	if os.Getenv("DEBUG") != "" {
		runtime.Breakpoint()
	}

	// Exit the program
	os.Exit(1)

}

// Fatal logs a fatal error and panics
func Fatal(a ...any) {

	// Get the current timestamp
	timestamp := time.Now().Format(time.RFC1123)

	// Format the message
	message := fmt.Sprint(a...)

	// Print the message to the console
	consolePrint(message, "FATAL", Red, timestamp, os.Stderr)

	// Log the message with FATAL severity
	log(message, "FATAL", timestamp)

	// Trigger a breakpoint if debug is enabled
	if os.Getenv("DEBUG") != "" {
		runtime.Breakpoint()
	}

	// Exit the program
	os.Exit(1)

}

// CloseLogger closes the log file
func CloseLogger() {
	if file != nil {
		_ = file.Close()
	}
}

// log logs a message with the specified category and log level
func log(message, level string, timestamp string) {

	// Append log to file if logFile is available
	if file != nil {

		// Replace escape sequences with their escaped representations to keep a single line in the log
		message = replacer.Replace(message)

		// Format the log message
		logMessage := fmt.Sprintf("[%s] [%s] (%s): %s\n", timestamp, level, pkg.AppName, message)

		logMu.Lock()
		defer logMu.Unlock()

		_, _ = file.WriteString(logMessage)

	}

}

// consolePrint prints a formatted message to the console
func consolePrint(message, level string, color ConsoleColor, timestamp string, file io.Writer) {

	if !consoleOutput {
		return
	}

	// Replace escape sequences with their escaped representations to keep a single line in the log
	message = replacer.Replace(message)

	logMu.Lock()
	defer logMu.Unlock()

	// Print the formatted message to the console
	_, _ = fmt.Fprintf(file, "[%s] [%s%s%s] (%s): %s\n", timestamp, color, level, Reset, pkg.AppName, message)

}

// logPath returns the write path for the input log file
func logPath(file string) string {

	// Get the user configuration directory
	// dir, err := os.UserConfigDir()

	// Return the original file path on failure
	// if err != nil {
	//	 return file
	// }

	// Add the app name and the file name to the path
	return filepath.Join(pkg.AppName, file)

}
