package paths

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const appName = "curfew"

type Layout struct {
	Home       string
	ConfigHome string
	DataHome   string
	StateHome  string
}

func Discover() (Layout, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Layout{}, err
	}

	configHome := cleanOrFallback(os.Getenv("XDG_CONFIG_HOME"), filepath.Join(home, ".config"))
	dataHome := cleanOrFallback(os.Getenv("XDG_DATA_HOME"), filepath.Join(home, ".local", "share"))
	stateHome := cleanOrFallback(os.Getenv("XDG_STATE_HOME"), filepath.Join(home, ".local", "state"))

	return Layout{
		Home:       home,
		ConfigHome: configHome,
		DataHome:   dataHome,
		StateHome:  stateHome,
	}, nil
}

func cleanOrFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" || !filepath.IsAbs(value) {
		return fallback
	}
	return filepath.Clean(value)
}

func (l Layout) ConfigDir() string {
	return filepath.Join(l.ConfigHome, appName)
}

func (l Layout) DataDir() string {
	return filepath.Join(l.DataHome, appName)
}

func (l Layout) StateDir() string {
	return filepath.Join(l.StateHome, appName)
}

func (l Layout) ConfigFile() string {
	return filepath.Join(l.ConfigDir(), "config.toml")
}

func (l Layout) HistoryDB() string {
	return filepath.Join(l.DataDir(), "history.db")
}

func (l Layout) RuntimeFile() string {
	return filepath.Join(l.StateDir(), "runtime.json")
}

func (l Layout) RuntimeLockFile() string {
	return filepath.Join(l.StateDir(), "runtime.lock")
}

func (l Layout) Ensure() error {
	for _, dir := range []string{l.ConfigDir(), l.DataDir(), l.StateDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (l Layout) Validate() error {
	if l.Home == "" {
		return errors.New("home directory is not set")
	}
	if !filepath.IsAbs(l.ConfigHome) || !filepath.IsAbs(l.DataHome) || !filepath.IsAbs(l.StateHome) {
		return errors.New("resolved curfew paths must be absolute")
	}
	return nil
}
