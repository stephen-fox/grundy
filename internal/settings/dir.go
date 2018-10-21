package settings

import (
	"os"
	"strings"
	"path"
	"runtime"
)

const (
	defaultDirMode         = 0755
	internalDirName        = ".internal"
	defaultSettingsDirname = ".grundy"
)

func DirPath() string {
	var parentPath string

	switch runtime.GOOS {
	case "darwin":
		fallthrough
	case "linux":
		parentPath = os.Getenv("HOME")
	case "windows":
		parentPath = strings.Replace(os.Getenv("USERPROFILE"), "\\", "/", -1)
	}

	return path.Join(parentPath, defaultSettingsDirname)
}

func CreateInternalFilesDir(settingsDirPath string) (string, error) {
	dirPath := path.Join(settingsDirPath, internalDirName)

	err := CreateDir(dirPath)
	if err != nil {
		return "", err
	}

	return dirPath, nil
}

func CreateDir(dirPath string) error {
	return os.MkdirAll(dirPath, defaultDirMode)
}
