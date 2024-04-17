/*
Package customsloglogger provides types and utilities for generating logs.

Logger implemented with this package will give you the possibility of :
  - generating leveled logs
  - on any io.Writer
  - with text colored log if needed
  - with all possibilities of standard slog log (source, additionnal attributes, grouped attributes)
  - and if needed, generating JSON logs that can be sent to a third-party http logging service
*/
package customsloglogger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// CtxKeyString is the customsloglogger type defined for passing keys in context
type CtxKeyString string

// Here are the definitions of ASCII colors for the logger.
// Theses colors will be used depending of the log level
const (
	COLOR_RESET    = "\033[0m"
	COLOR_DARKGRAY = "\033[90m"
	COLOR_RED      = "\033[31m"
	COLOR_BLUE     = "\033[34m"
	COLOR_YELLOW   = "\033[33m"
	COLOR_WHITE    = "\033[97m"
)

// colorize(colorCode, v) returns a colorized string of a string value.
func colorize(colorCode string, v string, colorized bool) string {

	if !colorized {
		return v
	}
	return fmt.Sprintf("%s%s%s", colorCode, v, COLOR_RESET)
}

// CustomHandlerOptions defines the behavior of the log handling
type CustomHandlerOptions struct {
	//AddSource causes the handler to compute the source code position
	//of the log statement and add a SourceKey attribute to the output.
	AddSource bool
	//ColorizeLors causes the handler to add colors to log
	//for text output, depending of the log level
	ColorizeLogs bool
	//JsonLogURL is the complete URL of a third-party logging service
	//if not empty, the handler will send json formatted log to it
	JsonLogURL string
	//MinimumLevel defines the minimum level considered to log (text or json)
	//If the slog.Record passed to the Handle() method has an inferior level to this one
	//it will be ignored
	MinimumLevel slog.Level
}

// CustomHandler is the custom slog handler, implementing the slog.Handler interface
type CustomHandler struct {
	//TextWriter is the io.Writer on which the text log (colorized or not) are written.
	//os.Stderr can ben used as an example
	TextWriter io.Writer
	//GroupName is an optional string the differents attributes will be grouped in
	//for text logging or for JSON logs sent to third party server.
	//GroupName can be passed when creating new CustomHandler but a better approach
	//is to generate a new CustomHandler from another one, using the WithGroup() method of the CustomLogger
	//to pass group name
	GroupName string
	//AdditionnalAttrs in a []slog.Attr containing additionnal attributes.
	//If empty, only attributes contained in the slog.Record will be handle.
	//If all logs contain the same attribute (url, userid, ...) it is a good
	//approach to add this attribute as "Additionnal Attribute" :
	//In this case, every log will contain it
	//AdditionnalAttrs can be passed when creating new CustomHandler but a better approach
	//is to generate a new CustomHandler from another one, using the With() method of the CustomLogger
	//to pass additionnal attributes
	AdditionnalAttrs []slog.Attr
	//CtxAttrsKeys in a []CtxKeyString containing additionnal attributes the Handle() function
	//will take in the logs from the context passed to it.
	//They correspond to keys of the context that will be log
	//CtxAttrsKeys can be passed when creating new CustomHandler but a better approach
	//is to generate a new CustomHandler from another one, using the WithCtxAttrsKeys of the CustomLogger
	//CtxAttrsKeys
	CtxAttrsKeys []CtxKeyString
	//Options are the *CustomHandlerOptions
	Options *CustomHandlerOptions
	//logText defines if the handler log in writer
	logText bool
	//sendJson defines if the handler send to json url
	logJson bool
	//add Mutex to concurrent safety while modifying logText or logJson
	*sync.Mutex
}

// Enabled : interface Handler method
// If true is returned, the Record will be handled.
// True is returned when the level of the Record is at least
// the minimum level defined in CustomHandlerOption
func (m *CustomHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= m.Options.MinimumLevel.Level()
}

