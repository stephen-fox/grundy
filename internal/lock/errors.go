package lock

type AcquireError struct {
	reason     string
	createFail bool
	readFail   bool
	inUse      bool
	dirFail    bool
}

func (o *AcquireError) Error() string {
	return o.reason
}

func (o *AcquireError) FailedToCreated() bool {
	return o.createFail
}

func (o *AcquireError) ReadFailed() bool {
	return o.readFail
}

func (o *AcquireError) AnotherInstanceOwnsLock() bool {
	return o.inUse
}

func (o *AcquireError) FailedToCreateParentDirectory() bool {
	return o.dirFail
}
