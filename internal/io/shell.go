package io

import (
	"fmt"
	"github.com/rainu/mqtt-shell/internal/config"
	"github.com/rainu/readline"
	"os"
	"strings"
	"unicode"
)

type shell struct {
	rlInstance *readline.Instance
	macros     map[string]config.Macro
}

func NewShell(prompt, historyFile string,
	macros map[string]config.Macro,
	unsubCompletionClb readline.DynamicCompleteFunc) (instance *shell, err error) {

	instance = &shell{
		macros: macros,
	}

	qosItem := readline.PcItem("-q",
		readline.PcItem("0"),
		readline.PcItem("1"),
		readline.PcItem("2"),
	)
	macroItem := generateMacroCompleter(macros)

	instance.rlInstance, err = readline.NewEx(&readline.Config{
		Prompt:      prompt,
		HistoryFile: historyFile,
		AutoComplete: readline.NewPrefixCompleter(
			readline.PcItem(commandListColors),
			readline.PcItem(commandExit),
			readline.PcItem(commandList),
			macroItem,
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
		),
		InterruptPrompt: "^C",
		EOFPrompt:       commandExit,

		HistorySearchFold: true,
	})
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func generateMacroCompleter(macros map[string]config.Macro) *readline.PrefixCompleter {
	macroItem := readline.PcItem(commandMacro, make([]readline.PrefixCompleterInterface, len(macros))...)

	i := 0
	for macroName, macroSpec := range macros {
		macroItem.GetChildren()[i] = readline.PcItem(macroName, make([]readline.PrefixCompleterInterface, len(macroSpec.Arguments))...)

		for j, arg := range macroSpec.Arguments {
			macroItem.GetChildren()[i].GetChildren()[j] = readline.PcItem(arg)
		}

		i++
	}
	return macroItem
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

			lines := []string{line}
			if strings.HasPrefix(line, commandMacro) {
				lines = s.resolveMacro(line)
			}

			for _, line := range lines {
				lineChannel <- line
			}
		}
	}()

	return lineChannel
}

func (s *shell) resolveMacro(line string) []string {
	chain, err := interpretLine(line)
	if err != nil {
		return []string{line}
	}

	if chain.Commands[0].Name != commandMacro {
		return []string{line}
	}

	if len(chain.Commands[0].Arguments) == 0 {
		//if only "macro" is typed, list all available macros
		for macroName, macroSpec := range s.macros {
			s.Write([]byte(fmt.Sprintf("%s - %s\n", macroName, macroSpec.Description)))
		}
		return nil
	}

	macroName := chain.Commands[0].Arguments[0]
	if _, ok := s.macros[macroName]; !ok {
		s.Write([]byte("unknown macro\n"))
		return nil
	}

	macroSpec := s.macros[macroName]
	if len(chain.Commands[0].Arguments)-1 < len(macroSpec.Arguments) || (!macroSpec.Varargs && len(chain.Commands[0].Arguments)-1 != len(macroSpec.Arguments)) {
		s.Write([]byte("invalid macro arguments\n"))
		s.Write([]byte("usage: " + macroName + " " + strings.Join(macroSpec.Arguments, " ") + "\n"))
		return nil
	}

	if len(macroSpec.Arguments) == 0 {
		return macroSpec.Commands
	}

	staticArgs := chain.Commands[0].Arguments[1:len(macroSpec.Arguments)]
	varArgs := chain.Commands[0].Arguments[len(macroSpec.Arguments):]
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

	return fmt.Fprintf(os.Stdout, "\r\033[2K%s", string(b))
}