func (l *CustomLogger) With(args ...any) *CustomLogger {
	return &CustomLogger{Logger: l.Logger.With(args...)}
}

func (l *CustomLogger) WithGroup(name string) *CustomLogger {
	return &CustomLogger{Logger: l.Logger.WithGroup(name)}
}

// WithAttrs : interface Handler method.
// This method is called when the With(attrs []slog.Attr) is called on an initial logger.
// It returns a new CustomHandler, based on the initial one
// (i.e. with the same TextWriter and same Options and same GroupName)
// but with AdditionnalAttrs that will be logged with each Record attributes
func (m *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := NewCustomLogger(m.TextWriter, m.Options).Handler()
	newHandler.GroupName = m.GroupName
	newHandler.AdditionnalAttrs = attrs
	return newHandler
}

// WithGroup : interface Handler method.
// This method is called when the WithGroup(group string) is called on an initial logger.
// It returns a new CustomHandler, based on the initial one
// (i.e. with the same TextWriter and same Options and same AdditionnalAttrs)
// but with a group name that will group every Record attributes
func (m *CustomHandler) WithGroup(name string) slog.Handler {
	newHandler := NewCustomLogger(m.TextWriter, m.Options).Handler()
	newHandler.GroupName = name
	newHandler.AdditionnalAttrs = m.AdditionnalAttrs
	return newHandler
}

// Handle : interface Handler method.
// This method is called when the slog.Record level is at least the minimum level
// defined in the CustomHandlerOptions.
// It will :
// - concat all slog.Record attributes with potential AdditionnalAttrs
// - group all theses attributes in a GroupName if defined
// - get the source code line if AddSource option is true
// - colorize all of this if ColorizeLog option is true
// - print all the result on the TextWriter if TextLog option is true
// - send all of this in json format to JsonLogUrl if this option is defined
// The sending to JsonLogUrl server will be "timed out" after 1 second
func (m *CustomHandler) Handle(ctx context.Context, r slog.Record) error {
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
	if m.GroupName != "" {
		groupPrefix = fmt.Sprintf("%s.", m.GroupName)
	}

	//init final text attrs
	textAttrs := make([]string, 0)

	//init final json attrs
	jsonAttrs := make([]slog.Attr, 0)

	//getting and adding potentialy additionnal attr
	for _, attr := range m.AdditionnalAttrs {
		textAttrs = append(textAttrs, fmt.Sprintf("\t- %s%s : %s", groupPrefix, attr.Key, attr.Value))
		jsonAttrs = append(jsonAttrs, attr)
	}

	//getting Record attributes
	r.Attrs(func(a slog.Attr) bool {
		textAttrs = append(textAttrs, fmt.Sprintf("\t- %s%s : %s", groupPrefix, a.Key, a.Value))
		jsonAttrs = append(jsonAttrs, a)
		return true
	})

	//getting potential context attributes
	for _, attr := range m.CtxAttrsKeys {
		v := ctx.Value(attr)
		if v == nil {
			v = ctx.Value(string(attr))
			if v == nil {
				continue
			}
		}
		value := fmt.Sprintf("%s", v)
		textAttrs = append(textAttrs, fmt.Sprintf("\t- %s%s : %s", groupPrefix, attr, value))
		jsonAttrs = append(jsonAttrs, slog.String(string(attr), value))
	}

	//concat output string
	textAttrsValues := ""
	if len(textAttrs) != 0 {
		textAttrsValues = fmt.Sprintf("\n%s", strings.Join(textAttrs, "\n"))
	}

	// getting source key
	source := ""
	if _, file, line, ok := runtime.Caller(2); ok && m.Options.AddSource {
		source = fmt.Sprintf("@%s:%d", filepath.Base(file), line)
	}

	//final display if logText is true
	if m.logText {
		fmt.Fprintln(
			m.TextWriter,
			colorize(color, fmt.Sprintf("===============%s================\n", r.Level.String()), m.Options.ColorizeLogs),
			colorize(color, r.Message, m.Options.ColorizeLogs),
			colorize(COLOR_DARKGRAY, fmt.Sprintf("\n %s %s", r.Time.Format(time.DateTime), source), m.Options.ColorizeLogs),
			textAttrsValues,
			colorize(color, "\n====================================", m.Options.ColorizeLogs),
		)
	}

	//sending to log microservice if option enables it
	if m.Options.JsonLogURL != "" && m.logJson {
		ch := make(chan int)

		jsonData := map[string]interface{}{
			"time":  r.Time.Format("2006-01-02 15:04:05"),
			"level": r.Level.String(),
			"msg":   r.Message,
		}

		if source != "" {
			jsonData["source"] = source
		}

		if m.GroupName != "" {
			groupMap := make(map[string]string)
			for _, attr := range jsonAttrs {
				groupMap[attr.Key] = attr.Value.String()
				jsonData[m.GroupName] = groupMap
			}
		} else {
			for _, attr := range jsonAttrs {
				jsonData[attr.Key] = attr.Value.String()
			}
		}

		jsonByte, err := json.Marshal(jsonData)
		if err != nil {
			return fmt.Errorf("unable to parse json request")
		}
		if req, err := http.NewRequest("POST", m.Options.JsonLogURL, bytes.NewReader(jsonByte)); err != nil {
			return fmt.Errorf("unable to create http request to send json log")
		} else {
			req.Header.Set("Content-Type", "application/json")

			go func() {
				defer func() {
					ch <- 1
				}()

				client := http.Client{}
				_, err := client.Do(req)
				if err != nil {
					fmt.Printf("error while sending to log service : %s\n", err)
				}
			}()

		}

		select {
		case <-ch:
		case <-time.After(1 * time.Second):
		}
	}

	return nil
}

