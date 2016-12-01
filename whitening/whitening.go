// Package whitening computes a whitening on a PCM signal.
// The algorithm is a simple retranscription from the echoprint-codegen C++
// implementation to golang.
//
// The code is a copied from the bufio package.
// It implements a buffered writer which computes the whitening filter just
// before flushing its data to the next writer.
//
// The copyright goes to the Echo Nest Corporation for the algorithm
// and to the Go team and Olivier Wulveryck for the go implementation.
//
// Explanation from the paper "ECHOPRINT - AN OPEN MUSIC IDENTIFICATION SERVICE"
// (http://static.echonest.com/echoprint_ismir.pdf):
//
// A 40-pole LPC filter is estimated from the autocorrelation of 1 sec blocks of the signal, smoothed with an 8
// sec decay constant. The inverse (FIR) filter is applied to the signal to achieve whitening. Thus, any strong, stationary
// resonances in the signal arising from speaker, microphone, or room in an OTA scenario will be moderated by a matching zero.
package whitening

import (
	"bytes"
	"encoding/binary"
	"io"
)

const (
	lpcOrder                 = 40  // 40-pole LPC filter
	t                        = 8.0 // decay constant
	alpha                    = 1.0 / t
	size                     = 10000 // default buffer size
	maxConsecutiveEmptyReads = 100
)

// Writer implements Whitening for an io.Writer object.
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
}

// NewWriter returns a new Writer whose buffer has the default size.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		buf: make([]byte, size),
		wr:  w,
	}
}

// Available returns how many bytes are unused in the buffer.
func (b *Writer) Available() int { return len(b.buf) - b.n }

// Flush writes any buffered data to the underlying io.Writer.
func (b *Writer) Flush() error {
	err := b.flush()
	return err
}

func computeBlock(p []byte) []byte {
	buf := bytes.NewReader(p)
	blockSize := len(p) / 2
	var R []float32 // Short time autocorrelation vector
	R = make([]float32, lpcOrder+1)
	var data = make([]int16, blockSize)
	var out = make([]int16, blockSize)
	// loop until EOF
	// Read data as a byte array
	binary.Read(buf, binary.LittleEndian, data)
	// calculate autocorrelation of current block
	for i := 0; i <= lpcOrder; i++ {
		// sum is the autocorrelation value for timelapse i
		var sum float32
		for j := i; j < blockSize; j++ {
			sum += float32(data[j] * data[j-i])
		}
		// smoothed update
		R[i] = alpha * sum
	}
	// calculate new filter coefficients
	// Durbin's recursion, per p. 411 of Rabiner & Schafer 1978
	var E float32 // Error
	var ai = make([]int16, blockSize)
	for i := 1; i < lpcOrder; i++ {
		var sumAlphaR float32
		sumAlphaR = 0
		for j := 1; j < i; j++ {
			sumAlphaR += float32(ai[j]) * R[i-j]
		}
		ki := int16((R[i] - sumAlphaR) / E)
		ai[i] = ki
		for j := 1; j <= i/2; j++ {
			aj := ai[j]
			aimj := ai[i-j]
			ai[j] = aj - ki*aimj
			ai[i-j] = aimj - ki*aj
		}
		E = (1 - float32(ki*ki)) * E
	}
	// Calculate the new output
	for i := 0; i < blockSize; i++ {
		acc := data[i]
		minip := i
		if lpcOrder < minip {
			minip = lpcOrder
		}
		for j := i + 1; j <= lpcOrder; j++ {
			acc -= ai[j]
		}
		for j := 1; j <= minip; j++ {
			acc -= ai[j] * data[i-j]
		}
		out[i] = acc
	}

	// write data back in the buffer
	var output []byte
	o := bytes.NewBuffer(output)
	binary.Write(o, binary.LittleEndian, out)
	return o.Bytes()
}

func (b *Writer) Write(p []byte) (nn int, err error) {
	for len(p) > b.Available() && b.err == nil {
		var n int
		if b.Buffered() == 0 {
			// Large write, empty buffer.
			// Write directly from p to avoid copy.
			// TODO: implement the algorithm
			n, b.err = b.wr.Write(computeBlock(p))
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
	n, err := b.wr.Write(computeBlock(b.buf[0:b.n]))
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

/*
// computeBlock does whitene the block of w starting at position
func (r *Reader) computeBlock(data []float32, offset int) {
	// Increase the capacity of whitened
	t := make([]float32, len(r.whitened), len(r.whitened)+len(data))
	copy(t, r.whitened)
	r.whitened = t

	var rr []float32
	rr[0] = 0.001
	var xo []float32
	var ai []float32
	blockSize := len(data)
	// calculate autocorrelation of current block
	for i := 0; i <= p; i++ {
		var acc float32
		for j := i; j < blockSize; j++ {
			//acc += r.pSamples[j+start] * r.pSamples[j-i+start]
			acc += data[j] * data[j-i]
		}
		// smoothed update
		rr[i] += alpha * (acc - rr[i])
	}
	// calculate new filter coefficients
	// Durbin's recursion, per p. 411 of Rabiner & Schafer 1978
	E := rr[0]
	for i := 0; i <= p; i++ {
		var sumAlphaR float32
		sumAlphaR = 0
		for j := 1; j <= i; j++ {
			sumAlphaR += ai[j] * rr[i-j]
		}
		ki := (rr[i] - sumAlphaR) / E
		ai[i] = ki
		for j := 1; j <= i/2; j++ {
			aj := ai[j]
			aimj := ai[i-j]
			ai[j] = aj - ki*aimj
			ai[i-j] = aimj - ki*aj
		}
		E = (1 - ki*ki) * E
	}
	// calculate new output
	for i := 0; i < blockSize; i++ {
		//acc := r.pSamples[i+start]
		acc := data[i]
		minip := i
		if p < minip {
			minip = p
		}
		for j := i + 1; j <= p; j++ {
			acc -= ai[j] * xo[p+i-j]
		}
		for j := 1; j <= minip; j++ {
			//acc -= ai[j] * r.pSamples[i-j+start]
			acc -= ai[j] * data[i-j]
		}
		//r.whitened[i+start] = acc
		r.whitened[i+offset] = acc
	}
	// save last few frames of input
	for i := 0; i <= p; i++ {
		//xo[i] = r.pSamples[blockSize-1-p+i+start]
		xo[i] = data[blockSize-1-p+i]
	}
}
*/
