package io

import "io"

type prefixWriter struct {
	Prefix        string
	prefixWritten bool
	Delegate      io.Writer
}

func (p *prefixWriter) Write(b []byte) (n int, err error) {
	if !p.prefixWritten {
		b2 := make([]byte, len(b)+len(p.Prefix))
		b2 = append(b2, []byte(p.Prefix)...)
		b2 = append(b2, b...)
		b = b2

		p.prefixWritten = true
	}

	return p.Delegate.Write(b)
}
