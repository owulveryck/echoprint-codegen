// Package fingerprint does the fingerprinting;
//
// This file does the analisys as explained in the original paper (http://static.echonest.com/echoprint_ismir.pdf):
//  Onset detection is performed independently in 8 frequency bands, corresponding to the
// lowest 8 bands in the MPEG-Audio 32 band filterbank (hence, nominally spanning 0 to 5512.5 Hz).
// The magnitude of the complex band-pass signal in each band is compared to
// an exponentially-decaying threshold, and an onset recorded  when the signal exceeds the threshold,
// at which point the threshold is increased to 1.05 x the new signal peak.
//
// The algorithm is copyrighted:
//  echoprint-codegen
//  Copyright 2011 The Echo Nest Corporation. All rights reserved.
package fingerprint

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/gonum/matrix/mat64"
	"math"
)

const (
	cLen     = 128
	subbands = 8
	mRows    = 8
	mCols    = 8
)

var (
	mr *mat64.Dense
	mi *mat64.Dense
	c  = []float32{
		0.000000477, 0.000000954, 0.000001431, 0.000002384, 0.000003815, 0.000006199, 0.000009060, 0.000013828,
		0.000019550, 0.000027657, 0.000037670, 0.000049591, 0.000062943, 0.000076771, 0.000090599, 0.000101566,
		-0.000108242, -0.000106812, -0.000095367, -0.000069618, -0.000027180, 0.000034332, 0.000116348, 0.000218868,
		0.000339031, 0.000472546, 0.000611782, 0.000747204, 0.000866413, 0.000954151, 0.000994205, 0.000971317,
		-0.000868797, -0.000674248, -0.000378609, 0.000021458, 0.000522137, 0.001111031, 0.001766682, 0.002457142,
		0.003141880, 0.003771782, 0.004290581, 0.004638195, 0.004752159, 0.004573822, 0.004049301, 0.003134727,
		-0.001800537, -0.000033379, 0.002161503, 0.004756451, 0.007703304, 0.010933399, 0.014358521, 0.017876148,
		0.021372318, 0.024725437, 0.027815342, 0.030526638, 0.032754898, 0.034412861, 0.035435200, 0.035780907,
		-0.035435200, -0.034412861, -0.032754898, -0.030526638, -0.027815342, -0.024725437, -0.021372318, -0.017876148,
		-0.014358521, -0.010933399, -0.007703304, -0.004756451, -0.002161503, 0.000033379, 0.001800537, 0.003134727,
		-0.004049301, -0.004573822, -0.004752159, -0.004638195, -0.004290581, -0.003771782, -0.003141880, -0.002457142,
		-0.001766682, -0.001111031, -0.000522137, -0.000021458, 0.000378609, 0.000674248, 0.000868797, 0.000971317,
		-0.000994205, -0.000954151, -0.000866413, -0.000747204, -0.000611782, -0.000472546, -0.000339031, -0.000218868,
		-0.000116348, -0.000034332, 0.000027180, 0.000069618, 0.000095367, 0.000106812, 0.000108242, 0.000101566,
		-0.000090599, -0.000076771, -0.000062943, -0.000049591, -0.000037670, -0.000027657, -0.000019550, -0.000013828,
		-0.000009060, -0.000006199, -0.000003815, -0.000002384, -0.000001431, -0.000000954, -0.000000477, 0,
	}
)

func init() {
	mr = mat64.NewDense(mRows, mCols, nil)
	mi = mat64.NewDense(mRows, mCols, nil)
	for i := 0; i < mRows; i++ {
		for k := 0; k < mCols; k++ {
			mr.Set(i, k, math.Cos(float64((2*i+1)*(k-4))*(math.Pi/16.0)))
			mi.Set(i, k, math.Sin(float64((2*i+1)*(k-4))*(math.Pi/16.0)))
		}
	}

}

// Code holds the couple time/Hash
// the time represent the time for onset ms quantized
type Code struct {
	Time int
	Hash uint32
}

// Fingerprinter implements the DSP interface
type Fingerprinter struct {
	numFrames int64
	data      *mat64.Dense
	Hash      chan Code
	samples   *[]int16
}

// NewFingerprinter returns a fingerprinter
func NewFingerprinter() *Fingerprinter {
	return &Fingerprinter{
		Hash: make(chan Code),
	}
}

