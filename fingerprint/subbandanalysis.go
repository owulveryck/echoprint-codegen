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
	c = []float32{
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

// Fingerprinter implements the DSP interface
type Fingerprinter struct {
	mr        *mat64.Dense
	mi        *mat64.Dense
	numFrames int64
	data      *mat64.Dense
}

// NewFingerprinter returns a fingerprinter
func NewFingerprinter() *Fingerprinter {
	mr := mat64.NewDense(mRows, mCols, nil)
	mi := mat64.NewDense(mRows, mCols, nil)
	for i := 0; i < mRows; i++ {
		for k := 0; k < mCols; k++ {
			mr.Set(i, k, math.Cos(float64((2*i+1)*(k-4))*(math.Pi/16.0)))
			mi.Set(i, k, math.Sin(float64((2*i+1)*(k-4))*(math.Pi/16.0)))
		}
	}
	return &Fingerprinter{
		mr: mr,
		mi: mi,
	}
}

// Compute the fingerprint
func (f Fingerprinter) Compute(p []byte) []byte {
	buf := bytes.NewReader(p)
	numSamples := len(p) / 2
	var samples = make([]int16, numSamples)
	var out = make([]int16, numSamples)
	// loop until EOF
	// Read data as a byte array
	binary.Read(buf, binary.LittleEndian, samples)

	z := make([]float32, cLen)
	y := make([]float32, mCols)

	numFrames := (numSamples - cLen + 1) / subbands
	if numFrames == 0 {
		panic(errors.New("No frames to compute"))
	}
	data := mat64.NewDense(subbands, numFrames, nil)
	for t := 0; t < numFrames; t++ {
		for i := 0; i < cLen; i++ {
			z[i] = float32(samples[t*subbands+i]) * c[i]
		}
		for i := 0; i < mCols; i++ {
			y[i] = z[i]
		}
		for i := 0; i < mCols; i++ {
			for j := 1; j < mRows; j++ {
				y[i] += z[i+mCols*j]
			}
		}
		for i := 0; i < mRows; i++ {
			var dr, di float32
			for j := 0; j < mCols; j++ {
				dr += float32(f.mr.At(i, j)) * y[j]
				di -= float32(f.mi.At(i, j)) * y[j]
			}
			data.Set(i, t, float64(dr*dr+di*di))
		}
	}

	// write data back in the buffer
	var output []byte
	o := bytes.NewBuffer(output)
	binary.Write(o, binary.LittleEndian, out)
	return o.Bytes()
}
