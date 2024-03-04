package adndataslog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// définition des couleurs ASCII pour le logger
const (
	COLOR_RESET    = "\033[0m" //reset (repasse en couleur standard)
	COLOR_DARKGRAY = "\033[90m"
	COLOR_RED      = "\033[31m"
	COLOR_BLUE     = "\033[34m"
	COLOR_YELLOW   = "\033[33m"
	COLOR_WHITE    = "\033[97m"
)

// colorize permet de retourner la chaine de caractère colorisée
func colorize(colorCode string, v string) string {
	return fmt.Sprintf("%s%s%s", colorCode, v, COLOR_RESET)
}

// CustomLoggerHandler est un handler de slog personnalisé
// il dispose d'un handler encapsulé : un appel explicite de la méthode Handle() de ce handler produira la génération d'un log dans un buffer
// ainsi, le CustomLoggerHandler pourra récupérer des informations qui ne seront PAS passé par le Record (comme la source)
type CustomLoggerHandler struct {
	writer  io.Writer     //writer de sortie
	handler slog.Handler  //texthandler sous jacent
	buffer  *bytes.Buffer //buffer permettant le handling sans affichage direct vers le io.writer
	mutex   *sync.Mutex   //mutex utilisée pour la safe concurrence
}

// Enabled nécessaire pour l'interface handler : simple délégation au handler sous jacent
func (m *CustomLoggerHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return m.handler.Enabled(ctx, level)
}

// WithAttrs nécessaire pour l'interface handler : simple délégation au handler sous jacent
func (m *CustomLoggerHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CustomLoggerHandler{handler: m.handler.WithAttrs(attrs), buffer: m.buffer, mutex: m.mutex}
}

// WithGroup nécessaire pour l'interface handler : simple délégation au handler sous jacent
func (m *CustomLoggerHandler) WithGroup(name string) slog.Handler {
	return &CustomLoggerHandler{handler: m.handler.WithGroup(name), buffer: m.buffer, mutex: m.mutex}
}

// Handle permet la personnalisation du log
func (m *CustomLoggerHandler) Handle(ctx context.Context, r slog.Record) error {

	//définition de la couleur en fonction du niveau de log
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

	//emplacement du log (code source)
	source := ""

	//récupération de la source :
	//cette information est spécifique au logger et n'est PAS contenu dans le Record
	//par conséquent, il convient d'appliquer explicitement Handle() sur handler sous jacent qui lui générera le flag "source"
	//le handler sous jacent a été conçu pour que son io.Writer soit le buffer interne
	m.mutex.Lock()
	defer func() {
		m.mutex.Unlock()
		m.buffer.Reset()
	}()

	if err := m.handler.Handle(ctx, r); err != nil { //le log du handle interne sera généré dans le buffer
		return err
	}

	//extraction de la source dans le buffer et récupération du flag "source"
	sourceKeys := strings.Split(m.buffer.String(), "source=")
	if len(sourceKeys) == 2 {
		sourceLocations := strings.Split(sourceKeys[1], " ")
		if len(sourceLocations) > 1 {
			source = sourceLocations[0]
		}
	}

	//initialisation de l'utilisateur
	userid := "nouser"

	if ctx.Value("userid") != nil {
		userid = fmt.Sprintf("%s", ctx.Value("userid"))
	}

	//récupération des attributs de log
	attrs := make([]string, 0)

	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, fmt.Sprintf(" - %s : %s", a.Key, a.Value))
		return true
	})

	//ajout de la source si présente
	if source != "" {
		attrs = append(attrs, fmt.Sprintf("\n %s@%s", userid, source))
	}

	//concaténation en chaine
	attrsValues := ""
	if len(attrs) != 0 {
		attrsValues = fmt.Sprintf("\n%s", strings.Join(attrs, "\n"))
	}

	//affichage final
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

// NewCustomLogger est une utilité pour générer un CustomLoggerHandler disposant du buffer sous jacent
func NewCustomLogger() *slog.Logger {
	b := &bytes.Buffer{}
	return slog.New(&CustomLoggerHandler{
		writer:  os.Stderr,
		handler: slog.NewTextHandler(b, &slog.HandlerOptions{AddSource: true}),
		buffer:  b,
		mutex:   &sync.Mutex{}})
}
