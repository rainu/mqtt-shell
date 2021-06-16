package io

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"strings"
	"testing"
)

func TestPrefixWriter_Write(t *testing.T) {
	sSource := `some test text`
	source := strings.NewReader(sSource)
	target := bytes.NewBuffer([]byte{})

	toTest := &prefixWriter{
		Prefix:   "prefix | ",
		Delegate: target,
	}

	n, err := io.Copy(toTest, source)

	assert.NoError(t, err)
	assert.EqualValues(t, len(sSource), n)
	assert.Equal(t, toTest.Prefix+sSource, target.String())
}