// NewCustomLogger() creates a new CustomLogger.
// A CustomLogger is a logger based on the slog package.
// It takes the textWriter as the default io.Writer to write logs.
// The *CustomHandlerOptions options defines the default behavior or the logger.
// If nil is passed as options, the default behavior will be used :
// - Logs record only (without any additionnal attrs) will be print on the textWriter,
// - without beeing grouped in any group name,
// - and colorized,
// - adding source code,
// - for all logs with a minimum Level of slog.LevelInfo
// - without sending json log to third party server
func NewCustomLogger(textWriter io.Writer, options *CustomHandlerOptions) *CustomLogger {
	internalOptions := &CustomHandlerOptions{
		ColorizeLogs: true,
		AddSource:    true,
		JsonLogURL:   "",
		MinimumLevel: slog.LevelInfo,
	}

	if options != nil {
		internalOptions = options
	}

	newLogger := CustomLogger{
		slog.New(&CustomHandler{
			TextWriter:       textWriter,
			CtxAttrsKeys:     []CtxKeyString{},
			AdditionnalAttrs: make([]slog.Attr, 0),
			GroupName:        "",
			Options:          internalOptions,
			logText:          true,
			logJson:          true,
			Mutex:            &sync.Mutex{},
		})}

	return &newLogger

}

// CustomLogger is a wrapper around *slog.Logger
// it simply contains an anonymous *slog.Logger field
type CustomLogger struct {
	*slog.Logger
}

// WithCtxAttrsKeys method allows to generate a new *slogLogger
// based on the first one, adding a []string of keys representing
// context keys to check in Handle() to log
// Even if the keys are string, they are converted into CtxKeyString type
// to avoid type collision in context
func (c *CustomLogger) WithCtxAttrsKeys(keys []string) *CustomLogger {
	newHandler := c.Handler()
	for _, key := range keys {
		newHandler.CtxAttrsKeys = append(newHandler.CtxAttrsKeys, CtxKeyString(key))
	}
	return &CustomLogger{slog.New(newHandler)}
}

// Handler() return the *CustomHandler
func (c *CustomLogger) Handler() *CustomHandler {
	if h, ok := c.Logger.Handler().(*CustomHandler); ok {
		return h
	}
	return nil
}

