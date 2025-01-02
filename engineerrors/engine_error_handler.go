package engineerrors

import (
	"fmt"
	"github.com/go-streamline/interfaces/definitions"
	"github.com/go-streamline/interfaces/utils"
	"github.com/google/uuid"
	"os"
	"path"
	"path/filepath"
	"streamline/config"
)

var ErrFailedToCreateDir = fmt.Errorf("failed to create directory")
var ErrFailedToCopyFile = fmt.Errorf("failed to copy source file to new path")

type engineErrorHandler struct {
	errorPath string
}

func CreateEngineErrorHandler(errorPath string) config.EngineErrorHandler {
	return &engineErrorHandler{
		errorPath: errorPath,
	}
}

func (e *engineErrorHandler) Handle(sessionUpdate definitions.SessionUpdate) error {
	if sessionUpdate.Error != nil && sessionUpdate.Finished {
		outputPath := path.Join(e.errorPath, sessionUpdate.SessionID.String())
		err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrFailedToCreateDir, err)
		}
		outputPath = path.Join(outputPath, uuid.NewString()+"_"+sessionUpdate.TPMark)
		err = utils.CopyFile(sessionUpdate.SourcePath, outputPath)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrFailedToCopyFile, err)
		}
	}

	return nil
}
