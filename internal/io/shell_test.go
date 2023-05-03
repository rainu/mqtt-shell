package io

import (
	"bytes"
	"errors"
	"github.com/rainu/mqtt-shell/internal/config"
	"github.com/rainu/readline"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"strings"
	"testing"
)

func TestShell_Write(t *testing.T) {
	sSource := `some test text`
	source := strings.NewReader(sSource)
	target := bytes.NewBuffer([]byte{})

	toTest := &shell{
		targetOut: target,
	}

	n, err := io.Copy(toTest, source)

	assert.NoError(t, err)
	assert.EqualValues(t, len(sSource), n)
	assert.Equal(t, "\r\x1b[2K"+sSource, target.String())
}

func TestNewShell(t *testing.T) {
	macroManager := &MacroManager{
		MacroSpecs: map[string]config.Macro{
			"test": {
				Arguments: []string{"arg1", "arg2"},
			},
		},
	}
	unsubClb := func(string) []string {
		return []string{"a/topic", "b/topic/#"}
	}

	toTest, err := NewShell("PROMPT>", "/tmp/history", macroManager, unsubClb)

	assert.NoError(t, err)

	assert.Same(t, os.Stdout, toTest.targetOut)
	assert.Same(t, macroManager, toTest.macroManager)
	assert.Equal(t, "PROMPT>", toTest.rlInstance.Config.Prompt)
	assert.Equal(t, commandExit, toTest.rlInstance.Config.EOFPrompt)
	assert.Equal(t, "^C", toTest.rlInstance.Config.InterruptPrompt)
	assert.Equal(t, "/tmp/history", toTest.rlInstance.Config.HistoryFile)
	assert.Equal(t, true, toTest.rlInstance.Config.HistorySearchFold)

	suggestions, _ := toTest.rlInstance.Config.AutoComplete.Do([]rune(""), 0)
	assert.Equal(t, []string{
		"test ", //macro
		commandListColors + " ",
		commandMacro + " ",
		commandExit + " ",
		commandHelp + " ",
		commandList + " ",
		commandPub + " ",
		commandSub + " ",
		commandUnsub + " ",
	}, rc(suggestions), "the default commands and macros should be suggested")

	suggestions, _ = toTest.rlInstance.Config.AutoComplete.Do([]rune("test "), 5)
	assert.Equal(t, []string{
		"arg1 ", //macro argument #1
		"arg2 ", //macro argument #2
	}, rc(suggestions), "the macro arguments should be suggested")

	suggestions, _ = toTest.rlInstance.Config.AutoComplete.Do([]rune(commandPub+" "), len(commandPub)+1)
	assert.Equal(t, []string{"-r ", "-q "}, rc(suggestions), "the pub arguments should be suggested")

	suggestions, _ = toTest.rlInstance.Config.AutoComplete.Do([]rune(commandPub+" -q "), len(commandPub)+4)
	assert.Equal(t, []string{"0 ", "1 ", "2 "}, rc(suggestions), "the qos levels should be suggested")

	suggestions, _ = toTest.rlInstance.Config.AutoComplete.Do([]rune(commandPub+" -q 0 "), len(commandPub)+6)
	assert.Equal(t, []string{"-r "}, rc(suggestions), "the retained flag should be suggested")

	suggestions, _ = toTest.rlInstance.Config.AutoComplete.Do([]rune(commandPub+" -r "), len(commandPub)+4)
	assert.Equal(t, []string{"-q "}, rc(suggestions), "the qos flag should be suggested")

	suggestions, _ = toTest.rlInstance.Config.AutoComplete.Do([]rune(commandSub+" "), len(commandSub)+1)
	assert.Equal(t, []string{"-q "}, rc(suggestions), "the qos flag should be suggested")

	suggestions, _ = toTest.rlInstance.Config.AutoComplete.Do([]rune(commandSub+" -q "), len(commandSub)+4)
	assert.Equal(t, []string{"0 ", "1 ", "2 "}, rc(suggestions), "the qos levels should be suggested")

	suggestions, _ = toTest.rlInstance.Config.AutoComplete.Do([]rune(commandUnsub+" "), len(commandUnsub)+1)
	assert.Equal(t, []string{"a/topic ", "b/topic/# "}, rc(suggestions), "the already subscribed topics should be suggested")

}

