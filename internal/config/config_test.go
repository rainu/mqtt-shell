package config

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"strings"
	"testing"
)

func TestReadConfig_moreHelp(t *testing.T) {
	resetFlags()

	origGetConfigDirectory := getConfigDirectory
	defer func() {
		getConfigDirectory = origGetConfigDirectory
	}()
	getConfigDirectory = func() string {
		return t.TempDir()
	}

	os.Args = []string{"mqtt-shell", "-hh"}
	os.Stderr, _ = os.OpenFile(path.Join(t.TempDir(), "stderr"), os.O_RDWR|os.O_CREATE, 0755)

	result, rc := ReadConfig("<version>", "<revision>")

	assert.Nil(t, result)
	assert.Equal(t, 1, rc)

	content, err := os.ReadFile(os.Stderr.Name())
	assert.NoError(t, err)
	assert.Equal(t, helpText, string(content))
}

func TestReadConfig_showVersion(t *testing.T) {
	resetFlags()

	origGetConfigDirectory := getConfigDirectory
	defer func() {
		getConfigDirectory = origGetConfigDirectory
	}()
	getConfigDirectory = func() string {
		return t.TempDir()
	}

	os.Args = []string{"mqtt-shell", "-v"}
	os.Stdout, _ = os.OpenFile(path.Join(t.TempDir(), "stdout"), os.O_RDWR|os.O_CREATE, 0755)

	result, rc := ReadConfig("<version>", "<revision>")

	assert.Nil(t, result)
	assert.Equal(t, 0, rc)

	content, err := os.ReadFile(os.Stdout.Name())
	assert.NoError(t, err)
	assert.Equal(t, "<version> - <revision>\n", string(content))
}

func TestReadConfig_brokerIsMandatory(t *testing.T) {
	resetFlags()

	origGetConfigDirectory := getConfigDirectory
	defer func() {
		getConfigDirectory = origGetConfigDirectory
	}()
	getConfigDirectory = func() string {
		return t.TempDir()
	}

	os.Args = []string{"mqtt-shell"}
	os.Stderr, _ = os.OpenFile(path.Join(t.TempDir(), "stderr"), os.O_RDWR|os.O_CREATE, 0755)

	result, rc := ReadConfig("<version>", "<revision>")

	assert.Nil(t, result)
	assert.Equal(t, 1, rc)

	content, err := os.ReadFile(os.Stderr.Name())
	assert.NoError(t, err)
	assert.Equal(t, "Broker is missing!", string(content))
}

func TestReadConfig_defaultValues(t *testing.T) {
	resetFlags()

	origGetConfigDirectory := getConfigDirectory
	defer func() {
		getConfigDirectory = origGetConfigDirectory
	}()
	cfgDir := t.TempDir()
	getConfigDirectory = func() string {
		return cfgDir
	}

	os.Args = []string{"mqtt-shell", "-b", "tcp://127.0.0.1:1883"}
	result, rc := ReadConfig("<version>", "<revision>")

	assert.Equal(t, -1, rc)
	assert.Equal(t, Config{
		Broker:         "tcp://127.0.0.1:1883",
		CaFile:         "",
		SubscribeQOS:   0,
		PublishQOS:     1,
		Username:       "",
		Password:       "",
		ClientId:       "mqtt-shell",
		CleanSession:   true,
		StartCommands:  nil,
		NonInteractive: false,
		HistoryFile:    path.Join(cfgDir, ".history"),
		Prompt:         "\x1b[36m»\x1b[0m ",
		Macros:         nil,
		ColorBlacklist: nil,
	}, *result)
}

func TestReadConfig_readGlobal(t *testing.T) {
	testReadConfig_readGlobal(t, ".global.yml")
}

func TestReadConfig_readGlobal2(t *testing.T) {
	testReadConfig_readGlobal(t, ".global.yaml")
}