// log() general method for logging, called for every Methods
func (c *CustomLogger) log(ctx context.Context, level slog.Level, msg string, logText, logJson bool, args ...any) {
	if h := c.Handler(); h != nil {
		h.Lock()
		defer h.Unlock()
		h.logJson = logJson
		h.logText = logText
	}
	c.Logger.Log(ctx, level, msg, args...)
}

// Log() re-defines the method of the inner slog.Logger, text and json logs are enable
func (c *CustomLogger) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	c.log(ctx, level, msg, true, true, args...)
}

// LogTextOnly() re-defines the method of the inner slog.Logger, text log is enable
func (c *CustomLogger) LogTextOnly(ctx context.Context, level slog.Level, msg string, args ...any) {
	c.log(ctx, level, msg, true, false, args...)
}

// LogJsonOnly() re-defines the method of the inner slog.Logger, json log is enable
func (c *CustomLogger) LogJsonOnly(ctx context.Context, level slog.Level, msg string, args ...any) {
	c.log(ctx, level, msg, false, true, args...)
}

// Warn() re-defines the method of the inner slog.Logger, text and json logs are enable
func (c *CustomLogger) Warn(msg string, args ...any) {
	c.log(context.TODO(), slog.LevelWarn, msg, true, true, args...)
}

// WarnTextOnly() re-defines the method of the inner slog.Logger, text log is enable
func (c *CustomLogger) WarnTextOnly(msg string, args ...any) {
	c.log(context.TODO(), slog.LevelWarn, msg, true, false, args...)
}

// WarnJsonOnly() re-defines the method of the inner slog.Logger, json log is enable
func (c *CustomLogger) WarnJsonOnly(msg string, args ...any) {
	c.log(context.TODO(), slog.LevelWarn, msg, false, true, args...)
}

// WarnContextTextOnly() re-defines the method of the inner slog.Logger, text log is enable
func (c *CustomLogger) WarnContextTextOnly(ctx context.Context, msg string, args ...any) {
	c.log(ctx, slog.LevelWarn, msg, true, false, args...)
}

// WarnContextJsonOnly() re-defines the method of the inner slog.Logger, json log is enable
func (c *CustomLogger) WarnContextJsonOnly(ctx context.Context, msg string, args ...any) {
	c.log(ctx, slog.LevelWarn, msg, false, true, args...)
}

// Info() re-defines the method of the inner slog.Logger, text and json logs are enable
func (c *CustomLogger) Info(msg string, args ...any) {
	c.log(context.TODO(), slog.LevelInfo, msg, true, true, args...)
}

// InfoTextOnly() re-defines the method of the inner slog.Logger, text log is enable
func (c *CustomLogger) InfoTextOnly(msg string, args ...any) {
	c.log(context.TODO(), slog.LevelInfo, msg, true, false, args...)
}

// InfoJsonOnly() re-defines the method of the inner slog.Logger, json log is enable
func (c *CustomLogger) InfoJsonOnly(msg string, args ...any) {
	c.log(context.TODO(), slog.LevelInfo, msg, false, true, args...)
}

// InfoContext() re-defines the method of the inner slog.Logger, text and json logs are enable
func (c *CustomLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	c.log(ctx, slog.LevelInfo, msg, true, true, args...)
}

// InfoContextTextOnly() re-defines the method of the inner slog.Logger, text log is enable
func (c *CustomLogger) InfoContextTextOnly(ctx context.Context, msg string, args ...any) {
	c.log(ctx, slog.LevelInfo, msg, true, false, args...)
}

// InfoContextJsonOnly() re-defines the method of the inner slog.Logger, json log is enable
func (c *CustomLogger) InfoContextJsonOnly(ctx context.Context, msg string, args ...any) {
	c.log(ctx, slog.LevelInfo, msg, false, true, args...)
}

