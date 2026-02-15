package noise

import (
	"math/rand"
)

type Color string

const (
	White  Color = "white"
	Pink   Color = "pink"
	Brown  Color = "brown"
	Blue   Color = "blue"
	Violet Color = "violet"
)

type Generator struct {
	sampleRate      int
	rng             *rand.Rand
	white           float64
	pink            [7]float64
	brown           float64
	bluePrev        float64
	violetPrevWhite float64
	violetPrevBlue  float64

	// Second generator state for blending
	pink2            [7]float64
	brown2           float64
	bluePrev2        float64
	violetPrevWhite2 float64
	violetPrevBlue2  float64
}

func NewGenerator(sampleRate int) *Generator {
	return &Generator{
		sampleRate: sampleRate,
		rng:        rand.New(rand.NewSource(rand.Int63())),
	}
}

// Reseed replaces the internal RNG with a new one seeded from the given value.
// Caller must ensure this is not called concurrently with Generate/GenerateBlended.
func (g *Generator) Reseed(seed int64) {
	g.rng = rand.New(rand.NewSource(seed))
}

// colorAnchors maps slider position to Color: 0=Brown, 25=Pink, 50=White, 75=Blue, 100=Violet
var colorAnchors = []struct {
	pos   float64
	color Color
}{
	{0, Brown},
	{25, Pink},
	{50, White},
	{75, Blue},
	{100, Violet},
}

// GenerateBlended maps a 0-100 color slider to two adjacent noise colors
// and crossfades between them.
func (g *Generator) GenerateBlended(colorSlider float64, samples int, volume float64) []float64 {
	if colorSlider <= 0 {
		return g.generateColor(Brown, true, samples, volume)
	}
	if colorSlider >= 100 {
		return g.generateColor(Violet, true, samples, volume)
	}

	for i := 0; i < len(colorAnchors)-1; i++ {
		lo := colorAnchors[i]
		hi := colorAnchors[i+1]
		if colorSlider >= lo.pos && colorSlider <= hi.pos {
			t := (colorSlider - lo.pos) / (hi.pos - lo.pos)
			if t <= 0.001 {
				return g.generateColor(lo.color, true, samples, volume)
			}
			if t >= 0.999 {
				return g.generateColor(hi.color, true, samples, volume)
			}
			samplesA := g.generateColor(lo.color, true, samples, volume)
			samplesB := g.generateColor(hi.color, false, samples, volume)
			result := make([]float64, samples)
			for j := 0; j < samples; j++ {
				result[j] = samplesA[j]*(1-t) + samplesB[j]*t
			}
			return result
		}
	}

	return g.generateColor(White, true, samples, volume)
}

func (g *Generator) generateColor(color Color, primary bool, samples int, volume float64) []float64 {
	switch color {
	case White:
		return g.generateWhite(samples, volume)
	case Pink:
		if primary {
			return g.generatePinkState(&g.pink, samples, volume)
		}
		return g.generatePinkState(&g.pink2, samples, volume)
	case Brown:
		if primary {
			return g.generateBrownState(&g.brown, samples, volume)
		}
		return g.generateBrownState(&g.brown2, samples, volume)
	case Blue:
		if primary {
			return g.generateBlueState(&g.bluePrev, samples, volume)
		}
		return g.generateBlueState(&g.bluePrev2, samples, volume)
	case Violet:
		if primary {
			return g.generateVioletState(&g.violetPrevWhite, &g.violetPrevBlue, samples, volume)
		}
		return g.generateVioletState(&g.violetPrevWhite2, &g.violetPrevBlue2, samples, volume)
	default:
		return g.generateWhite(samples, volume)
	}
}

func (g *Generator) Generate(color Color, samples int, volume float64) []float64 {
	return g.generateColor(color, true, samples, volume)
}

func (g *Generator) generateWhite(samples int, volume float64) []float64 {
	result := make([]float64, samples)
	for i := range samples {
		result[i] = (g.rng.Float64()*2 - 1) * volume
	}
	return result
}

func (g *Generator) generatePinkState(state *[7]float64, samples int, volume float64) []float64 {
	result := make([]float64, samples)
	coeffs := [7]float64{0.1294, 0.1875, 0.2414, 0.3026, 0.3830, 0.4962, 0.7195}

	for i := range samples {
		white := g.rng.Float64()*2 - 1
		state[0] = coeffs[0]*(white-state[0]) + state[0]
		state[1] = coeffs[1]*(white-state[1]) + state[1]
		state[2] = coeffs[2]*(white-state[2]) + state[2]
		state[3] = coeffs[3]*(white-state[3]) + state[3]
		state[4] = coeffs[4]*(white-state[4]) + state[4]
		state[5] = coeffs[5]*(white-state[5]) + state[5]
		state[6] = coeffs[6]*(white-state[6]) + state[6]

		pink := (state[0] + state[1] + state[2] + state[3] + state[4] + state[5] + state[6]) / 2.5
		result[i] = pink * volume
	}
	return result
}

func (g *Generator) generateBrownState(state *float64, samples int, volume float64) []float64 {
	result := make([]float64, samples)
	for i := range samples {
		white := g.rng.Float64()*2 - 1
		*state = (*state + 0.02*white) / 1.02
		result[i] = *state * 3.5 * volume
	}
	return result
}

func (g *Generator) generateBlueState(prev *float64, samples int, volume float64) []float64 {
	result := make([]float64, samples)
	for i := range samples {
		white := g.rng.Float64()*2 - 1
		result[i] = (white - *prev) * volume
		*prev = white
	}
	return result
}

func (g *Generator) generateVioletState(prevWhite, prevBlue *float64, samples int, volume float64) []float64 {
	result := make([]float64, samples)
	for i := range samples {
		white := g.rng.Float64()*2 - 1
		blue := white - *prevWhite
		result[i] = (blue - *prevBlue) * volume
		*prevBlue = blue
		*prevWhite = white
	}
	return result
}

type NoiseParams struct {
	ColorSlider float64
	Bass        float64
	Treble      float64
	Volume      float64
}
