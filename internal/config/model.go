package config

import "fmt"

type Config struct {
	Broker       string `yaml:"broker"`
	CaFile       string `yaml:"ca"`
	SubscribeQOS int    `yaml:"subscribe-qos"`
	PublishQOS   int    `yaml:"publish-qos"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	ClientId     string `yaml:"client-id"`
	CleanSession bool   `yaml:"clean-session"`

	StartCommands  []string         `yaml:"commands"`
	NonInteractive bool             `yaml:"non-interactive"`
	HistoryFile    string           `yaml:"history-file"`
	Prompt         string           `yaml:"prompt"`
	Macros         map[string]Macro `yaml:"macros"`
	ColorBlacklist []string         `yaml:"color-blacklist"`
}

type Macro struct {
	Description string   `yaml:"description"`
	Arguments   []string `yaml:"arguments,flow"`
	Varargs     bool     `yaml:"varargs"`
	Commands    []string `yaml:"commands,flow"`
	Script      string   `yaml:"script"`
}

type varArgs []string

func (i *varArgs) String() string {
	return fmt.Sprintf("%v", []string(*i))
}

func (i *varArgs) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *varArgs) Reset() {
	*i = []string{}
}
