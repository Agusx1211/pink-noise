package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	MQTTBroker   string
	MQTTPort     int
	MQTTUser     string
	MQTTPassword string
	MQTTTopic    string
	SampleRate   int
	BufferSize   int
	StateFile    string
}

func Load() *Config {
	broker := getEnv("MQTT_BROKER", "localhost")
	if !strings.HasPrefix(broker, "tcp://") && !strings.HasPrefix(broker, "ssl://") {
		broker = "tcp://" + broker
	}

	cfg := &Config{
		MQTTBroker:   broker,
		MQTTPort:     getEnvInt("MQTT_PORT", 1883),
		MQTTUser:     getEnv("MQTT_USER", ""),
		MQTTPassword: getEnv("MQTT_PASSWORD", ""),
		MQTTTopic:    getEnv("MQTT_TOPIC", "homeassistant/noise"),
		SampleRate:   getEnvInt("SAMPLE_RATE", 44100),
		BufferSize:   getEnvInt("BUFFER_SIZE", 2048),
		StateFile:    getEnv("STATE_FILE", "/var/lib/pink-noise/state.json"),
	}

	log.Printf("Config: MQTT=%s:%d, Topic=%s", cfg.MQTTBroker, cfg.MQTTPort, cfg.MQTTTopic)
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
