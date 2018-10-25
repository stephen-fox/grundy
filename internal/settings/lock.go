package settings

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

const (
	lockFilename       = ".lock"
	lockUpdateInterval = 5 * time.Second
)

type Lock interface {
	Acquire() error
	Errs() chan error
	Release()
}

type defaultLock struct {
	mutex         *sync.Mutex
	parentDirPath string
	errs          chan error
	stop          chan struct{}
}

func (o *defaultLock) Acquire() error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	select {
	case _, open := <-o.stop:
		if !open {
			o.stop = make(chan struct{})
		}
	default:
		return nil
	}

	filePath, err := createLock(o.parentDirPath)
	if err != nil {
		return err
	}

	go o.updateLock(filePath)

	return nil
}

func (o *defaultLock) Errs() chan error {
	return o.errs
}

func (o *defaultLock) Release() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	select {
	case _, open := <-o.stop:
		if !open {
			return
		}
	default:
	}

	close(o.stop)
}

func (o *defaultLock) updateLock(filePath string) {
	ticker := time.NewTicker(lockUpdateInterval)
	defer ticker.Stop()

	pid := os.Getpid()

	for {
		select {
		case <-ticker.C:
			err := ioutil.WriteFile(filePath, []byte(strconv.Itoa(pid)), defaultFileMode)
			if err != nil {
				o.errs <- err
			}
		case <-o.stop:
			return
		}
	}
}

func NewLock(internalSettingsDirPath string) Lock {
	l := &defaultLock{
		mutex:         &sync.Mutex{},
		parentDirPath: internalSettingsDirPath,
		errs:          make(chan error),
		stop:          make(chan struct{}),
	}

	close(l.stop)

	return l
}

func createLock(internalSettingsDirPath string) (string, error) {
	infos, err := ioutil.ReadDir(internalSettingsDirPath)
	if err != nil {
		return "", err
	}

	filePath := path.Join(internalSettingsDirPath, lockFilename)
	currentPid := os.Getpid()

	for _, in := range infos {
		if in.IsDir() || in.Name() != lockFilename {
			continue
		}

		if in.Size() > 100{
			break
		}

		raw, err := ioutil.ReadFile(filePath)
		if err != nil {
			return "", err
		}

		pid, convErr := strconv.Atoi(string(raw))
		if convErr != nil {
			break
		}

		if pid == currentPid {
			break
		}

		const wait = 10 * time.Second
		count := wait
		for {
			if count <= 0 {
				return "", errors.New("Another instance of the application " +
					"is already running as PID " + string(raw))
			}

			info, statErr := os.Stat(filePath)
			if statErr != nil {
				break
			}

			if time.Since(info.ModTime()) > wait {
				break
			}

			sleep := 1 * time.Second
			time.Sleep(sleep)
			count = count - sleep
		}

		break
	}

	err = ioutil.WriteFile(filePath, []byte(strconv.Itoa(currentPid)), defaultFileMode)
	if err != nil {
		return "", err
	}

	return filePath, nil
}
