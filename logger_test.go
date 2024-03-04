package customsloglogger

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"testing"
)

func TestHttp(t *testing.T) {
	mux := http.NewServeMux()

	logger := NewCustomLogger(os.Stderr, "userid")

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {

		//Info log : Simple Log, passing context without any attributes
		logger.LogAttrs(context.WithValue(r.Context(), ContextUser("userid"), "darthyoh"), slog.LevelInfo, "Welcome to API !!!")

		//Warn log : Warn Log, passing context with some attrs
		logger.LogAttrs(context.WithValue(r.Context(), "userid", "darthyoh"), slog.LevelWarn, "Warn level log", slog.String("warning message", "beware the dog"))

		//Error log : error level log, no context passed with attrs
		logger.LogAttrs(context.TODO(), slog.LevelError, "Error level log",
			slog.String("error message", "OUCHH !!!"),
			slog.Int("status code", 7),
		)

		fmt.Fprint(w, "Welcome to API !!!\n")
	})

	if err := http.ListenAndServe("localhost:8080", mux); err != nil {
		log.Fatalf("unable to listen")
	}
}
