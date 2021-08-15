package io

import (
	"bytes"
	"fmt"
	cmdchain "github.com/rainu/go-command-chain"
	"github.com/rainu/mqtt-shell/internal/config"
	"io"
	"strings"
	"text/template"
)

type macroManager struct {
	macroSpecs     map[string]config.Macro
	macroTemplates map[string]*template.Template
	macroFunctions map[string]interface{}
	output         io.Writer
}

func (m *macroManager) ResolveMacro(line string) []string {
	chain, err := interpretLine(line)
	if err != nil {
		return []string{line}
	}

	macroName := chain.Commands[0].Name
	if _, ok := m.macroSpecs[macroName]; !ok {
		m.output.Write([]byte("unknown macro\n"))
		return nil
	}

	macroSpec := m.macroSpecs[macroName]
	if len(chain.Commands[0].Arguments) < len(macroSpec.Arguments) || (!macroSpec.Varargs && len(chain.Commands[0].Arguments) != len(macroSpec.Arguments)) {
		m.output.Write([]byte("invalid macro arguments\n"))
		m.output.Write([]byte("usage: " + macroName + " " + strings.Join(macroSpec.Arguments, " ") + "\n"))
		return nil
	}

	splitLine := strings.SplitN(line, "|", 2)
	pipe := ""
	if len(splitLine) >= 2 {
		pipe = splitLine[1]
	}

	if len(macroSpec.Arguments) == 0 {
		if pipe == "" {
			return macroSpec.Commands
		}

		lines := make([]string, len(macroSpec.Commands))
		for i := 0; i < len(lines); i++ {
			lines[i] = macroSpec.Commands[i]
			if strings.HasPrefix(macroSpec.Commands[i], commandSub+" ") {
				lines[i] += " | " + pipe
			}
		}
		return lines
	}

	if len(macroSpec.Commands) > 0 {
		return m.resolveSimpleMacro(macroSpec, pipe, &chain)
	} else {
		return m.resolveScriptMacro(macroSpec, pipe, &chain)
	}
}

func (m *macroManager) resolveSimpleMacro(macroSpec config.Macro, pipe string, chain *Chain) []string {
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

			if pipe != "" && strings.HasPrefix(line, commandSub+" ") {
				line += " | " + pipe
			}

			lines = append(lines, line)
		}
	}

	return lines
}

func (m *macroManager) resolveScriptMacro(macroSpec config.Macro, pipe string, chain *Chain) []string {
	staticArgs := chain.Commands[0].Arguments[:len(macroSpec.Arguments)-1]
	varArgs := chain.Commands[0].Arguments[len(macroSpec.Arguments)-1:]
	lines := make([]string, 0, len(macroSpec.Commands)*len(varArgs))
	macroName := chain.Commands[0].Name

	tmplData := map[string]string{}

	i := 0
	for ; i < len(staticArgs); i++ {
		tmplData[fmt.Sprintf("Arg%d", i+1)] = staticArgs[i]
	}

	for _, arg := range varArgs {
		buf := bytes.NewBufferString("")
		tmplData[fmt.Sprintf("Arg%d", i+1)] = arg

		if err := m.macroTemplates[macroName].Execute(buf, tmplData); err != nil {
			m.output.Write([]byte(fmt.Sprintf("Error while execute macro script: %s\n", err.Error())))
			continue
		}

		for _, line := range strings.Split(buf.String(), "\n") {
			if pipe != "" && strings.HasPrefix(line, commandSub+" ") {
				line += " | " + pipe
			}
			lines = append(lines, line)
		}
	}

	return lines
}

func (m *macroManager) PrintMacros() {
	for macroName, macroSpec := range m.macroSpecs {
		m.output.Write([]byte(fmt.Sprintf("%s - %s\n", macroName, macroSpec.Description)))
	}
}

func (m *macroManager) ValidateAndInitMacros() error {
	m.macroTemplates = map[string]*template.Template{}
	m.macroFunctions = map[string]interface{}{
		"exec": m.macroFuncExec,
		"log":  m.macroFuncLog,
	}

	for macroName, macroSpec := range m.macroSpecs {
		if !isMacro(macroName) || !isMacro(macroName+" ") {
			//the given macroName is already in use of internal commands
			return fmt.Errorf(`invalid macro name '%s': reserved`, macroName)
		}
		if len(macroSpec.Commands) == 0 && macroSpec.Script == "" {
			return fmt.Errorf(`invalid macro '%s': there is no 'commands' nor 'script'`, macroName)
		} else if len(macroSpec.Commands) > 0 && macroSpec.Script != "" {
			return fmt.Errorf(`invalid macro '%s': only 'commands' or 'script' must be used`, macroName)
		}

		if macroSpec.Script != "" {
			tmpl, err := template.New(macroName).Funcs(m.macroFunctions).Parse(macroSpec.Script)
			if err != nil {
				return fmt.Errorf(`invalid macro '%s': unable to parse script: %w`, macroName, err)
			}
			m.macroTemplates[macroName] = tmpl
		}
	}

	return nil
}

func (m *macroManager) macroFuncExec(line string) string {
	chain, err := interpretLine(line)
	if err != nil {
		panic(err)
	}

	buf := bytes.NewBufferString("")

	var cmdChainBuilder cmdchain.ChainBuilder = cmdchain.Builder()
	for _, command := range chain.Commands {
		cmdChainBuilder = cmdChainBuilder.Join(command.Name, command.Arguments...)
	}

	err = cmdChainBuilder.Finalize().
		WithOutput(buf).
		WithError(buf).
		Run()

	if err != nil {
		panic(err)
	}

	return buf.String()
}

func (m *macroManager) macroFuncLog(format string, args ...interface{}) string {
	output := fmt.Sprintf(format, args...)

	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	m.output.Write([]byte(output))
	return ""
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
