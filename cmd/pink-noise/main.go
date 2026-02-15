package main

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/agusx1211/pink-noise/internal/audio"
	"github.com/agusx1211/pink-noise/internal/config"
	"github.com/agusx1211/pink-noise/internal/mixer"
	"github.com/agusx1211/pink-noise/internal/mqtt"
)

type PersistedState struct {
	MasterVolume float64 `json:"master_volume"`
	Color        float64 `json:"color"`
	Bass         float64 `json:"bass"`
	Treble       float64 `json:"treble"`
	Preset       string  `json:"preset"`
	Power        bool    `json:"power"`
}

func main() {
	cfg := config.Load()

	m := mixer.NewMixer(cfg.SampleRate)

	restoreState(m, cfg.StateFile)

	player, err := audio.NewPlayer(cfg.SampleRate, cfg.BufferSize)
	if err != nil {
		log.Fatalf("Failed to create audio player: %v", err)
	}
	defer player.Close()

	commandChan := make(chan mqtt.Command, 100)

	mqttClient, err := mqtt.NewClient(
		cfg.MQTTBroker,
		cfg.MQTTPort,
		cfg.MQTTUser,
		cfg.MQTTPassword,
		cfg.MQTTTopic,
		m,
		commandChan,
	)
	if err != nil {
		log.Fatalf("Failed to create MQTT client: %v", err)
	}
	defer mqttClient.Close()

	player.Start(m.Mix)

	go reseedLoop(m)
	go processCommands(m, commandChan, mqttClient, cfg.StateFile)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}

func reseedLoop(m *mixer.Mixer) {
	reseed(m)
	for range time.Tick(10 * time.Minute) {
		reseed(m)
	}
}

func reseed(m *mixer.Mixer) {
	f, err := os.Open("/dev/random")
	if err != nil {
		log.Printf("Failed to open /dev/random: %v", err)
		return
	}
	defer f.Close()

	var seed int64
	if err := binary.Read(f, binary.LittleEndian, &seed); err != nil {
		log.Printf("Failed to read /dev/random: %v", err)
		return
	}
	m.ReseedRNG(seed)
	log.Printf("Re-seeded RNG from /dev/random")
}

func processCommands(m *mixer.Mixer, cmdChan <-chan mqtt.Command, mqttClient *mqtt.Client, stateFile string) {
	stateTicker := time.NewTicker(2 * time.Second)
	defer stateTicker.Stop()

	for {
		select {
		case cmd, ok := <-cmdChan:
			if !ok {
				return
			}
			switch cmd.Action {
			case "set_power_on":
				m.SetPower(true)
			case "set_power_off":
				m.SetPower(false)
			case "set_volume":
				m.SetMasterVolume(cmd.Value)
			case "set_color":
				m.SetColor(cmd.Value)
				mqtt.CurrentPreset = "Custom"
			case "set_bass":
				m.SetBass(cmd.Value)
				mqtt.CurrentPreset = "Custom"
			case "set_treble":
				m.SetTreble(cmd.Value)
				mqtt.CurrentPreset = "Custom"
			case "set_preset":
				if p := mqtt.FindPreset(cmd.Preset); p != nil {
					m.SetColor(p.Color)
					m.SetBass(p.Bass)
					m.SetTreble(p.Treble)
					mqtt.CurrentPreset = p.Name
				}
			case "stop_all":
				m.SetPower(false)
			}
			saveState(m, stateFile)
			mqttClient.PublishState()
		case <-stateTicker.C:
			mqttClient.PublishState()
		}
	}
}

func saveState(m *mixer.Mixer, path string) {
	state := PersistedState{
		MasterVolume: m.GetMasterVolume(),
		Color:        m.GetColor(),
		Bass:         m.GetBass(),
		Treble:       m.GetTreble(),
		Preset:       mqtt.CurrentPreset,
		Power:        m.GetPower(),
	}

	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("Failed to marshal state: %v", err)
		return
	}

	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("Failed to save state: %v", err)
	}
}

func restoreState(m *mixer.Mixer, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var state PersistedState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("Failed to parse saved state: %v", err)
		return
	}

	m.SetMasterVolume(state.MasterVolume)
	m.SetColor(state.Color)
	m.SetBass(state.Bass)
	m.SetTreble(state.Treble)
	m.SetPower(state.Power)

	if state.Preset != "" {
		mqtt.CurrentPreset = state.Preset
	}

	log.Printf("Restored state: power=%v, volume=%.0f%%, color=%.0f, preset=%s",
		state.Power, state.MasterVolume*100, state.Color, state.Preset)
}
