package main

import (
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"streamline/config"
	"streamline/standard"
	"syscall"
)

func main() {
	app := config.Initialize(
		os.Args[1],
		standard.CreateProcessorFactory,
		standard.CreateDB,
		standard.CreateLoggerFactory,
		standard.CreateWriteAheadLogger,
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
}
