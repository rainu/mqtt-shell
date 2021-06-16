package io

import "io"

type prefixWriter struct {
	Prefix        string
	prefixWritten bool
	Delegate      io.Writer
}

func (p *prefixWriter) Write(b []byte) (n int, err error) {
	givenLen := len(b)
	writePrefix := !p.prefixWritten
	if writePrefix {
		b2 := make([]byte, 0, len(b)+len(p.Prefix))
		b2 = append(b2, []byte(p.Prefix)...)
		b2 = append(b2, b...)
		b = b2

		p.prefixWritten = true
	}

	n, err = p.Delegate.Write(b)
	if writePrefix && err == nil {
		//in happy case we have to make sure that the correct amount of read bytes
		//are returned -> otherwise this will cause many io trouble
		n = givenLen
	}

	return
}
