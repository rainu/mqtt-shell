package io

import (
	"fmt"
	"github.com/chzyer/readline"
	"os"
	"strings"
	"unicode"
)

type shell struct {
	rlInstance *readline.Instance
}

func NewShell(unsubCompletionClb readline.DynamicCompleteFunc) (instance *shell, err error) {
	instance = &shell{}

	qosItem := readline.PcItem("-q",
		readline.PcItem("0"),
		readline.PcItem("1"),
		readline.PcItem("2"),
	)

	instance.rlInstance, err = readline.NewEx(&readline.Config{
		Prompt:      "\033[31m»\033[0m ",
		HistoryFile: "/tmp/readline.tmp",
		AutoComplete: readline.NewPrefixCompleter(
			readline.PcItem("exit"),
			readline.PcItem("list"),
			readline.PcItem("macro"),
			readline.PcItem("pub",
				readline.PcItem("-r",
					qosItem,
				),
				readline.PcItem("-q",
					readline.PcItem("0", readline.PcItem("-r")),
					readline.PcItem("1", readline.PcItem("-r")),
					readline.PcItem("2", readline.PcItem("-r")),
				),
			),
			readline.PcItem("sub", qosItem),
			readline.PcItem("unsub", readline.PcItemDynamic(unsubCompletionClb)),
		),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold: true,
	})
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (s *shell) Start() chan string {
	lineChannel := make(chan string)

	go func() {
		defer close(lineChannel)
		defer s.Close()

		for {
			line, err := s.rlInstance.Readline()
			if err != nil && err != readline.ErrInterrupt {
				return
			}

			//remove non-printable characters to prevent possible strange bugs ;)
			line = strings.Map(func(r rune) rune {
				if unicode.IsGraphic(r) {
					return r
				}
				return -1
			}, line)

			lineChannel <- line
		}
	}()

	return lineChannel
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
