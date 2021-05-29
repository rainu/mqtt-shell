package config

type Macro struct {
	Description string   `yaml:"description"`
	Arguments   []string `yaml:"arguments,flow"`
	Varargs     bool     `yaml:"varargs"`
	Commands    []string `yaml:"commands,flow"`
}