func subbandAnalysis(samples *[]int16) *mat64.Dense {
	numSamples := len(*samples)
	z := make([]float32, cLen)
	y := make([]float32, mCols)

	numFrames := (numSamples - cLen + 1) / subbands
	if numFrames == 0 {
		panic(errors.New("No frames to compute"))
	}
	data := mat64.NewDense(subbands, numFrames, nil)
	for t := 0; t < numFrames; t++ {
		for i := 0; i < cLen; i++ {
			z[i] = float32((*samples)[t*subbands+i]) * c[i]
		}
		//for i := 0; i < mCols; i++ {
		//	y[i] = z[i]
		//}
		for i := 0; i < mCols; i++ {
			y[i] = z[i]
			for j := 1; j < mRows; j++ {
				y[i] += z[i+mCols*j]
			}
		}
		for i := 0; i < mRows; i++ {
			var dr, di float32
			for j := 0; j < mCols; j++ {
				dr += float32(mr.At(i, j)) * y[j]
				di -= float32(mi.At(i, j)) * y[j]
			}
			data.Set(i, t, float64(dr*dr+di*di))
		}
	}

	return data
}

// Compute the signal and returns the fingerprints
func (f Fingerprinter) Compute(p []byte) []byte {
	buf := bytes.NewReader(p)
	numSamples := len(p) / 2
	var samples = make([]int16, numSamples)
	// Read data as a byte array
	binary.Read(buf, binary.LittleEndian, samples)

	// Fingerprinting
	go func() {
		var out *mat64.Dense
		//var actualCodes uint16
		//actualCodes = 0
		var onsetCounterForBand *[]uint
		//var onsetCount uint16
		//onsetCount = adaptativeOnsets(345, out, onsetCounterForBand)
		adaptativeOnsets(345, out, onsetCounterForBand, &samples)
		for band := 0; band < subbands; band++ {
			if (*onsetCounterForBand)[band] > 2 {
				for onset := 0; onset < int((*onsetCounterForBand)[band]-uint(2)); onset++ {
					// What time was this onset at?
					timeForOnsetMsQuantized := quantizedTimeForFrameAbsolute(out.At(band, onset))
					var nhashes = 6
					var p = make([][]uint, 2)
					p[0] = make([]uint, nhashes)
					p[1] = make([]uint, nhashes)
					for i := 0; i < 6; i++ {
						p[0][i] = 0
						p[1][i] = 0
					}
					if onset == int((*onsetCounterForBand)[band])-4 {
						nhashes = 3
					}
					if onset == int((*onsetCounterForBand)[band])-3 {
						nhashes = 1
					}
					p[0][0] = uint(out.At(band, onset+1) - out.At(band, onset))
					p[1][0] = uint(out.At(band, onset+2) - out.At(band, onset+1))
					if nhashes > 1 {
						p[0][1] = uint(out.At(band, onset+1) - out.At(band, onset))
						p[1][1] = uint(out.At(band, onset+3) - out.At(band, onset+1))
						p[0][2] = uint(out.At(band, onset+2) - out.At(band, onset))
						p[1][2] = uint(out.At(band, onset+3) - out.At(band, onset+2))
						if nhashes > 3 {
							p[0][3] = uint(out.At(band, onset+1) - out.At(band, onset))
							p[1][3] = uint(out.At(band, onset+4) - out.At(band, onset+1))
							p[0][4] = uint(out.At(band, onset+2) - out.At(band, onset))
							p[1][4] = uint(out.At(band, onset+4) - out.At(band, onset+2))
							p[0][5] = uint(out.At(band, onset+3) - out.At(band, onset))
							p[1][5] = uint(out.At(band, onset+4) - out.At(band, onset+3))
						}
					}
					// For each pair emit a code
					for k := 0; k < 6; k++ {
						// Quantize the time deltas to 23ms
						timeDelta0 := quantizedTimeForFrameDelta(p[0][k])
						timeDelta1 := quantizedTimeForFrameDelta(p[1][k])
						f.Hash <- Code{
							Time: timeForOnsetMsQuantized,
							Hash: hashFunc(timeDelta0, timeDelta1, band),
						}
					}
				}
			}
		}
	}()

	// return the data unmodified
	return p
}

func adaptativeOnsets(ttarg int, out *mat64.Dense, onsetCounterForBand *[]uint, samples *[]int16) uint16 {
	//  E is a sgram-like matrix of energies.
	/*
		var e float32
		var bands, frames, i, j, k int
		deadtime := 128
		overfact := 1.1 // threshold relative to actual peak
		onsetCounter := 0
		E := subbandAnalysis(samples)
	*/
	// Take successive stretches of 8 subband samples and sum their energy under a hann window, then hop by 4 samples (50% window overlap).
	return 0
}
func quantizedTimeForFrameDelta(f uint) int {
	return 0
}
func quantizedTimeForFrameAbsolute(f float64) int {
	return 0
}
func hashFunc(t0, t1, band int) uint32 {
	var out uint32
	return out
}
