package settings

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Settings struct {
	Configuration           string
	DisableReportDeviceName bool
	DisableCheckUpdate      bool
}

var confFile = func() string {
	ex, _ := os.Executable()
	return filepath.Join(filepath.Dir(ex), "settings.json")
}()

func Load() Settings {
	var s Settings
	b, err := ioutil.ReadFile(confFile)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	return s
}

func FromMap(m map[string]interface{}) Settings {
	var s Settings
	if v, ok := m["configuration"].(string); ok {
		s.Configuration = v
	}
	if v, ok := m["reportDeviceName"].(bool); ok {
		s.DisableReportDeviceName = !v
	}
	if v, ok := m["checkUpdate"].(bool); ok {
		s.DisableCheckUpdate = !v
	}
	return s
}

func (s Settings) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"configuration":    s.Configuration,
		"reportDeviceName": !s.DisableReportDeviceName,
		"checkUpdate":      !s.DisableCheckUpdate,
	}
}

func (s Settings) Save() error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(confFile, b, 0600)
}
