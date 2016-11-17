package whitening

// This is a whitening library for echoprint
// it should be a simple retranscription from C++ to golang.
// Therefore the copyright may go to the Echo Nest Corporation.
//
// Explanation from [ECHOPRINT - AN OPEN MUSIC IDENTIFICATION SERVICE](http://static.echonest.com/echoprint_ismir.pdf):
//
// A 40-pole LPC filter is estimated from the autocorrelation of 1 sec blocks of the signal, smoothed with an 8
// sec decay constant. The inverse (FIR) filter is applied to the signal to achieve whitening. Thus, any strong, stationary
// resonances in the signal arising from speaker, microphone, or room in an OTA scenario will be moderated by a matching zero.

// Whitening represents the transcription of the Whitening class
type Whitening struct {
	pSamples   []float32
	whitened   []float32 // The result of the signal whitened
	numSamples uint64
}

const (
	p     = 40  // 40-pole LPC filter
	t     = 8.0 // decay constant
	alpha = 1.0 / t
)

var r []float32
var xo []float32
var ai []float32

// NewWhitening is the constructor
func NewWhitening(pSamples *[]float32, numSamples uint64) *Whitening {
	r = make([]float32, numSamples)
	r[0] = 0.001
	xo = make([]float32, numSamples)
	ai = make([]float32, numSamples)
	return &Whitening{
		pSamples:   *pSamples,
		numSamples: numSamples,
		whitened:   make([]float32, len(*pSamples)),
	}
}

// Compute the result
func (w *Whitening) Compute() {
	w.computeBlock(0, len(w.pSamples))
}

// computeBlock does whitene the block of w starting at position
func (w *Whitening) computeBlock(start, blockSize int) {
	// calculate autocorrelation of current block
	for i := 0; i <= p; i++ {
		var acc float32
		for j := i; j < blockSize; j++ {
			acc += w.pSamples[j+start] * w.pSamples[j-i+start]
		}
		// smoothed update
		r[i] += alpha * (acc - r[i])
	}
	// calculate new filter coefficients
	// Durbin's recursion, per p. 411 of Rabiner & Schafer 1978
	E := r[0]
	for i := 0; i <= p; i++ {
		var sumAlphaR float32
		sumAlphaR = 0
		for j := 1; j <= i; j++ {
			sumAlphaR += ai[j] * r[i-j]
		}
		ki := (r[i] - sumAlphaR) / E
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
		acc := w.pSamples[i+start]
		minip := i
		if p < minip {
			minip = p
		}
		for j := i + 1; j <= p; j++ {
			acc -= ai[j] * xo[p+i-j]
		}
		for j := 1; j <= minip; j++ {
			acc -= ai[j] * w.pSamples[i-j+start]
		}
		w.whitened[i+start] = acc
	}
	// save last few frames of input
	for i := 0; i <= p; i++ {
		xo[i] = w.pSamples[blockSize-1-p+i+start]
	}
}
