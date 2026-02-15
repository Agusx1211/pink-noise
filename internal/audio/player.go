package audio

import (
	"math"
	"time"

	oto "github.com/ebitengine/oto/v3"
)

type MixFunc func(samples int) []float64

type Player struct {
	context    *oto.Context
	player     *oto.Player
	sampleRate int
	bufferSize int
	stopChan   chan struct{}
}

func NewPlayer(sampleRate, bufferSize int) (*Player, error) {
	otoContext, readyChan, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: 2,
		Format:       oto.FormatFloat32LE,
		BufferSize:   500 * time.Millisecond,
	})
	if err != nil {
		return nil, err
	}

	<-readyChan

	return &Player{
		context:    otoContext,
		sampleRate: sampleRate,
		bufferSize: bufferSize,
		stopChan:   make(chan struct{}),
	}, nil
}

func (p *Player) Start(mixFn MixFunc) {
	p.player = p.context.NewPlayer(&mixReader{
		mixFn:      mixFn,
		bufferSize: p.bufferSize,
		stopChan:   p.stopChan,
	})
	p.player.Play()
}

func (p *Player) Stop() {
	close(p.stopChan)
	if p.player != nil {
		p.player.Pause()
	}
}

func (p *Player) Close() {
	p.Stop()
	if p.player != nil {
		p.player.Close()
	}
}

type mixReader struct {
	mixFn      MixFunc
	bufferSize int
	stopChan   <-chan struct{}
	buffer     []byte
	bufPos     int
}

func (r *mixReader) Read(buf []byte) (int, error) {
	totalRead := 0

	for totalRead < len(buf) {
		if r.bufPos >= len(r.buffer) {
			select {
			case <-r.stopChan:
				return totalRead, nil
			default:
			}

			samples := r.mixFn(r.bufferSize)
			r.buffer = float64ToBytes(samples)
			r.bufPos = 0
		}

		n := copy(buf[totalRead:], r.buffer[r.bufPos:])
		r.bufPos += n
		totalRead += n
	}

	return totalRead, nil
}

func float64ToBytes(samples []float64) []byte {
	result := make([]byte, len(samples)*4)

	for i, sample := range samples {
		clamped := math.Max(-1, math.Min(1, sample))
		bits := math.Float32bits(float32(clamped))
		result[i*4] = byte(bits)
		result[i*4+1] = byte(bits >> 8)
		result[i*4+2] = byte(bits >> 16)
		result[i*4+3] = byte(bits >> 24)
	}

	return result
}
