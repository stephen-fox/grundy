package settings

import (
	"io"
	"sync"

	"github.com/go-ini/ini"
)

type configFile interface {
	HasSection(section) bool
	HasKey(section, key) bool
	SectionKeys(section) []string
	SectionValues(section) []string
	SectionKeysToValues(section) map[string]string
	KeyValue(section, key) string
	AddSection(section)
	SetSectionComment(section, string)
	AddValueToSection(section, string)
	AddOrUpdateKeyValue(section, key, string)
	DeleteSection(section)
	DeleteKey(section, key)
	Clear()
	Save(io.Writer) error
	Reload(filePath string) error
}

type iniConfigFile struct {
	configFile
	mutex *sync.Mutex
	ini   *ini.File
}

func (o *iniConfigFile) Reload(filePath string) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	i, err := loadRawIni(filePath)
	if err != nil {
		return err
	}

	o.ini = i

	return nil
}

func (o *iniConfigFile) HasSection(s section) bool {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	_, err := o.ini.GetSection(string(s))
	if err != nil {
		return false
	}

	return true
}

func (o *iniConfigFile) HasKey(s section, k key) bool {
	for _, ke := range o.SectionKeys(s) {
		if ke == string(k) {
			return true
		}
	}

	return false
}

func (o *iniConfigFile) SectionKeys(s section) []string {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	sec, err := o.ini.GetSection(string(s))
	if err != nil {
		return []string{}
	}

	var keys []string

	for _, k := range sec.Keys() {
		keys = append(keys, k.Name())
	}

	return keys
}

func (o *iniConfigFile) SectionValues(s section) []string {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	sec, err := o.ini.GetSection(string(s))
	if err != nil {
		return []string{}
	}

	var values []string

	for _, k := range sec.Keys() {
		values = append(values, k.Value())
	}

	return values
}

func (o *iniConfigFile) SectionKeysToValues(s section) map[string]string {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	m := make(map[string]string)

	sec, err := o.ini.GetSection(string(s))
	if err != nil {
		return m
	}

	for _, k := range sec.Keys() {
		m[k.Name()] = k.Value()
	}

	return m
}

func (o *iniConfigFile) KeyValue(s section, k key) string {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	sec, err := o.ini.GetSection(string(s))
	if err != nil {
		return ""
	}

	if !sec.HasKey(string(k)) {
		return ""
	}

	ke := sec.Key(string(k))
	if ke == nil {
		return ""
	}

	return ke.Value()
}

func (o *iniConfigFile) AddSection(s section) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	_, err := o.ini.NewSection(string(s))
	if err != nil {
		return
	}
}

func (o *iniConfigFile) SetSectionComment(s section, comment string) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	sec, err := o.ini.GetSection(string(s))
	if err != nil {
		return
	}

	sec.Comment = comment
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

	k, err := sec.NewBooleanKey(v)
	if err != nil {
		return
	}

	k.SetValue(v)
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

func (o *iniConfigFile) DeleteKey(s section, k key) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	sec, err := o.ini.GetSection(string(s))
	if err != nil {
		return
	}

	sec.DeleteKey(string(k))
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

func loadIniConfigFile(filePath string) (configFile, error) {
	i, err := loadRawIni(filePath)
	if err != nil {
		return &iniConfigFile{}, err
	}

	return &iniConfigFile{
		mutex: &sync.Mutex{},
		ini:   i,
	}, nil
}

func loadRawIni(filePath string) (*ini.File, error) {
	options := ini.LoadOptions{
		AllowBooleanKeys: true,
	}

	i, err := ini.LoadSources(options, filePath)
	if err != nil {
		return ini.Empty(), err
	}

	return i, nil
}
