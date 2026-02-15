package filter

import "math"

type ShelfType int

const (
	LowShelf ShelfType = iota
	HighShelf
)

type Biquad struct {
	b0, b1, b2 float64
	a1, a2     float64

	x1, x2 float64
	y1, y2 float64

	shelfType  ShelfType
	f0         float64
	sampleRate float64
}

func NewShelf(shelfType ShelfType, f0, gainDB, sampleRate float64) *Biquad {
	b := &Biquad{
		shelfType:  shelfType,
		f0:         f0,
		sampleRate: sampleRate,
	}
	b.computeCoefficients(gainDB)
	return b
}

func (b *Biquad) UpdateGain(gainDB float64) {
	b.computeCoefficients(gainDB)
}

// computeCoefficients uses Robert Bristow-Johnson Audio EQ Cookbook formulas with S=1.
func (b *Biquad) computeCoefficients(gainDB float64) {
	A := math.Pow(10, gainDB/40.0)
	w0 := 2 * math.Pi * b.f0 / b.sampleRate
	cosw0 := math.Cos(w0)
	sinw0 := math.Sin(w0)
	// S=1 slope: alpha = sin(w0)/2 * sqrt(2)
	alpha := sinw0 / 2 * math.Sqrt2

	var b0, b1, b2, a0, a1, a2 float64

	switch b.shelfType {
	case LowShelf:
		twoSqrtAAlpha := 2 * math.Sqrt(A) * alpha
		b0 = A * ((A + 1) - (A-1)*cosw0 + twoSqrtAAlpha)
		b1 = 2 * A * ((A - 1) - (A+1)*cosw0)
		b2 = A * ((A + 1) - (A-1)*cosw0 - twoSqrtAAlpha)
		a0 = (A + 1) + (A-1)*cosw0 + twoSqrtAAlpha
		a1 = -2 * ((A - 1) + (A+1)*cosw0)
		a2 = (A + 1) + (A-1)*cosw0 - twoSqrtAAlpha
	case HighShelf:
		twoSqrtAAlpha := 2 * math.Sqrt(A) * alpha
		b0 = A * ((A + 1) + (A-1)*cosw0 + twoSqrtAAlpha)
		b1 = -2 * A * ((A - 1) + (A+1)*cosw0)
		b2 = A * ((A + 1) + (A-1)*cosw0 - twoSqrtAAlpha)
		a0 = (A + 1) - (A-1)*cosw0 + twoSqrtAAlpha
		a1 = 2 * ((A - 1) - (A+1)*cosw0)
		a2 = (A + 1) - (A-1)*cosw0 - twoSqrtAAlpha
	}

	b.b0 = b0 / a0
	b.b1 = b1 / a0
	b.b2 = b2 / a0
	b.a1 = a1 / a0
	b.a2 = a2 / a0
}

func (b *Biquad) Process(samples []float64) {
	for i, x := range samples {
		y := b.b0*x + b.b1*b.x1 + b.b2*b.x2 - b.a1*b.y1 - b.a2*b.y2
		b.x2 = b.x1
		b.x1 = x
		b.y2 = b.y1
		b.y1 = y
		samples[i] = y
	}
}