func testReadConfig_readGlobal(t *testing.T, fileName string) {
	resetFlags()

	origGetConfigDirectory := getConfigDirectory
	defer func() {
		getConfigDirectory = origGetConfigDirectory
	}()
	cfgDir := t.TempDir()
	getConfigDirectory = func() string {
		return cfgDir
	}

	err := os.WriteFile(path.Join(cfgDir, fileName), []byte(strings.ReplaceAll(strings.TrimSpace(`
broker: tcp://127.0.0.1:1883
ca: /tmp/ca.pam
subscribe-qos: 1
publish-qos: 2
username: rainu
password: secret
client-id: rainu-shell
clean-session: false
commands:
	- help
non-interactive: true
history-file: /tmp/history
prompt: =>
macros:
	test:
		description: some test
		arguments:
			- argN
		varargs: true
		commands:
			- help
		script: its a script
color-blacklist:
	- "00,11,22"
	`), "\t", "  ")), 0755)
	assert.Nil(t, err)

	os.Args = []string{"mqtt-shell"}
	result, rc := ReadConfig("<version>", "<revision>")

	assert.Equal(t, -1, rc)
	assert.Equal(t, Config{
		Broker:         "tcp://127.0.0.1:1883",
		CaFile:         "/tmp/ca.pam",
		SubscribeQOS:   1,
		PublishQOS:     2,
		Username:       "rainu",
		Password:       "secret",
		ClientId:       "rainu-shell",
		CleanSession:   false,
		StartCommands:  []string{"help"},
		NonInteractive: true,
		HistoryFile:    "/tmp/history",
		Prompt:         "=>",
		Macros: map[string]Macro{
			"test": {
				Description: "some test",
				Arguments:   []string{"argN"},
				Varargs:     true,
				Commands:    []string{"help"},
				Script:      "its a script",
			},
		},
		ColorBlacklist: []string{"00,11,22"},
	}, *result)
}

func TestReadConfig_argsOverridesConfigFiles(t *testing.T) {
	resetFlags()

	origGetConfigDirectory := getConfigDirectory
	defer func() {
		getConfigDirectory = origGetConfigDirectory
	}()
	cfgDir := t.TempDir()
	getConfigDirectory = func() string {
		return cfgDir
	}

	err := os.WriteFile(path.Join(cfgDir, ".global.yml"), []byte(strings.ReplaceAll(strings.TrimSpace(`
broker: tcp://127.0.0.1:1883
ca: /tmp/ca.pam
subscribe-qos: 1
publish-qos: 2
username: rainu
password: secret
client-id: rainu-shell
clean-session: false
commands:
	- help
non-interactive: false
history-file: /tmp/history
prompt: =>
color-blacklist:
	- "00,11,22"
	`), "\t", "  ")), 0755)
	assert.Nil(t, err)

	os.Args = []string{
		"mqtt-shell",
		"-b", "tcp://8.8.8.8:1883",
		"-ca", "/home/ca.pam",
		"-sq", "2",
		"-pq", "1",
		"-u", "admin",
		"-p", "password",
		"-c", "test-shell",
		"-cs",
		"-cmd", "test",
		"-ni",
		"-hf", "/home/history",
		"-sp", "$>",
		"-cb", "13,12,89",
	}
	result, rc := ReadConfig("<version>", "<revision>")

	assert.Equal(t, -1, rc)
	assert.Equal(t, Config{
		Broker:         "tcp://8.8.8.8:1883",
		CaFile:         "/home/ca.pam",
		SubscribeQOS:   2,
		PublishQOS:     1,
		Username:       "admin",
		Password:       "password",
		ClientId:       "test-shell",
		CleanSession:   true,
		StartCommands:  []string{"test"},
		NonInteractive: true,
		HistoryFile:    "/home/history",
		Prompt:         "$>",
		Macros:         nil,
		ColorBlacklist: []string{"13,12,89"},
	}, *result)
}

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet("mqtt-shell-test", flag.ExitOnError)
	flag.CommandLine.Usage = flag.Usage
}
