package io

import (
	"errors"
	"github.com/kballard/go-shellquote"
	cmdchain "github.com/rainu/go-command-chain"
	"io"
	"os"
	"os/exec"
	"strings"
)

const (
	linkOut         = "|"
	linkOutAndErr   = "|&"
	linkRedirect    = ">"
	linkRedirectErr = ">&"
	linkAppend      = ">>"
	linkAppendErr   = ">>&"
)

var links = map[string]bool{
	linkOut: true, linkOutAndErr: true,
	linkRedirect: true, linkRedirectErr: true,
	linkAppend: true, linkAppendErr: true,
}

type Command struct {
	Name      string
	Arguments []string
}

type Chain struct {
	Commands []Command
	Links    []string
	RawLine  []string
}

var interpretLine = func(line string) (Chain, error) {
	var err error
	chain := Chain{}
	chain.RawLine, err = shellquote.Split(line)
	if err != nil {
		return chain, err
	}

	cmdParts := [][]string{{}}
	for _, part := range chain.RawLine {
		if links[part] {
			chain.Links = append(chain.Links, part)
			cmdParts = append(cmdParts, []string{})
		} else {
			cmdParts[len(cmdParts)-1] = append(cmdParts[len(cmdParts)-1], part)
		}
	}

	if len(chain.Links) > 1 {
		for i := 0; i < len(chain.Links)-1; i++ {
			if strings.HasPrefix(chain.Links[i], ">") {
				return chain, errors.New("invalid syntax")
			}
		}
	}

	for _, part := range cmdParts {
		if len(part) > 0 {
			cmd := Command{}
			cmd.Name = part[0]

			if len(part) > 1 {
				cmd.Arguments = part[1:]
			}

			chain.Commands = append(chain.Commands, cmd)
		}
	}

	if len(chain.Commands) > 0 && chain.RawLine[len(chain.RawLine)-1] == "&" {
		//remove this from last commands last argument
		lastCommand := chain.Commands[len(chain.Commands)-1]
		lastCommand.Arguments = lastCommand.Arguments[:len(lastCommand.Arguments)-1]

		chain.Commands[len(chain.Commands)-1] = lastCommand
	}

	return chain, nil
}

func (c *Chain) ToCommand(input io.Reader, outputs ...io.Writer) (cmdchain.FinalizedBuilder, func(), error) {
	appending := c.IsAppending()

	b := cmdchain.Builder().WithInput(input)
	to := len(c.Commands)
	if appending {
		//the last "command" is not a command but a output file target
		to--
	}

	for i := 1; i < to; i++ {
		realCmd := exec.Command(c.Commands[i].Name, c.Commands[i].Arguments...)
		realCmd.Env = os.Environ() //pass our env through the new application

		cmd := b.JoinCmd(realCmd)

		//is not last command, check the link to the next command
		if i+1 < len(c.Commands) {
			if c.Links[i] == linkOutAndErr {
				cmd.ForwardError()
			}
		}
		b = cmd
	}

	//callback func will be called after the command is finished
	callbackFn := func() {}
	errOutputs := make([]io.Writer, 0, 1)

	if appending {
		flags := os.O_WRONLY | os.O_CREATE

		if strings.HasPrefix(c.Links[len(c.Links)-1], ">>") {
			flags = flags | os.O_APPEND
		} else {
			flags = flags | os.O_TRUNC
		}

		outFile, err := os.OpenFile(c.Commands[len(c.Commands)-1].Name, flags, 0644)
		if err != nil {
			return nil, callbackFn, err
		}

		if strings.Contains(c.Commands[len(c.Commands)-1].Name, "&") {
			errOutputs = append(errOutputs, outFile)
		}

		//let close the file if execution is finished
		callbackFn = func() {
			outFile.Close()
		}
		outputs = append(outputs, outFile)
	}

	return b.Finalize().WithOutput(outputs...).WithError(errOutputs...), callbackFn, nil
}

func (c *Chain) IsAppending() bool {
	if len(c.Links) > 0 {
		return strings.HasPrefix(c.Links[len(c.Links)-1], ">")
	}
	return false
}

func (c *Chain) IsLongTerm() bool {
	//if the last sign is "&"
	return c.RawLine[len(c.RawLine)-1] == "&"
}
