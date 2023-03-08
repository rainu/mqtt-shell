package config

import (
	"github.com/kirsle/configdir"
	"os"
	"path"
)

func getConfigDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "./"
	}

	// legacy directory
	envDir := path.Join(home, ".mqtt-shell")
	if _, err := os.Stat(envDir); err == nil {
		return envDir
	}

	return configdir.LocalConfig("mqtt-shell")
}
