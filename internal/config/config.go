package config

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"path"
	"strconv"
)

func ReadConfig(version, revision string) (*Config, int) {
	cfg := Config{}

	env := ""
	envDir := getConfigDirectory()

	var moreHelp, showVersion bool
	flag.BoolVar(&moreHelp, "hh", false, "Show detailed help text")
	flag.BoolVar(&showVersion, "v", false, "Show the version")

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
	flag.StringVar(&cfg.HistoryFile, "hf", path.Join(envDir, ".history"), "The history file path")
	flag.StringVar(&cfg.Prompt, "sp", `\033[36m»\033[0m `, "The prompt of the shell")

	var startCommands, macroFiles, colorBlacklist varArgs
	macroFiles.Set(path.Join(envDir, ".macros.yml"))

	flag.Var(&startCommands, "cmd", "The command(s) which should be executed at the beginning")
	flag.Var(&macroFiles, "m", "The macro file(s) which should be loaded")
	flag.Var(&colorBlacklist, "cb", "This color(s) will not be used")
	flag.Parse()

	if moreHelp {
		fmt.Fprint(os.Stderr, helpText)
		return nil, 1
	}

	if showVersion {
		fmt.Printf("%s - %s\n", version, revision)
		return nil, 0
	}

	if _, err := os.Stat(path.Join(envDir, ".global.yml")); err == nil {
		handleFile(envDir, ".global", &cfg)
	}
	if _, err := os.Stat(path.Join(envDir, ".global.yaml")); err == nil {
		handleFile(envDir, ".global", &cfg)
	}

	if env != "" {
		handleFile(envDir, env, &cfg)
	}

	// overwrite potential config values with argument values
	startCommands.Reset()
	colorBlacklist.Reset()
	flag.Parse()
	if len(startCommands) > 0 {
		cfg.StartCommands = startCommands
	}
	if len(colorBlacklist) > 0 {
		cfg.ColorBlacklist = colorBlacklist
	}

	if cfg.Broker == "" {
		fmt.Fprint(os.Stderr, "Broker is missing!")
		return nil, 1
	}
	loadMacroFiles(&cfg, macroFiles)

	return &cfg, -1
}

func handleFile(envDir, env string, cfg *Config) {
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
	defer envFile.Close()

	if err := yaml.NewDecoder(envFile).Decode(&cfg); err != nil {
		log.Fatal(fmt.Sprintf("Unable to parse environment file (%s): ", envFile.Name()), err)
	}
}

func loadMacroFiles(cfg *Config, macroFiles varArgs) {
	var err error

	cfg.Prompt, err = strconv.Unquote(`"` + cfg.Prompt + `"`)
	if err != nil {
		log.Fatal("Unable to parse prompt: ", err)
	}

	for _, filePath := range macroFiles {
		loadMacroFile(cfg, filePath)
	}
}

func loadMacroFile(cfg *Config, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			//skip this file
			return
		}

		log.Fatal("Can not open macro file: ", err)
	}
	defer file.Close()

	macros := map[string]Macro{}
	if err := yaml.NewDecoder(file).Decode(&macros); err != nil {
		log.Fatal(fmt.Sprintf("Unable to parse macro file '%s': ", filePath), err)
	}

	//merge macros
	for macroName, macroSpec := range macros {
		cfg.Macros[macroName] = macroSpec
	}
}