func rc(in [][]rune) []string {
	outer := make([]string, len(in))

	for i, runes := range in {
		outer[i] = string(runes)
	}

	return outer
}

func TestShell_Start_normalLine(t *testing.T) {
	toTest, _ := NewShell("PROMPT>", "/tmp/history", &MacroManager{}, nil)
	toTest.readline = func() (string, error) {
		return "sub a/topic", nil
	}

	lines := toTest.Start()

	assert.Equal(t, "sub a/topic", <-lines)

	_, ok := <-lines
	assert.True(t, ok)
}

func TestShell_Start_exit(t *testing.T) {
	toTest, _ := NewShell("PROMPT>", "/tmp/history", &MacroManager{}, nil)
	toTest.readline = func() (string, error) {
		return commandExit, nil
	}

	lines := toTest.Start()

	l, ok := <-lines
	assert.Equal(t, "", l)
	assert.False(t, ok)
}

func TestShell_Start_interrupt(t *testing.T) {
	toTest, _ := NewShell("PROMPT>", "/tmp/history", &MacroManager{}, nil)

	firstCall := true
	toTest.readline = func() (string, error) {
		if firstCall {
			firstCall = false
			return "", readline.ErrInterrupt
		}
		return commandList, nil
	}

	lines := toTest.Start()

	assert.Equal(t, commandList, <-lines)
}

func TestShell_Start_eof(t *testing.T) {
	toTest, _ := NewShell("PROMPT>", "/tmp/history", &MacroManager{}, nil)

	firstCall := true
	toTest.readline = func() (string, error) {
		if firstCall {
			firstCall = false
			return "", errors.New("someError")
		}
		return commandList, nil
	}

	lines := toTest.Start()

	_, ok := <-lines
	assert.False(t, ok)
}

func TestShell_Start_purging(t *testing.T) {
	toTest, _ := NewShell("PROMPT>", "/tmp/history", &MacroManager{}, nil)
	toTest.readline = func() (string, error) {
		return "\x00   " + commandList + "      ", nil
	}

	lines := toTest.Start()

	assert.Equal(t, commandList, <-lines, "the escape characters should be removed and the line should be trimmed")

	_, ok := <-lines
	assert.True(t, ok)
}

func TestShell_Start_multiline(t *testing.T) {
	toTest, _ := NewShell("PROMPT>", "/tmp/history", &MacroManager{}, nil)

	inputs := []string{
		"pub a/topic <<EOF",
		"  Multiline  ",
		"String which ends nowEOF",
	}
	curLine := 0
	toTest.readline = func() (string, error) {
		if curLine >= len(inputs) {
			return commandList, nil
		}

		l := inputs[curLine]
		curLine++

		return l, nil
	}

	lines := toTest.Start()

	assert.Equal(t, "pub a/topic <<EOF\n  Multiline  \nString which ends nowEOF", <-lines)

	_, ok := <-lines
	assert.True(t, ok)
}

func TestShell_Start_printMacros(t *testing.T) {
	output := &bytes.Buffer{}
	macroManager := &MacroManager{
		MacroSpecs: map[string]config.Macro{
			"Test": {
				Description: "test macro description",
			},
		},
		Output: output,
	}

	toTest, _ := NewShell("PROMPT>", "/tmp/history", macroManager, nil)

	firstCall := true
	toTest.readline = func() (string, error) {
		if firstCall {
			firstCall = false
			return commandMacro, nil
		}
		return commandList, nil
	}

	lines := toTest.Start()

	_, ok := <-lines
	assert.True(t, ok)
	assert.Equal(t, "Test - test macro description\n", output.String(), "all macros should be listed")
}

func TestShell_Start_execMacro(t *testing.T) {
	macroManager := &MacroManager{
		MacroSpecs: map[string]config.Macro{
			"Test": {
				Description: "test macro description",
				Commands:    []string{"first line", "second line"},
			},
		},
	}

	toTest, _ := NewShell("PROMPT>", "/tmp/history", macroManager, nil)

	firstCall := true
	toTest.readline = func() (string, error) {
		if firstCall {
			firstCall = false
			return "Test", nil
		}
		return commandList, nil
	}

	lines := toTest.Start()

	assert.Equal(t, "first line", <-lines)
	assert.Equal(t, "second line", <-lines)

	_, ok := <-lines
	assert.True(t, ok)
}
