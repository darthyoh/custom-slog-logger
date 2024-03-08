package customsloglogger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
)

// definitions of ASCII colors for the logger
const (
	COLOR_RESET    = "\033[0m" //reset
	COLOR_DARKGRAY = "\033[90m"
	COLOR_RED      = "\033[31m"
	COLOR_BLUE     = "\033[34m"
	COLOR_YELLOW   = "\033[33m"
	COLOR_WHITE    = "\033[97m"
)

// colorize(colorCode, v) returns a colorized string of a string value.
func colorize(colorCode string, v string) string {
	return fmt.Sprintf("%s%s%s", colorCode, v, COLOR_RESET)
}

// CustomLoggerHandler is the custom slog handler
// The Handle() method of a slog.Handler takes a Record in parameter. But some informations (like line code of calling) ARE NOT INCLUDED in the record,
// so, a workaround is to use an inner slog.Handler and an inner bytes.Buffer : an inner call of Handle() will cause the generation of a log (containing the code line) in the inner buffer.
// In a second time, the useful informations are taken from the buffer, and passed to the final output writer
// a mutex is used for concurrence safety
type CustomLoggerHandler struct {
	writer           io.Writer   //output writer (os.Stderr for example)
	userid           string      //userid variable name in the context
	groupName        string      //groupName for WithGroup use on logger
	additionnalAttrs []slog.Attr //additional Attributes for With use on logger
}

// Enabled : necessary method of interface handler : accept all level
func (m *CustomLoggerHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

// WithAttrs : necessary method of interface handler : simple delegation to the inner handler
func (m *CustomLoggerHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := NewCustomLogger(m.writer, m.userid).Handler().(*CustomLoggerHandler)
	newHandler.groupName = m.groupName
	newHandler.additionnalAttrs = attrs
	return newHandler
}

// WithGroup : necessary method of interface handler : simple delegation to the inner handler
func (m *CustomLoggerHandler) WithGroup(name string) slog.Handler {
	newHandler := NewCustomLogger(m.writer, m.userid).Handler().(*CustomLoggerHandler)
	newHandler.groupName = name
	newHandler.additionnalAttrs = m.additionnalAttrs
	return newHandler
}

type ContextUser string

// Handle : customise the log
func (m *CustomLoggerHandler) Handle(ctx context.Context, r slog.Record) error {

	//defines color / log level
	color := COLOR_WHITE

	switch r.Level {
	case slog.LevelDebug:
		color = COLOR_DARKGRAY
	case slog.LevelInfo:
		color = COLOR_BLUE
	case slog.LevelWarn:
		color = COLOR_YELLOW
	case slog.LevelError:
		color = COLOR_RED
	}

	//init potentiel groupName prefixe
	groupPrefix := ""
	if m.groupName != "" {
		groupPrefix = fmt.Sprintf("%s.", m.groupName)
	}

	//init final attrs
	attrs := make([]string, 0)

	//getting and adding potentialy additionnal attr
	for _, attr := range m.additionnalAttrs {
		attrs = append(attrs, fmt.Sprintf("\t- %s%s : %s", groupPrefix, attr.Key, attr.Value))
	}

	//getting Record attributes
	r.Attrs(func(a slog.Attr) bool {

		attrs = append(attrs, fmt.Sprintf("\t- %s%s : %s", groupPrefix, a.Key, a.Value))
		return true
	})

	//concat output string
	attrsValues := ""
	if len(attrs) != 0 {
		attrsValues = fmt.Sprintf("\n%s", strings.Join(attrs, "\n"))
	}

	// getting source key
	source := ""
	if _, file, line, ok := runtime.Caller(2); ok {
		source = fmt.Sprintf("@%s:%d", filepath.Base(file), line)
	}

	//final display
	fmt.Fprintln(
		m.writer,
		colorize(color, fmt.Sprintf("===============%s================\n", r.Level.String())),
		colorize(color, r.Message),
		colorize(COLOR_DARKGRAY, fmt.Sprintf("\n %s %s", r.Time.Format("2006-01-02 15:04:05"), source)),
		attrsValues,
		colorize(color, "\n===================================="),
	)
	return nil
}

// NewCustomLogger : utility for creating a custom logger
func NewCustomLogger(outputWriter io.Writer, useridContextValue string) *slog.Logger {

	return slog.New(&CustomLoggerHandler{
		writer:           outputWriter,
		additionnalAttrs: make([]slog.Attr, 0),
		groupName:        "",
	})
}
