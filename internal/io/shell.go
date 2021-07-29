package io

import (
	"fmt"
	"github.com/rainu/mqtt-shell/internal/config"
	"github.com/rainu/readline"
	"io"
	"os"
	"strings"
	"unicode"
)

type shell struct {
	rlInstance *readline.Instance
	macros     map[string]config.Macro

	targetOut io.Writer
}

func NewShell(prompt, historyFile string,
	macros map[string]config.Macro,
	unsubCompletionClb readline.DynamicCompleteFunc) (instance *shell, err error) {

	if err := validateMacros(macros); err != nil {
		return nil, err
	}

	instance = &shell{
		macros:    macros,
		targetOut: os.Stdout,
	}

	qosItem := readline.PcItem("-q",
		readline.PcItem("0"),
		readline.PcItem("1"),
		readline.PcItem("2"),
	)
	completer := generateMacroCompleter(macros)
	completer = append(completer,
		readline.PcItem(commandListColors),
		readline.PcItem(commandExit),
		readline.PcItem(commandList),
		readline.PcItem(commandPub,
			readline.PcItem("-r",
				qosItem,
			),
			readline.PcItem("-q",
				readline.PcItem("0", readline.PcItem("-r")),
				readline.PcItem("1", readline.PcItem("-r")),
				readline.PcItem("2", readline.PcItem("-r")),
			),
		),
		readline.PcItem(commandSub, qosItem),
		readline.PcItem(commandUnsub, readline.PcItemDynamic(unsubCompletionClb)),
	)

	instance.rlInstance, err = readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     historyFile,
		AutoComplete:    readline.NewPrefixCompleter(completer...),
		InterruptPrompt: "^C",
		EOFPrompt:       commandExit,

		HistorySearchFold: true,
	})
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func validateMacros(macros map[string]config.Macro) error {
	for macroName := range macros {
		if !isMacro(macroName) || !isMacro(macroName+" ") {
			//the given macroName is already in use of internal commands
			return fmt.Errorf(`invalid macro name '%s': reserved`, macroName)
		}
	}

	return nil
}

func generateMacroCompleter(macros map[string]config.Macro) []readline.PrefixCompleterInterface {
	macroItems := make([]readline.PrefixCompleterInterface, len(macros))

	i := 0
	for macroName, macroSpec := range macros {
		macroItems[i] = readline.PcItem(macroName, make([]readline.PrefixCompleterInterface, len(macroSpec.Arguments))...)

		for j, arg := range macroSpec.Arguments {
			macroItems[i].GetChildren()[j] = readline.PcItem(arg)
		}

		i++
	}
	return macroItems
}

func (s *shell) Start() chan string {
	lineChannel := make(chan string)

	go func() {
		defer close(lineChannel)
		defer s.Close()

		for {
			line, err := s.rlInstance.Readline()
			if err != nil {
				if err == readline.ErrInterrupt {
					continue
				}
				return
			}

			//remove non-printable characters to prevent possible strange bugs ;)
			line = strings.Map(func(r rune) rune {
				if unicode.IsGraphic(r) {
					return r
				}
				return -1
			}, line)
			line = strings.TrimSpace(line)

			if line == commandExit {
				break
			}

			if line == commandMacro {
				//if only ".macro" is typed, list all available macros
				for macroName, macroSpec := range s.macros {
					s.Write([]byte(fmt.Sprintf("%s - %s\n", macroName, macroSpec.Description)))
				}
				continue
			}

			lines := []string{line}
			if isMacro(line) {
				lines = s.resolveMacro(line)
			}

			for _, line := range lines {
				lineChannel <- line
			}
		}
	}()

	return lineChannel
}

func isMacro(line string) bool {
	switch {
	case line == commandExit:
		fallthrough
	case line == commandList:
		fallthrough
	case line == commandListColors:
		fallthrough
	case strings.HasPrefix(line, commandPub+" "):
		fallthrough
	case strings.HasPrefix(line, commandSub+" "):
		fallthrough
	case strings.HasPrefix(line, commandUnsub+" "):
		return false
	default:
		return true
	}
}

func (s *shell) resolveMacro(line string) []string {
	chain, err := interpretLine(line)
	if err != nil {
		return []string{line}
	}

	macroName := chain.Commands[0].Name
	if _, ok := s.macros[macroName]; !ok {
		s.Write([]byte("unknown macro\n"))
		return nil
	}

	macroSpec := s.macros[macroName]
	if len(chain.Commands[0].Arguments) < len(macroSpec.Arguments) || (!macroSpec.Varargs && len(chain.Commands[0].Arguments) != len(macroSpec.Arguments)) {
		s.Write([]byte("invalid macro arguments\n"))
		s.Write([]byte("usage: " + macroName + " " + strings.Join(macroSpec.Arguments, " ") + "\n"))
		return nil
	}

	if len(macroSpec.Arguments) == 0 {
		return macroSpec.Commands
	}

	staticArgs := chain.Commands[0].Arguments[:len(macroSpec.Arguments)-1]
	varArgs := chain.Commands[0].Arguments[len(macroSpec.Arguments)-1:]
	lines := make([]string, 0, len(macroSpec.Commands)*len(varArgs))

	for _, arg := range varArgs {
		for _, macroCommand := range macroSpec.Commands {
			line := strings.Replace(macroCommand, "\\$", "__DOLLAR_ESCAPE__", -1)

			i := 0
			for ; i < len(staticArgs); i++ {
				line = strings.Replace(line, fmt.Sprintf("$%d", i+1), staticArgs[i], -1)
			}
			line = strings.Replace(line, fmt.Sprintf("$%d", i+1), arg, -1)
			line = strings.Replace(line, "__DOLLAR_ESCAPE__", "$", -1)

			lines = append(lines, line)
		}
	}

	return lines
}

func (s *shell) Close() error {
	return s.rlInstance.Close()
}

func (s *shell) Write(b []byte) (n int, err error) {
	defer func() {
		if s.rlInstance != nil {
			s.rlInstance.Refresh()
		}
	}()

	n, err = fmt.Fprintf(s.targetOut, "\r\033[2K%s", string(b))
	if err == nil {
		//in happy case we have to make sure that the correct amount of read bytes
		//are returned -> otherwise this will cause many io trouble
		n = len(b)
	}

	return
}
