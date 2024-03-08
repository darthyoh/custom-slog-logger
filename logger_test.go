package customsloglogger

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"testing"
)

func TestHttp(t *testing.T) {
	mux := http.NewServeMux()

	logger := NewCustomLogger(os.Stderr, "userid")

	slog.SetDefault(logger)

	mux.HandleFunc("GET /url", func(w http.ResponseWriter, r *http.Request) {

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

	})

	if err := http.ListenAndServe("localhost:8080", mux); err != nil {
		log.Fatalf("unable to listen")
	}
}
