package settings

import (
	"io"
	"sync"

	"github.com/go-ini/ini"
)

type configFile interface {
	AddSection(section)
	AddValueToSection(section, string)
	AddOrUpdateKeyValue(section, key, string)
	DeleteSection(section)
	Clear()
	Save(io.Writer) error
}

type iniConfigFile struct {
	configFile
	mutex *sync.Mutex
	ini   *ini.File
}

func (o *iniConfigFile) AddSection(s section) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	_, err := o.ini.NewSection(string(s))
	if err != nil {
		return
	}
}

func (o *iniConfigFile) AddValueToSection(s section, v string) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	var err error

	sec := o.ini.Section(string(s))
	if sec == nil {
		sec, err = o.ini.NewSection(string(s))
		if err != nil {
			return
		}
	}

	_, err = sec.NewBooleanKey(v)
	if err != nil {
		return
	}
}

func (o *iniConfigFile) AddOrUpdateKeyValue(s section, k key, v string) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	var err error

	sec, err := o.ini.GetSection(string(s))
	if err != nil {
		sec, err = o.ini.NewSection(string(s))
		if err != nil {
			return
		}
	}

	key := sec.Key(string(k))
	if key == nil {
		key, err = sec.NewKey(string(k), v)
		if err != nil {
			return
		}
	}

	key.SetValue(v)
}

func (o *iniConfigFile) DeleteSection(s section) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.ini.DeleteSection(string(s))
}

func (o *iniConfigFile) Clear() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.ini = ini.Empty()
}

func (o *iniConfigFile) Save(w io.Writer) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	_, err := o.ini.WriteTo(w)
	if err != nil {
		return err
	}

	return nil
}

func newEmptyIniFile() configFile {
	return &iniConfigFile{
		mutex: &sync.Mutex{},
		ini:   ini.Empty(),
	}
}

func loadIniFile(filePath string) (configFile, error) {
	options := ini.LoadOptions{
		AllowBooleanKeys: true,
	}

	i, err := ini.LoadSources(options, filePath)
	if err != nil {
		return &iniConfigFile{}, err
	}

	return &iniConfigFile{
		mutex: &sync.Mutex{},
		ini:   i,
	}, nil
}
