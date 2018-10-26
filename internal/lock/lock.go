package lock

import (
	"time"
)

const (
	name                 = ".grundy-lock-41xJwGFewWhrYZje"
	acquireTimeout       = 2 * time.Second
	inUseErr             = "Another instance of the application is already running"
	unableToCreatePrefix = "Failed to create lock - "
	unableToReadPrefix   = "Failed to read lock - "
)

type Lock interface {
	Acquire() error
	Errs() chan error
	Release()
}
