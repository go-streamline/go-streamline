package config

import (
	"context"
	"github.com/go-streamline/core/flow/persist"
	"github.com/go-streamline/core/state"
	"github.com/go-streamline/core/zookeeper"
	engineconfig "github.com/go-streamline/engine/configuration"
	"github.com/go-streamline/engine/engine"
	"github.com/go-streamline/go-streamline/http"
	"github.com/go-streamline/interfaces/definitions"
	"github.com/go-zookeeper/zk"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"os"
	"path"
	"strings"
	"time"
)

type Config struct {
	Engine struct {
		Workdir           string `yaml:"workdir"`
		MaxWorkers        int    `yaml:"maxWorkers"`
		FlowCheckInterval int    `yaml:"flowCheckInterval"`
		FlowBatchSize     int    `yaml:"flowBatchSize"`
		ErrorHandlingPath string `yaml:"errorHandlingPath"`
		DB                struct {
			DSN string `yaml:"dsn"`
		} `yaml:"db"`
	} `yaml:"engine"`
	Zookeeper struct {
		Path             string `yaml:"path"`
		ConnectionString string `yaml:"connectionString"`
	} `yaml:"zookeeper"`
	WriteAheadLog struct {
		Enabled    bool `yaml:"enabled"`
		MaxBackups int  `yaml:"maxBackups"`
		MaxSizeMB  int  `yaml:"maxSizeMB"`
		MaxAgeDays int  `yaml:"maxAgeDays"`
	} `yaml:"writeAheadLog"`
	Logging struct {
		Level           string            `yaml:"level"`
		Filename        string            `yaml:"filename"`
		MaxSizeMB       int               `yaml:"maxSizeMB"`
		MaxAgeDays      int               `yaml:"maxAgeDays"`
		MaxBackups      int               `yaml:"maxBackups"`
		Compress        bool              `yaml:"compress"`
		LogToConsole    bool              `yaml:"logToConsole"`
		CustomLogLevels map[string]string `yaml:"customLogLevel"`
	} `yaml:"logging"`
}

type EngineErrorHandler interface {
	Handle(sessionUpdate definitions.SessionUpdate) error
}

func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SetupZookeeper(cfg *Config, logFactory definitions.LoggerFactory) (*zk.Conn, definitions.Coordinator, error) {
	var zkClient *zk.Conn
	var coordinator definitions.Coordinator
	if cfg.Zookeeper.ConnectionString != "" {
		zkClient, _, err := zk.Connect(strings.Split(cfg.Zookeeper.ConnectionString, ","), 5*time.Second)
		if err != nil {
			return nil, nil, err
		}

		leaderSelector := zookeeper.NewZookeeperLeaderSelector(
			zkClient,
			path.Join(cfg.Zookeeper.Path, "coordinator"),
			logFactory,
			"prime_",
		)
		err = leaderSelector.Start()
		if err != nil {
			return nil, nil, err
		}
		coordinator = zookeeper.NewCoordinator(
			leaderSelector,
			zkClient,
			path.Join(cfg.Zookeeper.Path, "coordinator", "leaders"),
			logFactory,
		)
	}
	return zkClient, coordinator, nil
}

func SetupStateManagement(cfg *Config, zkClient *zk.Conn) definitions.StateManagerFactory {
	return state.NewStateManagerFactory(zkClient, path.Join(cfg.Engine.Workdir, "state"), cfg.Zookeeper.Path)
}

func Initialize(
	ctx context.Context,
	configPath string,
	createProcessorFactory func(definitions.StateManagerFactory) definitions.ProcessorFactory,
	createDB func(*Config) (*gorm.DB, error),
	logFactoryCreator func(cfg *Config) (definitions.LoggerFactory, error),
	setupWriteAheadLogger func(cfg *Config, factory definitions.LoggerFactory) (definitions.WriteAheadLogger, error),
	engineErrorHandlerCreator func(cfg *Config) EngineErrorHandler,
) *fiber.App {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		logrus.WithError(err).Fatal("failed to load configuration")
	}
	logFactory, err := logFactoryCreator(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create logger factory")
	}

	log := logFactory.GetLogger("main", "main")

	writeAheadLogger, err := setupWriteAheadLogger(cfg, logFactory)
	if err != nil {
		log.WithError(err).Fatal("failed to initialize write-ahead logger")
	}

	zkClient, coordinator, err := SetupZookeeper(cfg, logFactory)
	if err != nil {
		log.WithError(err).Fatal("failed to setup Zookeeper")
	}

	stateManagementFactory := SetupStateManagement(cfg, zkClient)
	processorFactory := createProcessorFactory(stateManagementFactory)
	db, err := createDB(cfg)
	if err != nil {
		log.WithError(err).Fatal("failed to open database")
	}

	flowManager, err := persist.NewDBFlowManager(db, logFactory)
	if err != nil {
		log.WithError(err).Fatal("failed to initialize flow manager")
	}

	e, err := engine.New(&engineconfig.Config{
		Workdir:           cfg.Engine.Workdir,
		MaxWorkers:        cfg.Engine.MaxWorkers,
		FlowCheckInterval: cfg.Engine.FlowCheckInterval,
		FlowBatchSize:     cfg.Engine.FlowBatchSize,
	}, writeAheadLogger, logFactory, processorFactory, flowManager, coordinator)
	if err != nil {
		log.WithError(err).Fatal("failed to initialize engine")
	}

	err = e.Run()
	if err != nil {
		log.WithError(err).Fatal("failed to run engine")
	}
	engineErrorHandler := engineErrorHandlerCreator(cfg)
	go func() {
		for {
			select {
			case sessionUpdate := <-e.SessionUpdates():
				if sessionUpdate.Error != nil {
					err := engineErrorHandler.Handle(sessionUpdate)
					if err != nil {
						log.WithError(err).Error("failed to handle session update")
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return http.NewFlowManagerAPI(flowManager, processorFactory)
}
