package settings

import (
	"os"
	"path"
	"runtime"
	"strings"
)

const (
	defaultDirMode         = 0755
	defaultSettingsDirname = ".grundy"
	logFilesDirName        = "logs"
	internalDirName        = ".internal"
)

func DirPath() string {
	var parentPath string

	switch runtime.GOOS {
	case "darwin":
		fallthrough
	case "linux":
		parentPath = os.Getenv("HOME")
	case "windows":
		parentPath = strings.Replace(os.Getenv("ProgramData"), "\\", "/", -1)
		if len(strings.TrimSpace(parentPath)) == 0 {
			parentPath = "/ProgramData"
		}
	}

	if len(strings.TrimSpace(parentPath)) == 0 {
		return "./" + defaultSettingsDirname
	}

	return path.Join(parentPath, defaultSettingsDirname)
}

func CreateInternalFilesDir(settingsDirPath string) (string, error) {
	dirPath := InternalFilesDir(settingsDirPath)

	err := CreateDir(dirPath)
	if err != nil {
		return "", err
	}

	return dirPath, nil
}

func InternalFilesDir(settingsDirPath string) string {
	return path.Join(settingsDirPath, internalDirName)
}

func CreateLogFilesDir(settingsDirPath string) (string, error) {
	dirPath := path.Join(settingsDirPath, logFilesDirName)

	err := CreateDir(dirPath)
	if err != nil {
		return "", err
	}

	return dirPath, nil
}

func CreateDir(dirPath string) error {
	return os.MkdirAll(dirPath, defaultDirMode)
}
