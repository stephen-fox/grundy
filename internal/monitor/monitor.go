package monitor

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

const (
	defaultRefreshDelay = 10 * time.Second
)

// Monitor monitors a directory's sub directories for a given file.
// Refer to Config for configuration details.
type Monitor interface {
	Start()
	Stop()
}

type defaultMonitor struct {
	config Config
	stop   chan struct{}
}

func (o *defaultMonitor) Start() {
	if o.stop != nil {
		return
	}

	o.stop = make(chan struct{})

	go func() {
		delay := defaultRefreshDelay
		if o.config.RefreshDelay > 0 {
			delay = o.config.RefreshDelay
		}

		ticker := time.NewTicker(delay)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				o.config.Results <- o.produce()
			case <-o.stop:
				return
			}
		}
	}()
}

func (o *defaultMonitor) produce() ScanResult {
	subInfos, err := ioutil.ReadDir(o.config.RootDirPath)
	if err != nil {
		return ScanResult{
			Err: err,
		}
	}

	result := ScanResult{
		FilesOfInterest: make(map[string]os.FileInfo),
	}

	for _, sub := range subInfos {
		if !sub.IsDir() {
			continue
		}

		subDirPath := path.Join(o.config.RootDirPath, sub.Name())

		children, childErr := ioutil.ReadDir(subDirPath)
		if childErr != nil {
			continue
		}

		for _, c := range children {
			if c.IsDir() || c.Name() != o.config.MatchFileName {
				continue
			}

			cPath := path.Join(subDirPath, c.Name())

			result.FilesOfInterest[cPath] = c
		}
	}

	return result
}

func (o *defaultMonitor) Stop() {
	if o.stop != nil {
		close(o.stop)
	}

	o.stop = nil
}

type ScanResult struct {
	Err             error
	FilesOfInterest map[string]os.FileInfo
}

func (o ScanResult) IsErr() bool {
	return o.Err != nil
}

// Config specifies the configuration for a Monitor.
//
// Consider the following file tree:
//	My Files/
//	|
//	|-- text-files/
//	|  |
//	|  |-- SomeFile.txt
//	|
//	|-- stuff/
//	|  |
//	|  |-- Awesome.cfg
//	|
//	|-- gorbage/
//	   |
//	   |-- CoolStoryBro.txt
//
// If you specify the root directory to scan as 'My Files', and the match
// file name as 'Awesome.cfg', the Monitor will notify the caller that the
// file exists using the specified Results channel for the given RefreshDelay.
type Config struct {
	RefreshDelay  time.Duration
	RootDirPath   string
	MatchFileName string
	Results       chan ScanResult
}

func (o Config) IsValid() error {
	if len(strings.TrimSpace(o.RootDirPath)) == 0 {
		return errors.New("The specified directory path cannot not be empty")
	}

	if len(strings.TrimSpace(o.MatchFileName)) == 0 {
		return errors.New("The specified file name to match cannot not be empty")
	}

	if o.Results == nil {
		return errors.New("The results channel cannot be nil")
	}

	return nil
}

func NewMonitor(options Config) (Monitor, error) {
	err := options.IsValid()
	if err != nil {
		return &defaultMonitor{}, err
	}

	return &defaultMonitor{
		config: options,
	}, nil
}
