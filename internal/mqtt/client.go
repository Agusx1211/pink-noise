package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/agusx1211/pink-noise/internal/mixer"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Preset struct {
	Name   string
	Color  float64
	Bass   float64
	Treble float64
}

var Presets = []Preset{
	{"Womb Sounds", 5, 80, -60},
	{"Deep Sleep", 12, 50, -40},
	{"Shushing", 30, -20, 30},
	{"Fan Noise", 45, 40, -10},
	{"Gentle Rain", 25, 10, -20},
	{"Light Sleep", 35, 0, -30},
	{"Calming Wash", 20, 30, -50},
	{"Bright Comfort", 55, -10, 20},
	{"Brown Noise", 0, 0, 0},
	{"Pink Noise", 25, 0, 0},
	{"White Noise", 50, 0, 0},
}

type Client struct {
	client      mqtt.Client
	topic       string
	mixer       *mixer.Mixer
	commandChan chan<- Command
}

type Command struct {
	Action string
	Value  float64
	Preset string
}

func NewClient(broker string, port int, user, password, topic string, m *mixer.Mixer, cmdChan chan<- Command) (*Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s:%d", broker, port))
	opts.SetClientID(fmt.Sprintf("pink-noise-%d", time.Now().Unix()))

	if user != "" {
		opts.SetUsername(user)
	}
	if password != "" {
		opts.SetPassword(password)
	}

	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(30 * time.Second)

	c := &Client{
		topic:       topic,
		mixer:       m,
		commandChan: cmdChan,
	}

	opts.OnConnect = c.onConnect
	opts.OnConnectionLost = c.onConnectionLost
	opts.SetWill(topic+"/availability", "offline", 0, true)

	c.client = mqtt.NewClient(opts)
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return c, nil
}

func (c *Client) onConnect(client mqtt.Client) {
	log.Println("Connected to MQTT broker")

	client.Publish(c.topic+"/availability", 0, true, "online")

	subs := map[string]mqtt.MessageHandler{
		c.topic + "/power/set":    c.handlePower,
		c.topic + "/volume/set":   c.handleVolume,
		c.topic + "/preset/set":   c.handlePreset,
		c.topic + "/color/set":    c.handleColor,
		c.topic + "/bass/set":     c.handleBass,
		c.topic + "/treble/set":   c.handleTreble,
		c.topic + "/stop_all/set": c.handleStopAll,
	}

	for topic, handler := range subs {
		if token := client.Subscribe(topic, 0, handler); token.Wait() && token.Error() != nil {
			log.Printf("Failed to subscribe to %s: %v", topic, token.Error())
		}
	}

	c.cleanupOldEntities()
	c.publishDiscovery()
	c.PublishState()
}

func (c *Client) onConnectionLost(client mqtt.Client, err error) {
	log.Printf("MQTT connection lost: %v", err)
}

func (c *Client) handlePower(client mqtt.Client, msg mqtt.Message) {
	payload := strings.TrimSpace(string(msg.Payload()))
	action := "set_power_off"
	if payload == "ON" {
		action = "set_power_on"
	}
	c.sendCommand(Command{Action: action})
}

func (c *Client) handleVolume(client mqtt.Client, msg mqtt.Message) {
	v, err := strconv.ParseFloat(strings.TrimSpace(string(msg.Payload())), 64)
	if err != nil {
		return
	}
	c.sendCommand(Command{Action: "set_volume", Value: v / 100.0})
}

func (c *Client) handlePreset(client mqtt.Client, msg mqtt.Message) {
	name := strings.TrimSpace(string(msg.Payload()))
	c.sendCommand(Command{Action: "set_preset", Preset: name})
}

func (c *Client) handleColor(client mqtt.Client, msg mqtt.Message) {
	v, err := strconv.ParseFloat(strings.TrimSpace(string(msg.Payload())), 64)
	if err != nil {
		return
	}
	c.sendCommand(Command{Action: "set_color", Value: v})
}

func (c *Client) handleBass(client mqtt.Client, msg mqtt.Message) {
	v, err := strconv.ParseFloat(strings.TrimSpace(string(msg.Payload())), 64)
	if err != nil {
		return
	}
	c.sendCommand(Command{Action: "set_bass", Value: v})
}

func (c *Client) handleTreble(client mqtt.Client, msg mqtt.Message) {
	v, err := strconv.ParseFloat(strings.TrimSpace(string(msg.Payload())), 64)
	if err != nil {
		return
	}
	c.sendCommand(Command{Action: "set_treble", Value: v})
}

func (c *Client) handleStopAll(client mqtt.Client, msg mqtt.Message) {
	c.sendCommand(Command{Action: "stop_all"})
}

func (c *Client) sendCommand(cmd Command) {
	select {
	case c.commandChan <- cmd:
	default:
		log.Println("Command channel full")
	}
}