// Error() re-defines the method of the inner slog.Logger, text and json logs are enable
func (c *CustomLogger) Error(msg string, args ...any) {
	c.log(context.TODO(), slog.LevelError, msg, true, true, args...)
}

// ErrorTextOnly() re-defines the method of the inner slog.Logger, text log is enable
func (c *CustomLogger) ErrorTextOnly(msg string, args ...any) {
	c.log(context.TODO(), slog.LevelError, msg, true, false, args...)
}

// ErrorJsonOnly() re-defines the method of the inner slog.Logger, json log is enable
func (c *CustomLogger) ErrorJsonOnly(msg string, args ...any) {
	c.log(context.TODO(), slog.LevelError, msg, false, true, args...)
}

// ErrorContext() re-defines the method of the inner slog.Logger, text and json logs are enable
func (c *CustomLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	c.log(ctx, slog.LevelError, msg, true, true, args...)
}

// ErrorContextTextOnly() re-defines the method of the inner slog.Logger, text log is enable
func (c *CustomLogger) ErrorContextTextOnly(ctx context.Context, msg string, args ...any) {
	c.log(ctx, slog.LevelError, msg, true, false, args...)
}

// ErrorContextJsonOnly() re-defines the method of the inner slog.Logger, json log is enable
func (c *CustomLogger) ErrorContextJsonOnly(ctx context.Context, msg string, args ...any) {
	c.log(ctx, slog.LevelError, msg, false, true, args...)
}

// Debug() re-defines the method of the inner slog.Logger, text and json logs are enable
func (c *CustomLogger) Debug(msg string, args ...any) {
	c.log(context.TODO(), slog.LevelDebug, msg, true, true, args...)
}

// DebugTextOnly() re-defines the method of the inner slog.Logger, text log is enable
func (c *CustomLogger) DebugTextOnly(msg string, args ...any) {
	c.log(context.TODO(), slog.LevelDebug, msg, true, false, args...)
}

// DebugJsonOnly() re-defines the method of the inner slog.Logger, json log is enable
func (c *CustomLogger) DebugJsonOnly(msg string, args ...any) {
	c.log(context.TODO(), slog.LevelDebug, msg, false, true, args...)
}

// DebugContext() re-defines the method of the inner slog.Logger, text and json logs are enable
func (c *CustomLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	c.log(ctx, slog.LevelDebug, msg, true, true, args...)
}

// DebugContextTextOnly() re-defines the method of the inner slog.Logger, text log is enable
func (c *CustomLogger) DebugContextTextOnly(ctx context.Context, msg string, args ...any) {
	c.log(ctx, slog.LevelDebug, msg, true, false, args...)
}

// DebugContextJsonOnly() re-defines the method of the inner slog.Logger, json log is enable
func (c *CustomLogger) DebugContextJsonOnly(ctx context.Context, msg string, args ...any) {
	c.log(ctx, slog.LevelDebug, msg, false, true, args...)
}

func (c *CustomLogger) logAttrs(ctx context.Context, level slog.Level, msg string, logText, logJson bool, attrs ...slog.Attr) {
	if h := c.Handler(); h != nil {
		h.Lock()
		defer h.Unlock()
		h.logJson = logJson
		h.logText = logText
	}
	c.Logger.LogAttrs(ctx, level, msg, attrs...)
}

// LogAttrs() re-defines the method of the inner slog.Logger, indicating if text and json logs are enable
func (c *CustomLogger) LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	c.logAttrs(ctx, level, msg, true, true, attrs...)
}

// LogAttrsTextOnly() re-defines the method of the inner slog.Logger, indicating if text log is enable
func (c *CustomLogger) LogAttrsTextOnly(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	c.logAttrs(ctx, level, msg, true, false, attrs...)
}

// LogAttrsJsonOnly() re-defines the method of the inner slog.Logger, indicating if json logs is enable
func (c *CustomLogger) LogAttrsJsonOnly(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	c.logAttrs(ctx, level, msg, false, true, attrs...)
}
