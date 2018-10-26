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
	mode = 0644
)

type unixLock struct {
	Lock
	mutex *sync.Mutex
	errs  chan error
	stop  chan chan struct{}
	path  string
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

	_, statErr := os.Stat(o.path)
	if statErr != nil {
		err := syscall.Mkfifo(o.path, mode)
		if err != nil {
			return &AcquireError{
				reason:     unableToCreatePrefix + err.Error(),
				createFail: true,
			}
		}
	}

	readResult := make(chan error)

	go func() {
		f, err := os.Open(o.path)
		readResult <- err
		f.Close()
	}()

	timeout := time.NewTimer(acquireTimeout)

	select {
	case err := <-readResult:
		if err != nil {
			return &AcquireError{
				reason:   unableToReadPrefix + err.Error(),
				readFail: true,
			}
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
			f, err := os.OpenFile(o.path, os.O_WRONLY, mode)
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

				_, err = f.WriteString(strconv.Itoa(os.Getpid()))
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
	os.Remove(o.path)
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
		path:  path.Join(parentDirPath, name),
		mutex: &sync.Mutex{},
		errs:  make(chan error),
		stop:  make(chan chan struct{}),
	}

	close(l.stop)

	return l
}
