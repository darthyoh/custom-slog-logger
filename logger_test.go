package customsloglogger

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"testing"
)

func logJSONServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /logs", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("data received on JSON Server !!!")

		if jsonBody, err := io.ReadAll(r.Body); err != nil {
			fmt.Println("unable to read request body")
		} else {
			fmt.Println(string(jsonBody))
		}

	})
	fmt.Println("Started JSON log server on port 8081.")
	if err := http.ListenAndServe("localhost:8081", mux); err != nil {
		log.Fatalf("unable to listen")
	}
}

func mainServer() {

	mux := http.NewServeMux()

	logger := NewCustomLogger(os.Stderr, nil)

	slog.SetDefault(logger.Logger)

	mux.HandleFunc("GET /url", func(w http.ResponseWriter, r *http.Request) {

		//Info log : Simple Log, passing context without any attributes
		logger.Info("Welcome to API !!!!", true, false)

		//Warn log : Warn Log, passing some attrs
		logger.Warn("Warn level log", true, false, "warning_message", "beware the dog !")

		//Error log : error level log, no context passed with attrs
		logger.Error("Error level log", true, false, "error_message", "OUCH", "statut_code", 7)

		//Debug log
		logger.Debug("Debug level log", true, false, "custom-arg", "a value")

		//create a new logger from logger, with a group prefix before each attribute
		loggerWithGroupPrefix := logger.WithGroup("GroupPrefix")
		loggerWithGroupPrefix.Warn("Warn level log with group prefix", "warning_message", "beware the dog")

		//create a new logger from logger, with some repetitives attributes
		loggerWithAttrs := logger.With("url", r.URL)
		loggerWithAttrs.Error("Error level log with repetitive attribute", "error_message", "OUCH")

		//combining group and additionnal attributes
		loggerWithGroupAndAttrs := logger.WithGroup("AnotherPrefix").With("url", r.URL)
		loggerWithGroupAndAttrs.Info("Information", "info_message", "my message")

		//test other logger, without colorized text output
		monoLogger := NewCustomLogger(os.Stderr, &CustomHandlerOptions{ColorizeLogs: false, AddSource: true, MinimumLevel: 40})
		monoLogger.Info("test black and white", true, false)

		//same thing with an attr
		monoLoggerWithAttrs := monoLogger.With("url", r.URL)
		monoLoggerWithAttrs.Warn("warning !")

		//other logger, without colorized text output nor source
		monoLogger = NewCustomLogger(os.Stderr, &CustomHandlerOptions{ColorizeLogs: false, AddSource: false, MinimumLevel: 40})
		monoLogger.Info("test black and white", true, false)

		//test logger with sending http json log to another microservice
		jsonLogger := NewCustomLogger(os.Stderr, &CustomHandlerOptions{
			AddSource: true, JsonLogURL: "http://localhost:8081/logs",
		}).With("url", r.URL).WithGroup("values")
		jsonLogger.Info("foo")

		//test logger with context args
		ctx := context.WithValue(r.Context(), CtxKeyString("id"), "my id")
		ctxLogger := NewCustomLogger(os.Stderr, &CustomHandlerOptions{
			ColorizeLogs: true,
			AddSource:    true,
		}).WithCtxAttrsKeys([]string{"id"})

		ctxLogger.Log(ctx, slog.LevelWarn, "warning with ctx attrs", false, true)

		fmt.Fprintf(w, "Done !")

	})

	fmt.Println("Started test server on port 8080.")
	if err := http.ListenAndServe("localhost:8080", mux); err != nil {
		log.Fatalf("unable to listen")
	}
}

func TestHttp(t *testing.T) {

	go logJSONServer()
	go mainServer()

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()

}
