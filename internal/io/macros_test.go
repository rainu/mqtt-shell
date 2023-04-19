package io

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/rainu/mqtt-shell/internal/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMacroManager_ResolveMacro_interpretError(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		assert.Equal(t, "someLine", line)
		return Chain{}, errors.New("someError")
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	toTest := MacroManager{}

	result := toTest.ResolveMacro("someLine")

	assert.Equal(t, []string{"someLine"}, result)
}

func TestMacroManager_ResolveMacro_unknownMacro(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "invalid"}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output}

	result := toTest.ResolveMacro("someLine")
	assert.Nil(t, result)
	assert.Equal(t, "unknown macro\n", output.String())
}

func TestMacroManager_ResolveMacro_tooLessArguments(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro"}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{"arg0", "arg1"},
			Varargs:   false,
		},
	}}

	result := toTest.ResolveMacro("macro")
	assert.Nil(t, result)
	assert.Equal(t, "invalid macro arguments\nusage: macro arg0 arg1\n", output.String())
}

func TestMacroManager_ResolveMacro_tooMuchArguments(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro", Arguments: []string{"arg0", "arg1"}}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{},
			Varargs:   false,
		},
	}}

	result := toTest.ResolveMacro("macro")
	assert.Nil(t, result)
	assert.Equal(t, "invalid macro arguments\nusage: macro \n", output.String())
}

func TestMacroManager_ResolveMacro_simple(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro"}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{},
			Commands:  []string{"cmd1", "cmd2"},
		},
	}}

	result := toTest.ResolveMacro("macro")
	assert.Equal(t, []string{"cmd1", "cmd2"}, result)
	assert.Equal(t, "", output.String())
}

func TestMacroManager_ResolveMacro_simplePiped(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro"}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{},
			Commands:  []string{"sub topic/#", "cmd2"},
		},
	}}

	result := toTest.ResolveMacro("macro | wc -l")
	assert.Equal(t, []string{"sub topic/# |  wc -l", "cmd2"}, result)
	assert.Equal(t, "", output.String())
}

func TestMacroManager_ResolveMacro_withArguments(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro", Arguments: []string{"test/#"}}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{"topic"},
			Commands:  []string{"sub $1", "cmd2"},
		},
	}}

	result := toTest.ResolveMacro("macro test/#")
	assert.Equal(t, []string{"sub test/#", "cmd2"}, result)
	assert.Equal(t, "", output.String())
}

func TestMacroManager_ResolveMacro_withVarArguments(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro", Arguments: []string{"test/1", "test/2"}}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{"topic"},
			Varargs:   true,
			Commands:  []string{"sub $1", "cmd2"},
		},
	}}

	result := toTest.ResolveMacro("macro test/1 test/2")
	assert.Equal(t, []string{
		"sub test/1",
		"cmd2",
		"sub test/2",
		"cmd2",
	}, result)
	assert.Equal(t, "", output.String())
}

func TestMacroManager_ResolveMacro_withMixedArguments(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro", Arguments: []string{"test/1", "test/2", "test/3"}}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{"fixed topic", "topic"},
			Varargs:   true,
			Commands:  []string{"sub $1", "sub $2"},
		},
	}}

	result := toTest.ResolveMacro("macro test/1 test/2 test/3")
	assert.Equal(t, []string{
		"sub test/1",
		"sub test/2",
		"sub test/1",
		"sub test/3",
	}, result)
	assert.Equal(t, "", output.String())
}

func TestMacroManager_ResolveMacro_withArguments_piped(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro", Arguments: []string{"test/#"}}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{"topic"},
			Commands:  []string{"sub $1", "cmd2"},
		},
	}}

	result := toTest.ResolveMacro("macro test/# | wc -l")
	assert.Equal(t, []string{"sub test/# |  wc -l", "cmd2"}, result)
	assert.Equal(t, "", output.String())
}

func TestMacroManager_ResolveMacro_withVarArgumentsPiped(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro", Arguments: []string{"test/1", "test/2"}}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{"topic"},
			Varargs:   true,
			Commands:  []string{"sub $1", "cmd2"},
		},
	}}

	result := toTest.ResolveMacro("macro test/1 test/2 | wc -l")
	assert.Equal(t, []string{
		"sub test/1 |  wc -l",
		"cmd2",
		"sub test/2 |  wc -l",
		"cmd2",
	}, result)
	assert.Equal(t, "", output.String())
}

func TestMacroManager_ResolveMacro_script(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro", Arguments: []string{"test/#"}}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{"topic"},
			Script:    "sub {{.Arg1}}\ncmd2",
		},
	}}
	toTest.ValidateAndInitMacros()

	result := toTest.ResolveMacro("macro test/#")
	assert.Equal(t, []string{"sub test/#", "cmd2"}, result)
	assert.Equal(t, "", output.String())
}

func TestMacroManager_ResolveMacro_script_piped(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro", Arguments: []string{"test/#"}}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{"topic"},
			Script:    "sub {{.Arg1}}\ncmd2",
		},
	}}
	toTest.ValidateAndInitMacros()

	result := toTest.ResolveMacro("macro test/# | wc -l")
	assert.Equal(t, []string{"sub test/# |  wc -l", "cmd2"}, result)
	assert.Equal(t, "", output.String())
}

