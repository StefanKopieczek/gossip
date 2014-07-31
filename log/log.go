package log

import (
    "bytes"
    "fmt"
    "io"
    "log"
    "os"
    "runtime"
)

const c_STACK_BUFFER_SIZE int = 8192

type Level struct {
    Name string
    Level int
}

var (
    unspecified = Level{"NIL",    0}  // Discard first value so 0 can be a placeholder.
    DEBUG       = Level{"DEBUG",  1}
    FINE        = Level{"FINE",   2}
    INFO        = Level{"INFO",   3}
    WARN        = Level{"WARN",   4}
    SEVERE      = Level{"SEVERE", 5}
)

var c_DEFAULT_LOGGING_LEVEL = DEBUG
var c_DEFAULT_STACKTRACE_LEVEL = SEVERE

type Logger struct {
    *log.Logger
    Level Level
    StackTraceLevel Level
}

var defaultLogger *Logger

func New(out io.Writer, prefix string, flags int) (*Logger) {
    var logger Logger
    logger.Logger = log.New(out, prefix, flags)
    logger.Level = c_DEFAULT_LOGGING_LEVEL
    logger.StackTraceLevel = c_DEFAULT_STACKTRACE_LEVEL
    return &logger
}

func (l *Logger) Log(level Level, msg string, args ...interface{}) {
    if (level.Level < l.Level.Level) {
        return
    }

    msg = fmt.Sprintf(msg, args...)
    var buffer bytes.Buffer
    buffer.WriteString(level.Name)
    buffer.WriteString(": ")
    buffer.WriteString(msg)
    buffer.WriteString("\n")

    msg = level.Name + ": " + msg + "\n"

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
