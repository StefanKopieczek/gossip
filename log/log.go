package log

import (
    "bytes"
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
    INFO        = Level{"INFO",   2}
    WARN        = Level{"WARN",   3}
    SEVERE      = Level{"SEVERE", 4}
)

type Logger struct {
    *log.Logger
    Level Level
    StackTraceLevel Level
}

var defaultLogger *Logger

func New(out io.Writer, prefix string, flags int) (*Logger) {
    var logger Logger
    logger.Logger = log.New(out, prefix, flags)
    logger.Level = WARN
    logger.StackTraceLevel = SEVERE
    return &logger
}

func (l *Logger) Log(level Level, msg string) {
    if (level.Level < l.Level.Level) {
        return
    }

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

func (l *Logger) Debug(msg string) {
    l.Log(DEBUG, msg)
}

func (l *Logger) Info(msg string) {
    l.Log(INFO, msg)
}

func (l *Logger) Warn(msg string) {
    l.Log(WARN, msg)
}

func (l *Logger) Severe(msg string) {
    l.Log(SEVERE, msg)
}

func (l *Logger) PrintStack() {
    l.Logger.Printf(stackTrace())
}

func stackTrace() string {
    trace := make([]byte, c_STACK_BUFFER_SIZE)
    count := runtime.Stack(trace, true)
    return string(trace[:count])
}

func Debug(msg string) {
    if defaultLogger == nil {
        defaultLogger = New(os.Stderr, "", 0)
    }
    defaultLogger.Debug(msg)
}

func Info(msg string) {
    if defaultLogger == nil {
        defaultLogger = New(os.Stderr, "", 0)
    }
    defaultLogger.Info(msg)
}

func Warn(msg string) {
    if defaultLogger == nil {
        defaultLogger = New(os.Stderr, "", 0)
    }
    defaultLogger.Warn(msg)
}

func Severe(msg string) {
    if defaultLogger == nil {
        defaultLogger = New(os.Stderr, "", 0)
    }
    defaultLogger.Severe(msg)
}

func SetDefaultLogLevel(level Level) {
    if defaultLogger == nil {
        defaultLogger = New(os.Stderr, "", 0)
    }
    defaultLogger.Level = level
}
