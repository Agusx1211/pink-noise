# Pink Noise

[![Build](https://github.com/Agusx1211/pink-noise/actions/workflows/build.yml/badge.svg)](https://github.com/Agusx1211/pink-noise/actions/workflows/build.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A noise generator controllable via MQTT, designed for Home Assistant integration. Generates continuous noise across a color spectrum (brown through violet) with EQ controls and presets — great for sleep, focus, or soothing a baby.

## Features

- **Noise Color Spectrum**: Continuous slider blending between Brown, Pink, White, Blue, and Violet noise
- **EQ Controls**: Bass and treble shelf filters (-100 to +100)
- **Presets**: 11 built-in presets (Womb Sounds, Deep Sleep, Fan Noise, Pink Noise, etc.)
- **Home Assistant Integration**: Auto-discovery via MQTT — shows up as a device with sliders, switches, and presets
- **State Persistence**: Remembers power, volume, color, EQ, and preset across restarts
- **Smooth Transitions**: Volume changes fade smoothly to avoid clicks
- **Cross-Platform**: macOS (amd64/arm64) and Linux (amd64/arm64)
- **Docker Support**: Multi-stage Dockerfile included

## Building

```bash
# Build for current platform
make build

# Build macOS binaries (amd64 + arm64)
make build-mac

# Cross-compile Linux binaries via Docker (amd64 + arm64)
make docker-linux

# Build Docker image
make docker-build

# Clean build artifacts
make clean
```

## Configuration

Configuration is done via environment variables (see [`.env.example`](.env.example)):

| Variable | Default | Description |
|----------|---------|-------------|
| `MQTT_BROKER` | `localhost` | MQTT broker address (auto-prefixed with `tcp://` if needed) |
| `MQTT_PORT` | `1883` | MQTT broker port |
| `MQTT_USER` | | MQTT username |
| `MQTT_PASSWORD` | | MQTT password |
| `MQTT_TOPIC` | `homeassistant/noise` | MQTT topic prefix |
| `SAMPLE_RATE` | `44100` | Audio sample rate in Hz |
| `BUFFER_SIZE` | `2048` | Audio buffer size in samples |
| `STATE_FILE` | `/var/lib/pink-noise/state.json` | Path for persisted state |

## Running

```bash
# Run directly
make run

# Or after building
./build/pink-noise

# With Docker Compose
docker compose up -d
```

## Home Assistant Integration

The player registers itself via MQTT discovery as a **Pink Noise Generator** device with 7 entities:

| Entity | Type | Description |
|--------|------|-------------|
| Power | Switch | On/Off toggle |
| Volume | Number (0–100) | Master volume percentage |
| Preset | Select | Choose from 11 built-in presets |
| Color | Number (0–100) | Noise color slider: 0=Brown, 25=Pink, 50=White, 75=Blue, 100=Violet |
| Bass | Number (-100–100) | Low shelf EQ filter at 300 Hz |
| Treble | Number (-100–100) | High shelf EQ filter at 3 kHz |
| Stop All | Button | Turn off the player |

### MQTT Topics

All topics are under the configured prefix (default `homeassistant/noise`):

| Topic | Payload | Direction |
|-------|---------|-----------|
| `<prefix>/power/set` | `ON` / `OFF` | Command |
| `<prefix>/volume/set` | `0`–`100` | Command |
| `<prefix>/preset/set` | Preset name | Command |
| `<prefix>/color/set` | `0`–`100` | Command |
| `<prefix>/bass/set` | `-100`–`100` | Command |
| `<prefix>/treble/set` | `-100`–`100` | Command |
| `<prefix>/stop_all/set` | Any | Command |
| `<prefix>/state` | JSON | State (published) |
| `<prefix>/availability` | `online` / `offline` | Availability |

### Presets

| Name | Color | Bass | Treble |
|------|-------|------|--------|
| Womb Sounds | 5 | 80 | -60 |
| Deep Sleep | 12 | 50 | -40 |
| Shushing | 30 | -20 | 30 |
| Fan Noise | 45 | 40 | -10 |
| Gentle Rain | 25 | 10 | -20 |
| Light Sleep | 35 | 0 | -30 |
| Calming Wash | 20 | 30 | -50 |
| Bright Comfort | 55 | -10 | 20 |
| Brown Noise | 0 | 0 | 0 |
| Pink Noise | 25 | 0 | 0 |
| White Noise | 50 | 0 | 0 |

### Example Automation

```yaml
automation:
  - alias: "Baby sleep noise"
    trigger:
      - platform: time
        at: "20:00:00"
    action:
      - service: mqtt.publish
        data:
          topic: homeassistant/noise/preset/set
          payload: "Deep Sleep"
      - service: mqtt.publish
        data:
          topic: homeassistant/noise/volume/set
          payload: "40"
      - service: mqtt.publish
        data:
          topic: homeassistant/noise/power/set
          payload: "ON"
```

## Project Structure

```
.
├── cmd/pink-noise/main.go       # Entry point, state persistence, command loop
├── internal/
│   ├── audio/player.go          # Audio output (oto v3, float32 LE stereo)
│   ├── config/config.go         # Environment variable configuration
│   ├── filter/biquad.go         # Biquad shelf EQ filters
│   ├── mixer/mixer.go           # Audio mixer with volume smoothing and EQ
│   ├── mqtt/client.go           # MQTT client, HA discovery, presets
│   └── noise/generator.go       # Noise color generation and blending
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── go.mod
```

## Dependencies

- [oto v3](https://github.com/ebitengine/oto) — Cross-platform audio output
- [paho.mqtt.golang](https://github.com/eclipse/paho.mqtt.golang) — MQTT client

## License

[MIT](LICENSE)