func (c *Client) publishDiscovery() {
	device := map[string]interface{}{
		"identifiers":  []string{"pink_noise_generator"},
		"name":         "Pink Noise Generator",
		"manufacturer": "Pink Noise",
		"model":        "Noise Player",
	}

	availability := map[string]interface{}{
		"topic": c.topic + "/availability",
	}

	// Power switch
	c.publishEntity("switch", "pink_noise_power", map[string]interface{}{
		"name":            "Power",
		"unique_id":       "pink_noise_power",
		"device":          device,
		"availability":    availability,
		"command_topic":   c.topic + "/power/set",
		"state_topic":     c.topic + "/state",
		"value_template":  "{% if value_json.power %}ON{% else %}OFF{% endif %}",
		"payload_on":      "ON",
		"payload_off":     "OFF",
		"icon":            "mdi:power",
	})

	// Volume number
	c.publishEntity("number", "pink_noise_volume", map[string]interface{}{
		"name":                "Volume",
		"unique_id":           "pink_noise_volume",
		"device":              device,
		"availability":        availability,
		"command_topic":       c.topic + "/volume/set",
		"state_topic":         c.topic + "/state",
		"value_template":      "{{ (value_json.volume * 100) | round(0) }}",
		"min":                 0,
		"max":                 100,
		"step":                1,
		"unit_of_measurement": "%",
		"icon":                "mdi:volume-high",
	})

	// Preset select
	presetOptions := make([]string, 0, len(Presets)+1)
	for _, p := range Presets {
		presetOptions = append(presetOptions, p.Name)
	}
	presetOptions = append(presetOptions, "Custom")

	c.publishEntity("select", "pink_noise_preset", map[string]interface{}{
		"name":           "Preset",
		"unique_id":      "pink_noise_preset",
		"device":         device,
		"availability":   availability,
		"command_topic":  c.topic + "/preset/set",
		"state_topic":    c.topic + "/state",
		"value_template": "{{ value_json.preset }}",
		"options":        presetOptions,
		"icon":           "mdi:baby-face",
	})

	// Color slider
	c.publishEntity("number", "pink_noise_color", map[string]interface{}{
		"name":           "Color",
		"unique_id":      "pink_noise_color",
		"device":         device,
		"availability":   availability,
		"command_topic":  c.topic + "/color/set",
		"state_topic":    c.topic + "/state",
		"value_template": "{{ value_json.color | round(0) }}",
		"min":            0,
		"max":            100,
		"step":           1,
		"icon":           "mdi:palette",
	})

	// Bass slider
	c.publishEntity("number", "pink_noise_bass", map[string]interface{}{
		"name":           "Bass",
		"unique_id":      "pink_noise_bass",
		"device":         device,
		"availability":   availability,
		"command_topic":  c.topic + "/bass/set",
		"state_topic":    c.topic + "/state",
		"value_template": "{{ value_json.bass | round(0) }}",
		"min":            -100,
		"max":            100,
		"step":           1,
		"icon":           "mdi:music-clef-bass",
	})

	// Treble slider
	c.publishEntity("number", "pink_noise_treble", map[string]interface{}{
		"name":           "Treble",
		"unique_id":      "pink_noise_treble",
		"device":         device,
		"availability":   availability,
		"command_topic":  c.topic + "/treble/set",
		"state_topic":    c.topic + "/state",
		"value_template": "{{ value_json.treble | round(0) }}",
		"min":            -100,
		"max":            100,
		"step":           1,
		"icon":           "mdi:music-clef-treble",
	})

	// Stop All button
	c.publishEntity("button", "pink_noise_stop_all", map[string]interface{}{
		"name":          "Stop All",
		"unique_id":     "pink_noise_stop_all",
		"device":        device,
		"availability":  availability,
		"command_topic": c.topic + "/stop_all/set",
		"icon":          "mdi:stop",
	})

	log.Println("Published MQTT discovery (7 entities)")
}

func (c *Client) publishEntity(domain, entityID string, config map[string]interface{}) {
	data, _ := json.Marshal(config)
	topic := fmt.Sprintf("homeassistant/%s/%s/config", domain, entityID)
	if token := c.client.Publish(topic, 0, true, data); token.Wait() && token.Error() != nil {
		log.Printf("Failed to publish discovery for %s: %v", entityID, token.Error())
	}
}

// cleanupOldEntities removes old HA entities from the previous per-color layout
// by publishing empty config to their discovery topics.
func (c *Client) cleanupOldEntities() {
	oldSoundTypes := []string{"white", "pink", "brown", "blue", "violet"}
	for _, soundType := range oldSoundTypes {
		entityID := fmt.Sprintf("pink_noise_%s", soundType)
		c.publishEntity("number", entityID+"_volume", nil)
		c.publishEntity("switch", entityID+"_switch", nil)
	}
	// Also clean old master volume entity
	c.publishEntity("number", "pink_noise_master_volume", nil)
	log.Println("Cleaned up old HA entities")
}

type publishedState struct {
	Power  bool    `json:"power"`
	Volume float64 `json:"volume"`
	Preset string  `json:"preset"`
	Color  float64 `json:"color"`
	Bass   float64 `json:"bass"`
	Treble float64 `json:"treble"`
}

// CurrentPreset is maintained by main.go and passed here for state publishing.
var CurrentPreset string = "Custom"

func (c *Client) PublishState() {
	state := publishedState{
		Power:  c.mixer.GetPower(),
		Volume: c.mixer.GetMasterVolume(),
		Preset: CurrentPreset,
		Color:  c.mixer.GetColor(),
		Bass:   c.mixer.GetBass(),
		Treble: c.mixer.GetTreble(),
	}

	data, _ := json.Marshal(state)
	c.client.Publish(c.topic+"/state", 0, true, data)
}

func (c *Client) Close() {
	if c.client != nil {
		c.client.Publish(c.topic+"/availability", 0, true, "offline")
		c.client.Disconnect(250)
	}
}

func FindPreset(name string) *Preset {
	for i := range Presets {
		if Presets[i].Name == name {
			return &Presets[i]
		}
	}
	return nil
}
