package config

import (
	"flag"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"path"
	"strconv"
)

type Config struct {
	Broker       string `yaml:"broker"`
	CaFile       string `yaml:"ca"`
	SubscribeQOS int    `yaml:"subscribe-qos"`
	PublishQOS   int    `yaml:"publish-qos"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	ClientId     string `yaml:"client-id"`
	CleanSession bool   `yaml:"clean-session"`

	StartCommands  []string `yaml:"commands"`
	NonInteractive bool     `yaml:"non-interactive"`
	HistoryFile    string   `yaml:"history-file"`
	Prompt         string   `yaml:"prompt"`
}

func NewConfig() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "./"
	}

	cfg := Config{}

	env := ""
	envDir := path.Join(home, ".mqtt-shell")

	flag.StringVar(&env, "e", "", "The environment which should be used")
	flag.StringVar(&envDir, "ed", envDir, "The environment directory")
	flag.StringVar(&cfg.Broker, "b", "", "The broker URI. ex: tcp://127.0.0.1:1883")
	flag.StringVar(&cfg.CaFile, "ca", "", "MQTT ca file path (if tls is used)")
	flag.IntVar(&cfg.SubscribeQOS, "sq", 0, "The default Quality of Service for subscription 0,1,2")
	flag.IntVar(&cfg.PublishQOS, "pq", 1, "The default Quality of Service for publishing 0,1,2")
	flag.StringVar(&cfg.Username, "u", "", "The username")
	flag.StringVar(&cfg.Password, "p", "", "The password")
	flag.StringVar(&cfg.ClientId, "c", "mqtt-shell", "The ClientID")
	flag.BoolVar(&cfg.CleanSession, "cs", true, "Indicating that no messages saved by the broker for this client should be delivered")
	flag.BoolVar(&cfg.NonInteractive, "ni", false, "Should this shell be non interactive. Only useful in combination with 'cmd' option")
	flag.StringVar(&cfg.HistoryFile, "hf", path.Join(home, ".mqtt-shell", ".history"), "The history file path")
	flag.StringVar(&cfg.Prompt, "sp", `\033[36m»\033[0m `, "The prompt of the shell")

	var startCommands varArgs
	flag.Var(&startCommands, "cmd", "The command(s) which should be executed at the beginning")
	flag.Parse()

	cfg.StartCommands = startCommands

	if env != "" {
		var suffix string
		if _, err := os.Stat(path.Join(envDir, env+".yaml")); os.IsNotExist(err) {
			if _, err := os.Stat(path.Join(envDir, env+".yml")); os.IsNotExist(err) {
				log.Fatal("No environment file found")
			} else {
				suffix = ".yml"
			}
		} else {
			suffix = ".yaml"
		}

		envFile, err := os.Open(path.Join(envDir, env+suffix))
		if err != nil {
			log.Fatal("Can not open environment file: ", err)
		}

		if err := yaml.NewDecoder(envFile).Decode(&cfg); err != nil {
			log.Fatal("Unable to parse environment file: ", err)
		}
	}

	if cfg.Broker == "" {
		log.Fatal("Broker is missing!")
	}

	cfg.Prompt, err = strconv.Unquote(`"` + cfg.Prompt + `"`)
	if err != nil {
		log.Fatal("Unable to parse prompt: ", err)
	}

	return cfg
}
