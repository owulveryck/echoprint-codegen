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
	"encoding/binary"
	"io"
)

// A Reader implements the io.Reader
type Reader struct {
	r        io.Reader
	whitened []float32 // The result of the signal whitened
}

const (
	p     = 40  // 40-pole LPC filter
	t     = 8.0 // decay constant
	alpha = 1.0 / t
)

// NewReader returns a new Reader reading from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		r: r,
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
func (r *Reader) Read(p []byte) (int, error) {
	return 0, nil
}

// Compute the result and returns the number of segment processed
func (r *Reader) Compute() (int, error) {
	i := 0
	n := 100000
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
