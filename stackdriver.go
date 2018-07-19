package logutil

import (
	"context"
	"io"
	"io/ioutil"
	"os"

	"cloud.google.com/go/logging"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh/terminal"
)

// A StackdriverLoggingWriter accepts pre-encoded JSON messages and writes
// them to Google Stackdriver Logging. It implements zerolog.LevelWriter and
// maps Zerolog levels to Stackdriver levels.
//
// If Tee is not nil, it receives a copy of each write.
type StackdriverLoggingWriter struct {
	Logger *logging.Logger
	Tee    io.Writer
}

// Write always returns len(p), nil.
func (w *StackdriverLoggingWriter) Write(p []byte) (int, error) {
	w.Logger.Log(logging.Entry{Payload: rawJSON(p)})

	if w.Tee != nil {
		w.Tee.Write(p)
	}

	return len(p), nil
}

// WriteLevel implements zerolog.LevelWriter. It always returns len(p), nil.
func (w *StackdriverLoggingWriter) WriteLevel(level zerolog.Level, p []byte) (int, error) {
	severity := logging.Default

	// More efficient than logging.ParseSeverity(level.String())
	switch level {
	case zerolog.DebugLevel:
		severity = logging.Debug
	case zerolog.InfoLevel:
		severity = logging.Info
	case zerolog.WarnLevel:
		severity = logging.Warning
	case zerolog.ErrorLevel:
		severity = logging.Error
	case zerolog.FatalLevel:
		severity = logging.Critical
	case zerolog.PanicLevel:
		severity = logging.Critical
	}

	w.Logger.Log(logging.Entry{Payload: rawJSON(p), Severity: severity})

	if w.Tee != nil {
		if lw, ok := w.Tee.(zerolog.LevelWriter); ok {
			lw.WriteLevel(level, p)
		} else {
			w.Tee.Write(p)
		}
	}

	return len(p), nil
}

func (w *StackdriverLoggingWriter) Flush() error {
	return w.Logger.Flush()
}

// UseStackdriverLogging causes the global zerolog/log.Logger to write
// (properly structured, leveled) payloads to Stackdriver logging. The
// returned client should be closed before the program exits.
//
// The labels argument is ignored if opts includes CommonLabels.
func UseStackdriverLogging(project, logID string, labels map[string]string, opts ...logging.LoggerOption) (*logging.Client, error) {
	client, err := logging.NewClient(context.Background(), "projects/"+project)
	if err != nil {
		return nil, errors.Wrap(err, "create client")
	}
	if err := client.Ping(context.Background()); err != nil {
		return nil, errors.Wrap(err, "ping")
	}

	// labels comes before opts so that any CommonLabels in opts take precedence.
	opts = append([]logging.LoggerOption{logging.CommonLabels(labels)}, opts...)
	log.Logger = zerolog.New(&StackdriverLoggingWriter{Logger: client.Logger(logID, opts...)})

	return client, nil
}

// MustUseStackdriverLogging calls UseStackdriverLogging and panics if it returns an error.
func MustUseStackdriverLogging(project, logID string, labels map[string]string, opts ...logging.LoggerOption) *logging.Client {
	closer, err := UseStackdriverLogging(project, logID, labels, opts...)
	if err != nil {
		panic(err)
	}
	return closer
}

// ConsoleWriterIfTerminal returns a zerolog.ConsoleWriter if f is a terminal.
// Otherwise, it returns f.
func ConsoleWriterIfTerminal(f *os.File, colorful bool) io.Writer {
	if terminal.IsTerminal(int(f.Fd())) {
		return zerolog.ConsoleWriter{Out: f}
	}

	return ioutil.Discard
}

type rawJSON []byte

func (r rawJSON) MarshalJSON() ([]byte, error)  { return []byte(r), nil }
func (r *rawJSON) UnmarshalJSON(b []byte) error { *r = rawJSON(b); return nil }
