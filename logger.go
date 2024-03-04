package customsloglogger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
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
	writer  io.Writer     //output writer (os.Stderr for example)
	handler slog.Handler  //inner handler
	buffer  *bytes.Buffer //inner buffer
	mutex   *sync.Mutex   //mutex used for safe concurrency
	userid  string        //userid variable name in the context
}

// Enabled : necessary method of interface handler : simple delegation to the inner handler
func (m *CustomLoggerHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return m.handler.Enabled(ctx, level)
}

// WithAttrs : necessary method of interface handler : simple delegation to the inner handler
func (m *CustomLoggerHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CustomLoggerHandler{handler: m.handler.WithAttrs(attrs), buffer: m.buffer, mutex: m.mutex}
}

// WithGroup : necessary method of interface handler : simple delegation to the inner handler
func (m *CustomLoggerHandler) WithGroup(name string) slog.Handler {
	return &CustomLoggerHandler{handler: m.handler.WithGroup(name), buffer: m.buffer, mutex: m.mutex}
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

	//get the source position (source code)
	source := ""

	//this information is NOT contained in the Record
	//so, first call Handle() on the inner handler to get the source position
	//(this inner handler was created using the inner buffer as output writer...)
	m.mutex.Lock()
	defer func() {
		m.mutex.Unlock()
		m.buffer.Reset()
	}()

	if err := m.handler.Handle(ctx, r); err != nil { //generation of the log in the inner buffer
		return err
	}

	//extraction of the source position from the buffer to the source var
	sourceKeys := strings.Split(m.buffer.String(), "source=")
	if len(sourceKeys) == 2 {
		sourceLocations := strings.Split(sourceKeys[1], " ")
		if len(sourceLocations) > 1 {
			source = sourceLocations[0]
		}
	}

	//init userid from context
	userid := "nouser"

	userId, ok := ctx.Value(ContextUser(m.userid)).(string)
	if ok {
		userid = userId
	} else {
		//test if the client "simply pass userid as string..."
		if ctx.Value(m.userid) != nil {
			userid = fmt.Sprintf("%v", ctx.Value(m.userid))
		}
	}

	//getting Record attributes
	attrs := make([]string, 0)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, fmt.Sprintf(" - %s : %s", a.Key, a.Value))
		return true
	})

	//add source code position if found
	if source != "" {
		attrs = append(attrs, fmt.Sprintf("\n %s@%s", userid, source))
	}

	//concat output string
	attrsValues := ""
	if len(attrs) != 0 {
		attrsValues = fmt.Sprintf("\n%s", strings.Join(attrs, "\n"))
	}

	//final display
	fmt.Fprintln(
		m.writer,
		colorize(color, fmt.Sprintf("===============%s================\n", r.Level.String())),
		colorize(COLOR_DARKGRAY, r.Time.Format("[2006-01-02 15:04:05]")),
		colorize(color, r.Message),
		attrsValues,
		colorize(color, "\n===================================="),
	)
	return nil
}

// NewCustomLogger : utility for creating a custom logger
func NewCustomLogger(outputWriter io.Writer, useridContextValue string) *slog.Logger {
	b := &bytes.Buffer{}
	return slog.New(&CustomLoggerHandler{
		writer:  outputWriter,
		handler: slog.NewTextHandler(b, &slog.HandlerOptions{AddSource: true}),
		buffer:  b,
		mutex:   &sync.Mutex{},
		userid:  useridContextValue,
	})
}
