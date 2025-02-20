package main

import (
	"context"
	"github.com/go-streamline/go-streamline/config"
	"github.com/go-streamline/go-streamline/standard"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	app := config.Initialize(
		ctx,
		os.Getenv("CONFIG_PATH"),
		standard.CreateProcessorFactory,
		standard.CreateDB,
		standard.CreateLoggerFactory,
		standard.CreateWriteAheadLogger,
		standard.CreateEngineErrorHandler,
	)
	go func() {
		err := app.Listen(":8080")
		if err != nil {
			logrus.WithError(err).Fatalf("could not init rest api")
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)
	<-ch
	cancel()
}
