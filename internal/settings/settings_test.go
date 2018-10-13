package settings

import (
	"bytes"
	"testing"
)

func TestNewAppSettings(t *testing.T) {
	i := NewAppSettings()

	b := bytes.NewBuffer([]byte{})

	err := i.Save(b)
	if err != nil {
		t.Error(err.Error())
	}

	exp := `[settings]
auto_start = true

[watch_paths]

`
	result := b.String()

	if result != exp {
		t.Error("Result was", result)
	}
}

func TestAppSettingsManipulateValues(t *testing.T) {
	i := NewAppSettings()

	testPath := "/path/to/junk"
	i.AddWatchPath(testPath)

	if !i.HasWatchPath(testPath) {
		t.Error("Missing watch path -", testPath)
	}

	b := bytes.NewBuffer([]byte{})

	err := i.Save(b)
	if err != nil {
		t.Error(err.Error())
	}

	exp := `[settings]
auto_start = true

[watch_paths]
`

	expWithPath := exp + testPath + "\n"
	result := b.String()

	if result != expWithPath {
		t.Error("Result was", result)
	}

	i.RemoveWatchPath(testPath)

	if i.HasWatchPath(testPath) {
		t.Error("Watch path is still present")
	}

	b.Reset()

	err = i.Save(b)
	if err != nil {
		t.Error(err.Error())
	}

	result = b.String()

	if result != exp + "\n" {
		t.Error("Result was", result)
	}
}
