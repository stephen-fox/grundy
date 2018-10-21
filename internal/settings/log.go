package settings

import (
	"os"
	"time"
	"path"
)

const (
	logFileExtension = ".log"
)

func LogFile(settingsDirPath string) (*os.File, error) {
	dirPath, err := CreateLogFilesDir(settingsDirPath)
	if err != nil {
		return &os.File{}, err
	}

	filePath := path.Join(dirPath, time.Now().Format("2006-01-02") + logFileExtension)

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, defaultFileMode)
	if err != nil {
		return &os.File{}, err
	}

	return f, nil
}
