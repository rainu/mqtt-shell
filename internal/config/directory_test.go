package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"testing"
)

func Test_getConfigDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	assert.NoError(t, err)

	assert.Equal(t, path.Join(home, ".config", "mqtt-shell"), getConfigDirectory())
}

func Test_getConfigDirectory_legacy(t *testing.T) {
	fakeHome := t.TempDir()

	osUserHomeDir = func() (string, error) {
		return fakeHome, nil
	}
	defer func() {
		osUserHomeDir = os.UserHomeDir
	}()

	assert.NoError(t, os.Mkdir(path.Join(fakeHome, ".mqtt-shell"), 0660))
	assert.Equal(t, path.Join(fakeHome, ".mqtt-shell"), getConfigDirectory())
}
