// +build !windows

package lock

import (
	"os"
	"path"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const (
	dirMode  = 0755
	pipeMode = 0644
)

type unixLock struct {
	Lock
	mutex         *sync.Mutex
	errs          chan error
	stop          chan chan struct{}
	parentDirPath string
	pipePath      string
}

func (o *unixLock) Acquire() error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	select {
	case _, open := <-o.stop:
		if !open {
			o.stop = make(chan chan struct{})
		}
	default:
		return nil
	}

	err := os.MkdirAll(o.parentDirPath, dirMode)
	if err != nil {
		return err
	}

	_, statErr := os.Stat(o.pipePath)
	if statErr != nil {
		err := syscall.Mkfifo(o.pipePath, pipeMode)
		if err != nil {
			close(o.stop)
			return &AcquireError{
				reason:     unableToCreatePrefix + err.Error(),
				createFail: true,
			}
		}
	}

	readResult := make(chan error)

	go func() {
		f, err := os.Open(o.pipePath)
		readResult <- err
		f.Close()
	}()

	timeout := time.NewTimer(acquireTimeout)

	select {
	case err := <-readResult:
		// Another instance of the application owns the pipe
		// if we can read before the timeout occurs.

		close(o.stop)

		if err != nil {
			return &AcquireError{
				reason:   unableToReadPrefix + err.Error(),
				readFail: true,
			}
		}

		return &AcquireError{
			reason: inUseErr,
			inUse:  true,
		}
	case <-timeout.C:
		// No one is home.
	}

	go o.manage()

	return nil
}

func (o *unixLock) manage() {
	done := make(chan struct{})

	go func() {
		for {
			f, err := os.OpenFile(o.pipePath, os.O_WRONLY, pipeMode)
			select {
			case _, open := <-done:
				if !open {
					f.Close()
					return
				}
			default:
				if err != nil {
					o.errs <- err
					continue
				}

				_, err = f.WriteString(strconv.Itoa(os.Getpid()) + "\n")
				if err != nil {
					f.Close()
					o.errs <- err
					continue
				}

				f.Close()
			}
		}
	}()

	c := <-o.stop
	close(done)
	os.Remove(o.pipePath)
	c <- struct{}{}
}

func (o *unixLock) Errs() chan error {
	return o.errs
}

func (o *unixLock) Release() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	select {
	case _, open := <-o.stop:
		if !open {
			return
		}
	default:
	}

	c := make(chan struct{})
	o.stop <- c
	<-c

	close(o.stop)
}

func NewLock(parentDirPath string) Lock {
	l := &unixLock{
		parentDirPath: parentDirPath,
		pipePath:      path.Join(parentDirPath, name),
		mutex:         &sync.Mutex{},
		errs:          make(chan error),
		stop:          make(chan chan struct{}),
	}

	close(l.stop)

	return l
}
