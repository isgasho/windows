package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var currentVersion = func() string {
	ex, _ := os.Executable()
	file := filepath.Join(filepath.Dir(ex), "version.txt")
	b, _ := ioutil.ReadFile(file)
	return string(b)
}()

func CurrentVersion() string {
	return currentVersion
}

type Updater struct {
	URL string

	OnUpgrade func(newVersion string)

	// ErrorLog specifies an optional log function for errors. If not set,
	// errors are not reported.
	ErrorLog func(error)

	// Channel is the channel to use for updates.
	Channel string

	mu   sync.Mutex
	stop func()
}

type info struct {
	Version string
	URL     string
}

func (u *Updater) SetAutoRun(enabled bool) {
	if currentVersion == "" {
		// Updater disabled
		return
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.stop == nil && enabled {
		go u.run()
	} else if u.stop != nil && !enabled {
		u.stop()
		u.stop = nil
	}
}

func (u *Updater) run() {
	var ctx context.Context
	ctx, u.stop = context.WithCancel(context.Background())
	t := time.NewTicker(24 * time.Hour)
	if err := u.CheckNow(); err != nil {
		u.logErr(err)
	}
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := u.CheckNow(); err != nil {
				u.logErr(err)
			}
		}
	}
}

func (u *Updater) CheckNow() error {
	if currentVersion == "" {
		// Updater disabled
		return nil
	}
	res, err := http.Get(u.URL)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	dec := json.NewDecoder(res.Body)
	var i map[string]info
	if err := dec.Decode(&i); err != nil {
		return err
	}
	channelName := strings.ToLower(u.Channel)
	if channelName == "" {
		channelName = "stable"
	}
	channel, found := i[channelName]
	if !found {
		return errors.New("stable version info not found")
	}
	if channel.Version != currentVersion {
		// Already on last version
		if u.OnUpgrade != nil {
			u.OnUpgrade(channel.Version)
		}
		return u.upgrade(channel.URL, channel.Version)
	}
	return nil
}

func (u *Updater) upgrade(url, version string) error {
	installerPath, err := u.downloadInstaller(url, version)
	if err != nil {
		return err
	}
	cmd := exec.Command(installerPath, "/S")
	if err := cmd.Run(); err != nil {
		_ = os.Remove(installerPath)
	}
	return err
}

func (u *Updater) downloadInstaller(url, version string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	installPath := filepath.Join(os.TempDir(), fmt.Sprintf("NextDNS Upgrader %s.exe", version))
	if st, err := os.Stat(installPath); err == nil && time.Since(st.ModTime()) < 24*time.Hour {
		// We already have the installer for this version in the tmp directory,
		// do not re-download it.
		return installPath, nil
	}
	os.Remove(installPath)
	f, err := os.OpenFile(installPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = io.Copy(f, res.Body)
	if err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return installPath, nil
}

func (u *Updater) logErr(err error) {
	if u.ErrorLog != nil {
		u.ErrorLog(fmt.Errorf("updater: %v", err))
	}
}
