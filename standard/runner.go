package standard

import (
	"context"
	"github.com/go-streamline/go-streamline/config"
	"github.com/go-streamline/interfaces/definitions"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"os"
	"os/signal"
	"syscall"
)

type AppPackage struct {
	Ctx                      context.Context
	ConfigPath               string
	CreateProcessorFactory   func(definitions.StateManagerFactory) definitions.ProcessorFactory
	CreateDB                 func(*config.Config) (*gorm.DB, error)
	LogFactoryCreator        func(cfg *config.Config) (definitions.LoggerFactory, error)
	CreateWriteAheadLogger   func(cfg *config.Config, logFactory definitions.LoggerFactory) (definitions.WriteAheadLogger, error)
	CreateEngineErrorHandler func(cfg *config.Config) config.EngineErrorHandler
	Port                     string
}

func GetDefaultAppPackage() *AppPackage {
	return &AppPackage{
		Ctx:                      context.Background(),
		ConfigPath:               "",
		CreateProcessorFactory:   CreateProcessorFactory,
		CreateDB:                 CreateDB,
		LogFactoryCreator:        CreateLoggerFactory,
		CreateWriteAheadLogger:   CreateWriteAheadLogger,
		CreateEngineErrorHandler: CreateEngineErrorHandler,
		Port:                     ":8080",
	}
}

func Run(appPackage *AppPackage) {
	ctx, cancel := context.WithCancel(appPackage.Ctx)
	configPath := appPackage.ConfigPath
	port := appPackage.Port
	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}
	if port == "" {
		port = os.Getenv("SERVER_PORT")
	}
	app := config.Initialize(
		ctx,
		os.Getenv("CONFIG_PATH"),
		CreateProcessorFactory,
		CreateDB,
		CreateLoggerFactory,
		CreateWriteAheadLogger,
		CreateEngineErrorHandler,
	)
	go func() {
		err := app.Listen(port)
		if err != nil {
			logrus.WithError(err).Fatalf("could not init rest api")
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)
	<-ch
	cancel()
}
