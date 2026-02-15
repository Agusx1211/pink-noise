package mixer

import (
	"math"
	"sync"

	"github.com/agusx1211/pink-noise/internal/filter"
	"github.com/agusx1211/pink-noise/internal/noise"
)

type Mixer struct {
	mu sync.RWMutex

	noiseGen   *noise.Generator
	sampleRate int

	power        bool
	masterVolume float64
	targetVolume float64
	colorSlider  float64
	bassGain     float64
	trebleGain   float64

	lowShelfL  *filter.Biquad
	lowShelfR  *filter.Biquad
	highShelfL *filter.Biquad
	highShelfR *filter.Biquad
}

func NewMixer(sampleRate int) *Mixer {
	return &Mixer{
		noiseGen:     noise.NewGenerator(sampleRate),
		sampleRate:   sampleRate,
		power:        false,
		masterVolume: 0.5,
		targetVolume: 0.5,
		colorSlider:  25, // pink noise default
		lowShelfL:    filter.NewShelf(filter.LowShelf, 300, 0, float64(sampleRate)),
		lowShelfR:    filter.NewShelf(filter.LowShelf, 300, 0, float64(sampleRate)),
		highShelfL:   filter.NewShelf(filter.HighShelf, 3000, 0, float64(sampleRate)),
		highShelfR:   filter.NewShelf(filter.HighShelf, 3000, 0, float64(sampleRate)),
	}
}

func (m *Mixer) SetPower(on bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.power = on
}

func (m *Mixer) GetPower() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.power
}

func (m *Mixer) SetMasterVolume(volume float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.targetVolume = math.Max(0, math.Min(1, volume))
}

func (m *Mixer) GetMasterVolume() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.targetVolume
}

func (m *Mixer) SetColor(value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.colorSlider = math.Max(0, math.Min(100, value))
}

func (m *Mixer) GetColor() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.colorSlider
}

// sliderToGainDB maps -100..+100 slider to -12..+12 dB
func sliderToGainDB(slider float64) float64 {
	return slider / 100.0 * 12.0
}

func (m *Mixer) SetBass(value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bassGain = math.Max(-100, math.Min(100, value))
	gainDB := sliderToGainDB(m.bassGain)
	m.lowShelfL.UpdateGain(gainDB)
	m.lowShelfR.UpdateGain(gainDB)
}

func (m *Mixer) GetBass() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bassGain
}

func (m *Mixer) SetTreble(value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trebleGain = math.Max(-100, math.Min(100, value))
	gainDB := sliderToGainDB(m.trebleGain)
	m.highShelfL.UpdateGain(gainDB)
	m.highShelfR.UpdateGain(gainDB)
}

func (m *Mixer) GetTreble() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.trebleGain
}

func (m *Mixer) ReseedRNG(seed int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.noiseGen.Reseed(seed)
}

func (m *Mixer) Mix(samples int) []float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]float64, samples*2)

	if !m.power {
		// Smooth volume to zero when off
		for i := range samples {
			m.masterVolume += (0 - m.masterVolume) * 0.001
			_ = i
		}
		return result
	}

	// Generate blended noise (mono)
	mono := m.noiseGen.GenerateBlended(m.colorSlider, samples, 1.0)

	// Split to L/R for independent filter state
	left := make([]float64, samples)
	right := make([]float64, samples)
	copy(left, mono)
	copy(right, mono)

	// Apply EQ filters
	m.lowShelfL.Process(left)
	m.highShelfL.Process(left)
	m.lowShelfR.Process(right)
	m.highShelfR.Process(right)

	// Apply volume with smoothing
	for i := range samples {
		m.masterVolume += (m.targetVolume - m.masterVolume) * 0.001
		result[i*2] = math.Max(-1, math.Min(1, left[i]*m.masterVolume))
		result[i*2+1] = math.Max(-1, math.Min(1, right[i]*m.masterVolume))
	}

	return result
}
