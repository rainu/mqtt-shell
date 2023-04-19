package io

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInterpretLine(t *testing.T) {
	tests := []struct {
		line     string
		expected Chain
		err      string
	}{
		{"name   ", Chain{
			[]Command{{"name", nil}},
			nil,
			[]string{"name"},
		}, ""},
		{"name cmd1 cmd2", Chain{
			[]Command{{"name", []string{"cmd1", "cmd2"}}},
			nil,
			[]string{"name", "cmd1", "cmd2"},
		}, ""},
		{"   name   cmd1    cmd2   ", Chain{
			[]Command{{"name", []string{"cmd1", "cmd2"}}},
			nil,
			[]string{"name", "cmd1", "cmd2"},
		}, ""},
		{`   name   "cmd with spaces"`, Chain{
			[]Command{{"name", []string{"cmd with spaces"}}},
			nil,
			[]string{"name", "cmd with spaces"},
		}, ""},
		{`   name   "cmd with escaped \""`, Chain{
			[]Command{{"name", []string{"cmd with escaped \""}}},
			nil,
			[]string{"name", `cmd with escaped "`},
		}, ""},
		{`   name   'cmd with "'`, Chain{
			[]Command{{"name", []string{"cmd with \""}}},
			nil,
			[]string{"name", `cmd with "`},
		}, ""},
		{`echo test | grep t`, Chain{
			[]Command{{"echo", []string{"test"}}, {"grep", []string{"t"}}},
			[]string{"|"},
			[]string{"echo", "test", "|", "grep", "t"},
		}, ""},
		{`echo test |& grep t | wc -l`, Chain{
			[]Command{{"echo", []string{"test"}}, {"grep", []string{"t"}}, {"wc", []string{"-l"}}},
			[]string{"|&", "|"},
			[]string{"echo", "test", "|&", "grep", "t", "|", "wc", "-l"},
		}, ""},
		{`unfinished "quote`, Chain{RawLine: []string{"unfinished"}}, "Unterminated double-quoted string"},
		{`unfinished 'quote`, Chain{RawLine: []string{"unfinished"}}, "Unterminated single-quoted string"},
		{"multiline arg1 <<EOF\nthis is\na multiline\nargumentEOF", Chain{
			[]Command{{"multiline", []string{"arg1", "this is\na multiline\nargument"}}},
			nil,
			[]string{"multiline", "arg1", "this is\na multiline\nargument"},
		}, ""},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("TestInterpretLine_%d", i), func(t *testing.T) {
			result, err := interpretLine(test.line)

			if test.err == "" {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, test.err, err.Error())
			}
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestChain_ToCommand(t *testing.T) {
	toTest, err := interpretLine(`sub test | echo test |& grep t | wc -l`)
	assert.NoError(t, err)

	testInput := &bytes.Buffer{}
	testOutput := &bytes.Buffer{}

	cmd, fn, err := toTest.ToCommand(testInput, testOutput)
	defer fn()

	assert.NoError(t, err)

	assert.Equal(t,
		`[IS] *bytes.Buffer ╮
[OS]               │                                                               ╭ *bytes.Buffer
[SO]               │                      ╭╮                   ╭╮                  │
[CM]               ╰ /usr/bin/echo "test" ╡╞ /usr/bin/grep "t" ╡╰ /usr/bin/wc "-l" ╡
[SE]                                      ╰╯                   ╽                   ╽`, cmd.String())
}

func TestChain_ToCommand_Appending(t *testing.T) {
	toTest, err := interpretLine(`sub test | echo test |& grep t | wc -l > /tmp/test`)
	assert.NoError(t, err)

	testInput := &bytes.Buffer{}
	testOutput := &bytes.Buffer{}

	cmd, fn, err := toTest.ToCommand(testInput, testOutput)
	defer fn()

	assert.NoError(t, err)

	assert.Equal(t,
		`[IS] *bytes.Buffer ╮
[OS]               │                                                               ╭ *bytes.Buffer, *os.File
[SO]               │                      ╭╮                   ╭╮                  │
[CM]               ╰ /usr/bin/echo "test" ╡╞ /usr/bin/grep "t" ╡╰ /usr/bin/wc "-l" ╡
[SE]                                      ╰╯                   ╽                   ╽`, cmd.String())
}

func TestChain_IsLongTerm(t *testing.T) {
	toTest, err := interpretLine(`sub test | echo test |& grep t | wc -l &`)
	assert.NoError(t, err)

	assert.True(t, toTest.IsLongTerm())
}
