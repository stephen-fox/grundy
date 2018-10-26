package lock

import (
	"net"
	"sync"

	"github.com/Microsoft/go-winio"
)

const (
	lockUri = "\\\\.\\pipe\\" + name
)

type windowsLock struct {
	Lock
	mutex *sync.Mutex
	errs  chan error
	stop  chan chan struct{}
}

func (o *windowsLock) Acquire() error {
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

	timeout := acquireTimeout
	c, dialErr := winio.DialPipe(lockUri, &timeout)
	if dialErr == nil {
		close(o.stop)
		c.Close()
		return &AcquireError{
			reason: inUseErr,
			inUse:  true,
		}
	}

	listener, err := winio.ListenPipe(lockUri, &winio.PipeConfig{})
	if err != nil {
		close(o.stop)
		return &AcquireError{
			reason: unableToCreatePrefix + err.Error(),
			inUse:  true,
		}
	}

	go o.manage(listener)

	return nil
}

func (o *windowsLock) manage(listener net.Listener) {
	done := make(chan struct{})

	go func() {
		for {
			c, err := listener.Accept()
			select {
			case _, open := <-done:
				if !open {
					return
				}
			default:
				if err != nil {
					o.errs <- err
					continue
				}

				c.Close()
			}
		}
	}()

	c := <-o.stop
	close(done)
	listener.Close()
	c <- struct{}{}
}

func (o *windowsLock) Errs() chan error {
	return o.errs
}

func (o *windowsLock) Release() {
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
	l := &windowsLock{
		mutex: &sync.Mutex{},
		errs:  make(chan error),
		stop:  make(chan chan struct{}),
	}

	close(l.stop)

	return l
}
