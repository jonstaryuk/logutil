package logutil_test

import (
	"context"
	"os"

	"cloud.google.com/go/logging"
	"github.com/rs/zerolog/log"

	"github.com/jonstaryuk/logutil"
)

func ExampleStackdriverLoggingWriter() {
	client, err := logging.NewClient(context.Background(), "projects/my-project-id")
	if err != nil {
		panic(err)
	}
	defer client.Close()

	slw := logutil.StackdriverLoggingWriter{
		Logger: client.Logger("my-log-id"),
		Tee:    logutil.ConsoleWriterIfTerminal(os.Stderr, true),
	}
	log.Logger = log.Output(slw)
}
