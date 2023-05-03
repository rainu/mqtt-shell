package io

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRemoveDecoratorFromPool(t *testing.T) {
	odp := decoratorPool
	defer func() {
		decoratorPool = odp
	}()

	decoratorPool = []decorator{
		[]string{"DEC", "1"},
		[]string{"DEC", "2"},
		[]string{"DEC", "3"},
	}

	RemoveDecoratorFromPool("DEC 2")

	assert.Equal(t, []decorator{
		[]string{"DEC", "1"},
		[]string{"DEC", "3"},
	}, decoratorPool)
}

func TestGetNextDecorator(t *testing.T) {
	odp := decoratorPool
	defer func() {
		decoratorPool = odp
	}()

	decoratorPool = []decorator{
		[]string{"DEC", "1"},
		[]string{"DEC", "2"},
	}

	assert.Equal(t, decorator{"DEC", "1"}, getNextDecorator())
	assert.Equal(t, decorator{"DEC", "2"}, getNextDecorator())
	assert.Equal(t, decorator{"DEC", "1"}, getNextDecorator())
}

func TestDecorate(t *testing.T) {
	assert.Equal(t, "\x1b[1mTEXT\x1b[0m", decorate("TEXT", "1"))
	assert.Equal(t, "\x1b[1m\x1b[2mTEXT\x1b[0m", decorate("TEXT", "1", "2"))
}
