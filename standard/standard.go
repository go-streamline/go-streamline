package standard

import (
	coreconfig "github.com/go-streamline/core/config"
	"github.com/go-streamline/core/logger"
	"github.com/go-streamline/core/wal"
	"github.com/go-streamline/interfaces/definitions"
	standard_processors_bundle "github.com/go-streamline/standard-processors-bundle"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path"
	"path/filepath"
	"streamline/config"
	"strings"
)

var (
	CreateProcessorFactory = standard_processors_bundle.Create
	CreateDB               = createDB
	CreateWriteAheadLogger = createWriteAheadLogger
	CreateLoggerFactory    = createLoggerFactory
)

func createDB(cfg *config.Config) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(path.Join(cfg.Engine.Workdir, "flow.db")), &gorm.Config{})
}

func createLoggerFactory(cfg *config.Config) (definitions.LoggerFactory, error) {
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		return nil, err
	}
	customLevels := make(map[string]logrus.Level)
	for k, v := range cfg.Logging.CustomLogLevels {
		l, err := logrus.ParseLevel(v)
		if err != nil {
			return nil, err
		}
		customLevels[k] = l
	}
	return logger.New(
		replaceRelativePath(cfg.Logging.Filename, cfg.Engine.Workdir),
		cfg.Logging.MaxSizeMB,
		cfg.Logging.MaxAgeDays,
		cfg.Logging.MaxBackups,
		cfg.Logging.Compress,
		cfg.Logging.LogToConsole,
		level,
		customLevels,
	)
}

func replaceRelativePath(path, workDir string) string {
	if strings.HasPrefix(path, "./") {
		// Remove the "./" prefix and concatenate with workDir
		newPath := filepath.Join(workDir, path[2:])
		return newPath
	}
	return path
}

func createWriteAheadLogger(cfg *config.Config, logFactory definitions.LoggerFactory) (definitions.WriteAheadLogger, error) {
	return wal.NewWriteAheadLogger(path.Join(cfg.Engine.Workdir, "wal"), coreconfig.WriteAheadLogging{
		Enabled:    cfg.WriteAheadLog.Enabled,
		MaxBackups: cfg.WriteAheadLog.MaxBackups,
		MaxSizeMB:  cfg.WriteAheadLog.MaxSizeMB,
		MaxAgeDays: cfg.WriteAheadLog.MaxAgeDays,
	}, logFactory)
}
