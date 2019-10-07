package settings

import (
	"encoding/json"

	"github.com/mosteknoloji/inputbox"
	"github.com/shibukawa/configdir"
)

type Settings struct {
	Configuration    string
	ReportDeviceInfo bool
}

// TODO: replace with os.UserConfigDir
func confDir() *configdir.Config {
	return configdir.New("NextDNS", "NextDNS").QueryFolders(configdir.Global)[0]
}

func Load() Settings {
	var s Settings
	b, err := confDir().ReadFile("settings.json")
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	return s
}

func (s Settings) Save() error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return confDir().WriteFile("settings.json", b)
}

func (s Settings) Edit() error {
	// TODO: use https://github.com/lxn/walk
	newConfig, ok := inputbox.InputBox("NextDNS Configuration", "Configuration ID", s.Configuration)
	if ok && newConfig != "" {
		s.Configuration = newConfig
		if err := s.Save(); err != nil {
			return err
		}
	}
	return nil
}
