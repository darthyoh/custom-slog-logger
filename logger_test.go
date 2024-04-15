package customsloglogger

import (
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

	mux.HandleFunc("GET /textjson", func(w http.ResponseWriter, r *http.Request) {
		logger := NewCustomLogger(os.Stderr,
			&CustomHandlerOptions{
				AddSource:    true,
				ColorizeLogs: true,
				JsonLogURL:   "http://localhost:8081/logs",
			})

		//Info log : Simple Log, passing context without any attributes
		logger.Info("Welcome to API !!!!")

		//Warn log : Warn Log, passing some attrs
		logger.Warn("Warn level log", "warning_message", "beware the dog !")

		//Error log : error level log, no context passed with attrs
		logger.Error("Error level log", "error_message", "OUCH", "statut_code", 7)

		//Debug log
		logger.Debug("Debug level log", "custom-arg", "a value")

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
		monoLogger.Info("test black and white")

		//same thing with an attr
		monoLoggerWithAttrs := monoLogger.With("url", r.URL)
		monoLoggerWithAttrs.Warn("warning !")

		//other logger, without colorized text output nor source
		monoLogger = NewCustomLogger(os.Stderr, &CustomHandlerOptions{ColorizeLogs: false, AddSource: false, MinimumLevel: 40})
		monoLogger.Info("test black and white")
	})

	mux.HandleFunc("GET /textonly", func(w http.ResponseWriter, r *http.Request) {
		logger := NewCustomLogger(os.Stderr,
			&CustomHandlerOptions{
				AddSource:    true,
				ColorizeLogs: true,
				JsonLogURL:   "http://localhost:8081/logs",
			})

		//Info log : Simple Log, passing context without any attributes
		logger.InfoTextOnly("Welcome to API !!!!")

		//Warn log : Warn Log, passing some attrs
		logger.WarnTextOnly("Warn level log", "warning_message", "beware the dog !")

		//Error log : error level log, no context passed with attrs
		logger.ErrorTextOnly("Error level log", "error_message", "OUCH", "statut_code", 7)

		//Debug log
		logger.DebugTextOnly("Debug level log", "custom-arg", "a value")

		//create a new logger from logger, with a group prefix before each attribute
		loggerWithGroupPrefix := logger.WithGroup("GroupPrefix")
		loggerWithGroupPrefix.WarnTextOnly("Warn level log with group prefix", "warning_message", "beware the dog")

		//create a new logger from logger, with some repetitives attributes
		loggerWithAttrs := logger.With("url", r.URL)
		loggerWithAttrs.ErrorTextOnly("Error level log with repetitive attribute", "error_message", "OUCH")

		//combining group and additionnal attributes
		loggerWithGroupAndAttrs := logger.WithGroup("AnotherPrefix").With("url", r.URL)
		loggerWithGroupAndAttrs.InfoTextOnly("Information", "info_message", "my message")

		//test other logger, without colorized text output
		monoLogger := NewCustomLogger(os.Stderr, &CustomHandlerOptions{ColorizeLogs: false, AddSource: true, MinimumLevel: 40})
		monoLogger.InfoTextOnly("test black and white")

		//same thing with an attr
		monoLoggerWithAttrs := monoLogger.With("url", r.URL)
		monoLoggerWithAttrs.WarnTextOnly("warning !")

		//other logger, without colorized text output nor source
		monoLogger = NewCustomLogger(os.Stderr, &CustomHandlerOptions{ColorizeLogs: false, AddSource: false, MinimumLevel: 40})
		monoLogger.InfoTextOnly("test black and white")
	})

	mux.HandleFunc("GET /jsononly", func(w http.ResponseWriter, r *http.Request) {
		logger := NewCustomLogger(os.Stderr,
			&CustomHandlerOptions{
				AddSource:    true,
				ColorizeLogs: true,
				JsonLogURL:   "http://localhost:8081/logs",
			})

		//Info log : Simple Log, passing context without any attributes
		logger.InfoJsonOnly("Welcome to API !!!!")

		//Warn log : Warn Log, passing some attrs
		logger.WarnJsonOnly("Warn level log", "warning_message", "beware the dog !")

		//Error log : error level log, no context passed with attrs
		logger.ErrorJsonOnly("Error level log", "error_message", "OUCH", "statut_code", 7)

		//Debug log
		logger.DebugJsonOnly("Debug level log", "custom-arg", "a value")

		//create a new logger from logger, with a group prefix before each attribute
		loggerWithGroupPrefix := logger.WithGroup("GroupPrefix")
		loggerWithGroupPrefix.WarnJsonOnly("Warn level log with group prefix", "warning_message", "beware the dog")

		//create a new logger from logger, with some repetitives attributes
		loggerWithAttrs := logger.With("url", r.URL)
		loggerWithAttrs.ErrorJsonOnly("Error level log with repetitive attribute", "error_message", "OUCH")

		//combining group and additionnal attributes
		loggerWithGroupAndAttrs := logger.WithGroup("AnotherPrefix").With("url", r.URL)
		loggerWithGroupAndAttrs.InfoJsonOnly("Information", "info_message", "my message")

		//test other logger, without colorized text output
		monoLogger := NewCustomLogger(os.Stderr, &CustomHandlerOptions{ColorizeLogs: false, AddSource: true, MinimumLevel: 40})
		monoLogger.InfoJsonOnly("test black and white")

		//same thing with an attr
		monoLoggerWithAttrs := monoLogger.With("url", r.URL)
		monoLoggerWithAttrs.WarnJsonOnly("warning !")

		//other logger, without colorized text output nor source
		monoLogger = NewCustomLogger(os.Stderr, &CustomHandlerOptions{ColorizeLogs: false, AddSource: false, MinimumLevel: 40})
		monoLogger.InfoJsonOnly("test black and white")
	})

	fmt.Println("Testing CustomLogger....")
	fmt.Println("try :")
	fmt.Println("- curl `http://localhost:8080/textjson` for testing text logging and json sending")
	fmt.Println("- curl `http://localhost:8080/textonly` for testing text logging only")
	fmt.Println("- curl `http://localhost:8080/jsononly` for testing json sending only")

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
