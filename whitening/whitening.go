// Package whitening computes a whitening on a PCM signal.
// The algorithm is a simple retranscription from the echoprint-codegen C++
// implementation to golang. I've add some readers to be more idiomatic.
// The copyright may goes to the Echo Nest Corporation for the algorithm
// and to Olivier Wulveryck for the go implementation..
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
	size = 10000 // default buffer size
)

// Writer implements buffering for an io.Writer object.
// If an error occurs writing to a Writer, no more data will be
// accepted and all subsequent writes will return the error.
// After all data has been written, the client should call the
// Flush method to guarantee all data has been forwarded to
// the underlying io.Writer.
type Writer struct {
	err error
	buf []byte
	n   int
	wr  io.Writer
}

// NewWriter ...
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

func (b *Writer) Write(p []byte) (nn int, err error) {
	for len(p) > b.Available() && b.err == nil {
		var n int
		if b.Buffered() == 0 {
			// Large write, empty buffer.
			// Write directly from p to avoid copy.
			// TODO: implement the algorithm
			n, b.err = b.wr.Write(p)
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
	n, err := b.wr.Write(b.buf[0:b.n])
	if n < b.n && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		if n > 0 && n < b.n {
			// TODO : Comhpte
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

// ----------------------------------

// A Reader implements the io.Reader
type Reader struct {
	r         io.Reader
	channels  int
	rate      int
	bits      int
	byteOrder binary.ByteOrder
}

const (
	lpcOrder = 40  // 40-pole LPC filter
	t        = 8.0 // decay constant
	alpha    = 1.0 / t
)

// NewReader returns a new Reader reading from r.
func NewReader(r io.Reader, channels, rate, bits int, byteOrder binary.ByteOrder) *Reader {
	return &Reader{
		r:         r,
		channels:  channels,
		rate:      rate,
		bits:      bits,
		byteOrder: byteOrder,
	}
}

// Read reads up to len(p) bytes into p. It returns the number of bytes
// read (0 <= n <= len(p)) and any error encountered. Even if Read
// returns n < len(p), it may use all of p as scratch space during the call.
// If some data is available but not len(p) bytes, Read conventionally
// returns what is available instead of waiting for more.
//
// When Read encounters an error or end-of-file condition after
// successfully reading n > 0 bytes, it returns the number of
// bytes read. It may return the (non-nil) error from the same call
// or return the error (and n == 0) from a subsequent call.
// An instance of this general case is that a Reader returning
// a non-zero number of bytes at the end of the input stream may
// return either err == EOF or err == nil. The next Read should
// return 0, EOF.
//
// Callers should always process the n > 0 bytes returned before
// considering the error err. Doing so correctly handles I/O errors
// that happen after reading some bytes and also both of the
// allowed EOF behaviors.
//
// Implementations of Read are discouraged from returning a
// zero byte count with a nil error, except when len(p) == 0.
// Callers should treat a return of 0 and nil as indicating that
// nothing happened; in particular it does not indicate EOF.
// the information expected through r is PCM signed 16 bit flow
// rated at 11025 and ordered in little endian
func (r *Reader) Read(p []byte) (int, error) {
	var correlation []float32
	correlation = make([]float32, lpcOrder+1)
	blockSize := len(p) / 2
	data := make([]int16, blockSize)
	// loop until EOF
	if blockSize > 0 {
		// Read data as a byte array
		err := binary.Read(r.r, binary.LittleEndian, data)
		if err != nil {
			if err != io.EOF {
				return 0, err
			}
		}
		// calculate autocorrelation of current block
		for i := 0; i <= lpcOrder; i++ {
			// sum is the autocorrelation value for timelapse i
			var sum float32
			for j := i; j < blockSize; j++ {
				sum += float32(data[j] * data[j-i])
			}
			// smoothed update
			correlation[i] = alpha * sum
		}
		// write data back in the buffer
		buf := bytes.NewBuffer(p)
		binary.Write(buf, binary.LittleEndian, data)
		p = buf.Bytes()
	}
	return blockSize * 2, nil
}

/*

// Compute the result and returns the number of segment processed
func (r *Reader) Compute() (int, error) {
	i := 0
	n := 10000
	data := make([]float32, n)
	for {
		// Read the flow from stdin
		if err := binary.Read(r.r, binary.LittleEndian, data); err != nil {
			if err != io.EOF {
				return n * i, err
			}
			break
		}

		r.computeBlock(data, n*i)
		i++
	}
	return n * i, nil
}

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
