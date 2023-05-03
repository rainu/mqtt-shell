package io

import (
	"fmt"
	"github.com/rainu/mqtt-shell/internal/config"
	"github.com/rainu/readline"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode"
)

var multilineRegex = regexp.MustCompile(`<<([a-zA-Z0-9]+)$`)

type shell struct {
	rlInstance   *readline.Instance
	macroManager *MacroManager

	targetOut io.Writer

	// wrap function for monkey patching purposes (unit tests)
	readline func() (string, error)
}

func NewShell(prompt, historyFile string,
	macroManager *MacroManager,
	unsubCompletionClb readline.DynamicCompleteFunc) (instance *shell, err error) {

	instance = &shell{
		macroManager: macroManager,
		targetOut:    os.Stdout,
	}

	qosItem := readline.PcItem("-q",
		readline.PcItem("0"),
		readline.PcItem("1"),
		readline.PcItem("2"),
	)
	completer := generateMacroCompleter(macroManager.MacroSpecs)
	completer = append(completer,
		readline.PcItem(commandListColors),
		readline.PcItem(commandMacro),
		readline.PcItem(commandExit),
		readline.PcItem(commandHelp),
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

	instance.readline = instance.rlInstance.Readline

	return instance, nil
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
			line, err := s.readline()
			if err != nil {
				if err == readline.ErrInterrupt {
					continue
				}
				break
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

			if multilineRegex.MatchString(line) {
				line = s.readMultilines(line)
			}

			if line == "" {
				continue
			}

			if line == commandMacro {
				//if only ".macro" is typed, list all available macroSpecs
				s.macroManager.PrintMacros()
				continue
			}

			lines := []string{line}
			if s.macroManager.IsMacro(line) {
				lines = s.macroManager.ResolveMacro(line)
			}

			for _, line := range lines {
				lineChannel <- line
			}
		}
	}()

	return lineChannel
}

func (s *shell) readMultilines(line string) string {
	sb := strings.Builder{}
	sb.WriteString(line)

	eofWord := multilineRegex.FindStringSubmatch(line)[1]

	//do not safe the lines hin history
	s.rlInstance.HistoryDisable()
	defer s.rlInstance.HistoryEnable()

	for {
		newLine, err := s.readline()
		if err != nil {
			return "" //if user cancel the read
		}

		sb.WriteString("\n")
		sb.WriteString(newLine)

		if strings.HasSuffix(newLine, eofWord) {
			break
		}
	}

	return sb.String()
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

	_, err = fmt.Fprintf(s.targetOut, "\r\033[2K%s", string(b))
	if err == nil {
		//in happy case we have to make sure that the correct amount of read bytes
		//are returned -> otherwise this will cause many io trouble
		n = len(b)
	}

	return
}
