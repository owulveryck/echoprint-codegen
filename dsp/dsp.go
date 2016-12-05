// Package dsp is a wrapper
package dsp

import (
	"io"
)

const (
	size                     = 10000 // default buffer size
	maxConsecutiveEmptyReads = 100
)

// Writer implements DSP for an io.Writer object.
// If an error occurs writing to a Writer, no more data will be
// accepted and all subsequent writes will return the error.
// After all data has been written, the client should call the
// Flush method to guarantee all data has been forwarded to
// the underlying io.Writer.
// The data in the buffer is computed before beeing flushed to the io.writer
type Writer struct {
	err error
	buf []byte
	n   int
	wr  io.Writer
	dsp DSP
}

// DSP is an interface for any signal processing algrithm.
// The Compupte method should be able to
type DSP interface {
	Compute([]byte) []byte
}

// NewWriter returns a new Writer whose buffer has the default size.
func NewWriter(w io.Writer, d DSP) *Writer {
	return &Writer{
		buf: make([]byte, size),
		wr:  w,
		dsp: d,
	}
}

// Available returns how many bytes are unused in the buffer.
func (b *Writer) Available() int { return len(b.buf) - b.n }

// Flush writes any buffered data to the underlying io.Writer.
func (b *Writer) Flush() error {
	err := b.flush()
	return err
}

func (b *Writer) Write(p []byte) (nn int, err error) {
	for len(p) > b.Available() && b.err == nil {
		var n int
		if b.Buffered() == 0 {
			// Large write, empty buffer.
			// Write directly from p to avoid copy.
			// TODO: implement the algorithm
			n, b.err = b.wr.Write(b.dsp.Compute(p))
		} else {
			n = copy(b.buf[b.n:], p)
			b.n += n
			b.flush()
		}
		nn += n
		p = p[n:]
	}
	if b.err != nil {
		return nn, b.err
	}
	n := copy(b.buf[b.n:], p)
	b.n += n
	nn += n
	return nn, nil
}

func (b *Writer) flush() error {
	if b.err != nil {
		return b.err
	}
	if b.n == 0 {
		return nil
	}
	n, err := b.wr.Write(b.dsp.Compute(b.buf[0:b.n]))
	if n < b.n && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		if n > 0 && n < b.n {
			copy(b.buf[0:b.n-n], b.buf[n:b.n])
		}
		b.n -= n
		b.err = err
		return err
	}
	b.n = 0
	return nil
}

// Buffered returns the number of bytes that have been written into the current buffer.
func (b *Writer) Buffered() int { return b.n }

// ReadFrom implements io.ReaderFrom.
func (b *Writer) ReadFrom(r io.Reader) (n int64, err error) {
	if b.Buffered() == 0 {
		if w, ok := b.wr.(io.ReaderFrom); ok {
			return w.ReadFrom(r)
		}
	}
	var m int
	for {
		if b.Available() == 0 {
			if err1 := b.flush(); err1 != nil {
				return n, err1
			}
		}
		nr := 0
		for nr < maxConsecutiveEmptyReads {
			m, err = r.Read(b.buf[b.n:])
			if m != 0 || err != nil {
				break
			}
			nr++
		}
		if nr == maxConsecutiveEmptyReads {
			return n, io.ErrNoProgress
		}
		b.n += m
		n += int64(m)
		if err != nil {
			break
		}
	}
	if err == io.EOF {
		// If we filled the buffer exactly, flush preemptively.
		if b.Available() == 0 {
			err = b.flush()
		} else {
			err = nil
		}
	}
	return n, err
}
