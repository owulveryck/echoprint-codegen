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
)

const (
	lpcOrder = 40  // 40-pole LPC filter
	t        = 8.0 // decay constant
	alpha    = 1.0 / t
)

// Whitener implements DSP interface
type Whitener struct {
	X []int16
}

// NewWhitener returns a new Whitener
func NewWhitener() *Whitener {
	return &Whitener{
		X: make([]int16, lpcOrder+1),
	}
}

// Compute the []bytes and return a whitened version
func (b Whitener) Compute(p []byte) []byte {
	buf := bytes.NewReader(p)
	blockSize := len(p) / 2
	var R []float32 // Short time autocorrelation vector
	R = make([]float32, lpcOrder+1)
	R[0] = 0.001
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
		R[i] = alpha * (sum - R[i])
	}
	// calculate new filter coefficients
	// Durbin's recursion, per p. 411 of Rabiner & Schafer 1978
	var E float32 // Error
	E = R[0]
	var ai = make([]int16, blockSize)
	for i := 1; i <= lpcOrder; i++ {
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
			acc -= ai[j] * b.X[lpcOrder+i-j]
		}
		for j := 1; j <= minip; j++ {
			acc -= ai[j] * data[i-j]
		}
		out[i] = acc
	}

	// save last few frames of input
	for i := 0; i <= lpcOrder; i++ {
		b.X[i] = data[blockSize-1-lpcOrder+i]
	}
	// write data back in the buffer
	var output []byte
	o := bytes.NewBuffer(output)
	binary.Write(o, binary.LittleEndian, out)
	return o.Bytes()
}
