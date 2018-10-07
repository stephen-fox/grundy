package monitor

import (
	"os"
	"path"
	"testing"
	"time"
)

func TestConfig_IsValid(t *testing.T) {
	emptyErr := Config{}.IsValid()
	if emptyErr == nil {
		t.Error("Empty config did not generate an error")
	}

	noDirErr := Config{
		MatchFileName: "bla",
		Results:       make(chan ScanResult),
	}.IsValid()
	if noDirErr == nil {
		t.Error("Empty directory path did not generate an error")
	}

	noChannelErr := Config{
		RootDirPath:   "fsfds",
		MatchFileName: "akdka",
	}.IsValid()
	if noChannelErr == nil {
		t.Error("Empty results channel did not generate an error")
	}

	err := Config{
		RootDirPath:   "fdf",
		MatchFileName: "bla",
		Results:       make(chan ScanResult),
	}.IsValid()
	if err != nil {
		t.Error("Valid config generated an error -", err.Error())
	}
}

func TestNewMonitor(t *testing.T) {
	config := Config{}
	_, err := NewMonitor(config)
	if err == nil {
		t.Error("Empty config did not generate an error")
	}

	config = Config{
		MatchFileName: "bla",
		Results:       make(chan ScanResult),
	}
	_, err = NewMonitor(config)
	if err == nil {
		t.Error("Empty directory path did not generate an error")
	}

	config = Config{
		RootDirPath:   "fsfds",
		MatchFileName: "akdka",
	}
	_, err = NewMonitor(config)
	if err == nil {
		t.Error("Empty results channel did not generate an error")
	}

	config = Config{
		RootDirPath:   "fdf",
		MatchFileName: "bla",
		Results:       make(chan ScanResult),
	}
	_, err = NewMonitor(config)
	if err != nil {
		t.Error("Valid config generated an error -", err.Error())
	}
}

func TestDefaultMonitor_Start(t *testing.T) {
	current, err := os.Getwd()
	if err != nil {
		t.Error(err.Error())
	}

	oneUp := path.Dir(current)

	config := Config{
		RefreshDelay:  1 * time.Second,
		RootDirPath:   oneUp,
		MatchFileName: "doc.go",
		Results:       make(chan ScanResult),
	}
	m, err := NewMonitor(config)
	if err != nil {
		t.Error("Valid config generated an error -", err.Error())
	}
	defer m.Stop()

	m.Start()

	result := <-config.Results
	if result.IsErr() {
		t.Error("Monitoring failed unexpectedly -", result.Err)
	}

	exp := path.Join(current, config.MatchFileName)

	if _, ok := result.FilesOfInterest[exp]; !ok {
		t.Error("Monitor did not find expected file -", exp)
	}
}

func TestDefaultMonitor_StartMultipleTimes(t *testing.T) {
	current, err := os.Getwd()
	if err != nil {
		t.Error(err.Error())
	}

	oneUp := path.Dir(current)

	config := Config{
		RefreshDelay:  1 * time.Second,
		RootDirPath:   oneUp,
		MatchFileName: "doc.go",
		Results:       make(chan ScanResult),
	}
	m, err := NewMonitor(config)
	if err != nil {
		t.Error("Valid config generated an error -", err.Error())
	}
	defer m.Stop()

	m.Start()
	m.Start()
	m.Start()
	m.Start()
	m.Start()

	var count int
	for i := 0; i < 5; i++ {
		time.Sleep(config.RefreshDelay)
		select {
		case <-config.Results:
			count++
		default:
			continue
		}
	}

	if count > 5 {
		t.Error("Too many results from monitor -", count)
	}
}

func TestDefaultMonitor_Stop(t *testing.T) {
	current, err := os.Getwd()
	if err != nil {
		t.Error(err.Error())
	}

	oneUp := path.Dir(current)

	config := Config{
		RefreshDelay:  1 * time.Second,
		RootDirPath:   oneUp,
		MatchFileName: "doc.go",
		Results:       make(chan ScanResult),
	}
	m, err := NewMonitor(config)
	if err != nil {
		t.Error("Valid config generated an error -", err.Error())
	}

	m.Start()

	m.Stop()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var count int
	for {
		select {
		case <-ticker.C:
			return
		case <-config.Results:
			count++
		}
	}

	if count > 1 {
		t.Error("Monitor produced more than one result after it was stopped -", count)
	}
}

func TestDefaultMonitor_StopMultipleTimes(t *testing.T) {
	current, err := os.Getwd()
	if err != nil {
		t.Error(err.Error())
	}

	oneUp := path.Dir(current)

	config := Config{
		RefreshDelay:  1 * time.Second,
		RootDirPath:   oneUp,
		MatchFileName: "doc.go",
		Results:       make(chan ScanResult),
	}
	m, err := NewMonitor(config)
	if err != nil {
		t.Error("Valid config generated an error -", err.Error())
	}

	m.Start()

	m.Stop()
	m.Stop()
	m.Stop()
	m.Stop()
	m.Stop()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var count int
	for {
		select {
		case <-ticker.C:
			return
		case <-config.Results:
			count++
		}
	}

	if count > 1 {
		t.Error("Monitor produced more than one result after it was stopped multiple times -", count)
	}
}
