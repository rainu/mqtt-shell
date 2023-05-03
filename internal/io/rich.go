package io

import (
	"fmt"
	"github.com/gookit/color"
	"strings"
	"sync"
)

type decorator []string

var decoratorPool []decorator
var nextDecorator = 0
var decoratorPoolLock = sync.Mutex{}

func init() {
	colorLevel := color.DetectColorLevel()

	if colorLevel == color.Level16 {
		for i := 30; i <= 37; i++ {
			decoratorPool = append(decoratorPool, []string{fmt.Sprintf("%d", i)})
		}
		for i := 30; i <= 37; i++ {
			decoratorPool = append(decoratorPool, []string{fmt.Sprintf("%d", i), "1"})
		}
	} else if colorLevel >= color.Level256 {
		for i := 15; i >= 0; i-- {
			for j := 0; j < 16; j++ {
				decoratorPool = append(decoratorPool, []string{fmt.Sprintf("38;5;%d", j*16+i)})
			}
		}
		for i := 15; i >= 0; i-- {
			for j := 0; j < 16; j++ {
				decoratorPool = append(decoratorPool, []string{fmt.Sprintf("38;5;%d", j*16+i), "1"})
			}
		}
	}
}

func (d decorator) String() string {
	return strings.Join(d, " ")
}

func RemoveDecoratorFromPool(decorator string) {
	toRemove := -1

	for i, dec := range decoratorPool {
		if dec.String() == decorator {
			toRemove = i
			break
		}
	}

	if toRemove != -1 {
		decoratorPool = append(decoratorPool[:toRemove], decoratorPool[toRemove+1:]...)
	}
}

var getNextDecorators = func() decorator {
	decoratorPoolLock.Lock()
	defer decoratorPoolLock.Unlock()

	if len(decoratorPool) == 0 {
		return nil
	}

	if nextDecorator >= len(decoratorPool) {
		nextDecorator = 0
	}
	defer func() {
		nextDecorator++
	}()

	return decoratorPool[nextDecorator]
}

func decorate(text string, richCodes ...string) string {
	sb := strings.Builder{}

	for _, richCode := range richCodes {
		sb.WriteString("\u001B[")
		sb.WriteString(richCode)
		sb.WriteString("m")
	}
	sb.WriteString(text)
	sb.WriteString("\u001B[0m")

	return sb.String()
}