func TestMacroManager_ResolveMacro_scriptVarArgs(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro", Arguments: []string{"test/1", "test/2"}}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{"topic"},
			Varargs:   true,
			Script:    "sub {{.Arg1}}\ncmd2",
		},
	}}
	toTest.ValidateAndInitMacros()

	result := toTest.ResolveMacro("macro test/1 test/2")
	assert.Equal(t, []string{
		"sub test/1",
		"cmd2",
		"sub test/2",
		"cmd2",
	}, result)
	assert.Equal(t, "", output.String())
}

func TestMacroManager_ResolveMacro_scriptVarArgs_piped(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro", Arguments: []string{"test/1", "test/2"}}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{"topic"},
			Varargs:   true,
			Script:    "sub {{.Arg1}}\ncmd2",
		},
	}}
	toTest.ValidateAndInitMacros()

	result := toTest.ResolveMacro("macro test/1 test/2 | wc -l")
	assert.Equal(t, []string{
		"sub test/1 |  wc -l",
		"cmd2",
		"sub test/2 |  wc -l",
		"cmd2",
	}, result)
	assert.Equal(t, "", output.String())
}

func TestMacroManager_ResolveMacro_scriptMixedArgs(t *testing.T) {
	oInterpretLine := interpretLine
	interpretLine = func(line string) (Chain, error) {
		return Chain{
			Commands: []Command{{Name: "macro", Arguments: []string{"test/1", "test/2", "test/3"}}},
		}, nil
	}
	defer func() {
		interpretLine = oInterpretLine
	}()

	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro": {
			Arguments: []string{"fixed topic", "topic"},
			Varargs:   true,
			Script:    "sub {{.Arg1}}\nsub {{.Arg2}}",
		},
	}}
	toTest.ValidateAndInitMacros()

	result := toTest.ResolveMacro("macro test/1 test/2")
	assert.Equal(t, []string{
		"sub test/1",
		"sub test/2",
		"sub test/1",
		"sub test/3",
	}, result)
	assert.Equal(t, "", output.String())
}

func TestMacroManager_PrintMacros(t *testing.T) {
	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output, MacroSpecs: map[string]config.Macro{
		"macro":  {Description: "this will do something"},
		"macro2": {Description: "this will do something else"},
	}}

	toTest.PrintMacros()
	assert.Contains(t, output.String(), "macro - this will do something\n")
	assert.Contains(t, output.String(), "macro2 - this will do something else\n")
}

func TestMacroManager_ValidateAndInitMacros_reservedMacroName(t *testing.T) {
	toTest := MacroManager{MacroSpecs: map[string]config.Macro{
		commandSub: {Description: "this will do something"},
	}}

	err := toTest.ValidateAndInitMacros()
	assert.Error(t, err)
	assert.Equal(t, "invalid macro name '"+commandSub+"': reserved", err.Error())
}

func TestMacroManager_ValidateAndInitMacros_noScriptOrCommand(t *testing.T) {
	toTest := MacroManager{MacroSpecs: map[string]config.Macro{
		"macro": {Description: "this will do something"},
	}}

	err := toTest.ValidateAndInitMacros()
	assert.Error(t, err)
	assert.Equal(t, "invalid macro 'macro': there is no 'commands' nor 'script'", err.Error())
}

func TestMacroManager_ValidateAndInitMacros_scriptAndCommand(t *testing.T) {
	toTest := MacroManager{MacroSpecs: map[string]config.Macro{
		"macro": {Description: "this will do something", Script: "script", Commands: []string{"cmd"}},
	}}

	err := toTest.ValidateAndInitMacros()
	assert.Error(t, err)
	assert.Equal(t, "invalid macro 'macro': only 'commands' or 'script' must be used", err.Error())
}

func TestMacroManager_ValidateAndInitMacros_invalidScript(t *testing.T) {
	toTest := MacroManager{MacroSpecs: map[string]config.Macro{
		"macro": {Description: "this will do something", Script: "{{"},
	}}

	err := toTest.ValidateAndInitMacros()
	assert.Error(t, err)
	assert.Equal(t, "invalid macro 'macro': unable to parse script: template: macro:1: unclosed action", err.Error())
}

func TestMacroManager_macroFuncExec(t *testing.T) {
	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output}

	assert.Equal(t, "hello world\n", toTest.macroFuncExec("echo hello world"))
}

func TestMacroManager_macroFuncLog(t *testing.T) {
	output := &bytes.Buffer{}
	toTest := MacroManager{Output: output}

	assert.Equal(t, "", toTest.macroFuncLog("hello %s", "world"))
	assert.Equal(t, "hello world\n", output.String())
}

func TestMacroManager_IsMacro(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{commandExit, false},
		{commandHelp, false},
		{commandList, false},
		{commandListColors, false},
		{commandPub + " test/topic content", false},
		{commandSub + " test/topic", false},
		{commandUnsub + " test/topic", false},
		{"macro", true},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("TestMacroManager_IsMacro_%d", i), func(t *testing.T) {
			toTest := MacroManager{}
			if test.expected {
				assert.True(t, toTest.IsMacro(test.line))
			} else {
				assert.False(t, toTest.IsMacro(test.line))
			}
		})
	}
}
