package log

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const c_STACK_BUFFER_SIZE int = 8192

// Number of frames to search up the stack for a function not in the log package.
// Used to find the function name of the entry point.
const c_NUM_STACK_FRAMES int = 5

// Format for stack info.
// The lengths of these two format strings should add up so that logs line up correctly.
// c_STACK_INFO_FMT takes three parameters: filename, line number, function name.
// c_NO_STACK_FMT takes one parameter: [Unidentified Location]
const c_STACK_INFO_FMT string = "%20.20s %04d %40.40s"
const c_NO_STACK_FMT string = "%-66s"

// Format for log level. ie. [FINE ]
const c_LOG_LEVEL_FMT string = "[%-5.5s]"

// Format for timestamps.
const c_TIMESTAMP_FMT string = "2006-01-02 15:04:05.000"

type Level struct {
	Name  string
	Level int
}

var (
	unspecified = Level{"NIL", 0} // Discard first value so 0 can be a placeholder.
	DEBUG       = Level{"DEBUG", 1}
	FINE        = Level{"FINE", 2}
	INFO        = Level{"INFO", 3}
	WARN        = Level{"WARN", 4}
	SEVERE      = Level{"SEVERE", 5}
)

var c_DEFAULT_LOGGING_LEVEL = INFO
var c_DEFAULT_STACKTRACE_LEVEL = SEVERE

type Logger struct {
	*log.Logger
	Level           Level
	StackTraceLevel Level
}

var defaultLogger *Logger

func New(out io.Writer, prefix string, flags int) *Logger {
	var logger Logger
	logger.Logger = log.New(out, prefix, flags)
	logger.Level = c_DEFAULT_LOGGING_LEVEL
	logger.StackTraceLevel = c_DEFAULT_STACKTRACE_LEVEL
	return &logger
}

func (l *Logger) Log(level Level, msg string, args ...interface{}) {
	if level.Level < l.Level.Level {
		return
	}

	// Get current time.
	now := time.Now()

	// Get information about the stack.
	// Try and find the first stack frame outside the logging package.
	// Only search up a few frames, it should never be very far.
	stackInfo := fmt.Sprintf(c_NO_STACK_FMT, "[Unidentified Location]")
	for depth := 0; depth < c_NUM_STACK_FRAMES; depth++ {
		if pc, file, line, ok := runtime.Caller(depth); ok {
			funcName := runtime.FuncForPC(pc).Name()
			funcName = path.Base(funcName)

			// Go up another stack frame if this function is in the logging package.
			isLog := strings.HasPrefix(funcName, "log.")
			if isLog {
				continue
			}

			// Now generate the string.
			stackInfo = fmt.Sprintf(c_STACK_INFO_FMT,
				filepath.Base(file),
				line,
				funcName)
			break
		}

		// If we get here, we failed to retrieve the stack information.
		// Just give up.
		break
	}

	// Write all the data into a buffer.
	// Format is:
	// [LEVEL]<timestamp> <file> <line> <function> - <message>
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf(c_LOG_LEVEL_FMT, level.Name))
	buffer.WriteString(now.Format(c_TIMESTAMP_FMT))
	buffer.WriteString(" ")
	buffer.WriteString(stackInfo)
	buffer.WriteString(" - ")
	buffer.WriteString(fmt.Sprintf(msg, args...))
	buffer.WriteString("\n")

	if level.Level >= l.StackTraceLevel.Level {
		buffer.WriteString("--- BEGIN stacktrace: ---\n")
		buffer.WriteString(stackTrace())
		buffer.WriteString("--- END stacktrace ---\n\n")
	}

	l.Logger.Printf(buffer.String())
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	l.Log(DEBUG, msg, args...)
}

func (l *Logger) Fine(msg string, args ...interface{}) {
	l.Log(FINE, msg, args...)
}

func (l *Logger) Info(msg string, args ...interface{}) {
	l.Log(INFO, msg, args...)
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	l.Log(WARN, msg, args...)
}

func (l *Logger) Severe(msg string, args ...interface{}) {
	l.Log(SEVERE, msg, args...)
}

func (l *Logger) PrintStack() {
	l.Logger.Printf(stackTrace())
}

func stackTrace() string {
	trace := make([]byte, c_STACK_BUFFER_SIZE)
	count := runtime.Stack(trace, true)
	return string(trace[:count])
}

func Debug(msg string, args ...interface{}) {
	if defaultLogger == nil {
		defaultLogger = New(os.Stderr, "", 0)
	}
	defaultLogger.Debug(msg, args...)
}

func Fine(msg string, args ...interface{}) {
	if defaultLogger == nil {
		defaultLogger = New(os.Stderr, "", 0)
	}
	defaultLogger.Fine(msg, args...)
}

func Info(msg string, args ...interface{}) {
	if defaultLogger == nil {
		defaultLogger = New(os.Stderr, "", 0)
	}
	defaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	if defaultLogger == nil {
		defaultLogger = New(os.Stderr, "", 0)
	}
	defaultLogger.Warn(msg, args...)
}

func Severe(msg string, args ...interface{}) {
	if defaultLogger == nil {
		defaultLogger = New(os.Stderr, "", 0)
	}
	defaultLogger.Severe(msg, args...)
}

func SetDefaultLogLevel(level Level) {
	if defaultLogger == nil {
		defaultLogger = New(os.Stderr, "", 0)
	}
	defaultLogger.Level = level
}
