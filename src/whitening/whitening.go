package whitening

// This is a whitening library that should be a simple retranscription from C++ to golang.
// Explanation from [ECHOPRINT - AN OPEN MUSIC IDENTIFICATION SERVICE](http://static.echonest.com/echoprint_ismir.pdf):
//
// A 40-pole LPC filter is estimated from the autocorrelation of 1 sec blocks of the signal, smoothed with an 8
// sec decay constant. The inverse (FIR) filter is applied to the signal to achieve whitening. Thus, any strong, stationary
// resonances in the signal arising from speaker, microphone, or room in an OTA scenario will be moderated by a matching zero.

// Whitening represents the transcription of the Whitening class
type Whitening struct {
	pSamples   []float64
	whitened   []float64 // The result of the signal whitened
	numSamples uint64
	r          float64
	xo         float64
	ai         float64
}

const (
	p = 40
)

// R ...
var R = make([]float64, p)

// NewWhitening is the constructor
func NewWhitening(pSamples float64, numSamples int64) *Whitening {
	return &Whitening{
		pSamples:   pSamples,
		numSamples: numSamples,
		p:          40,
	}
}

func init() {
	R[0] = 0.001
}

// Compute the result
func (w *Whitening) Compute() {}

// ComputeBlock does whitene the block of w starting at position
func (w *Whitening) ComputeBlock(start, blockSize int) {
	T := 8
	alpha = 1.0 / T
	// calculate autocorrelation of current block
	for i := 0; i <= p; i++ {
		acc := 0
		for j := 0; j < blockSize; j++ {
			acc += w.pSamples[j+start] * w.pSamples[j-i+start]
		}
		// smoothed update
		R[i] += alpha * (acc - R[i])
	}
}
