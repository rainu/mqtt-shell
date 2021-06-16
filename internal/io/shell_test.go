package io

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
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
