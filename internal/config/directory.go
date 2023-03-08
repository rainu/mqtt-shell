package config

import (
	"github.com/kirsle/configdir"
	"os"
	"path"
)

var osUserHomeDir = os.UserHomeDir

var getConfigDirectory = func() string {
	home, err := osUserHomeDir()
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
