package mqtt

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// TelemetryPayload represents the JSON object published to the MQTT broker
type TelemetryPayload struct {
	ActiveBees    int     `json:"activeBees"`
	CountUp       int     `json:"countUp"`
	CountDown     int     `json:"countDown"`
	AvgPathLength float64 `json:"avgPathLength"`
	AvgSpeed      float64 `json:"avgSpeed"`
	AvgDistance   float64 `json:"avgDistance"`
	FPS           float64 `json:"fps"`
	Timestamp     string  `json:"timestamp"`
}

// Client wraps the MQTT client connection
type Client struct {
	client paho.Client
	topic  string
}

// getEnv returns environment variable value or default if not found
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

// NewClient initializes the MQTT client and connects asynchronously
func NewClient() (*Client, error) {
	broker := getEnv("MQTT_BROKER", "tcp://localhost:1883")
	topic := getEnv("MQTT_TOPIC", "beesmart/telemetry")
	clientID := getEnv("MQTT_CLIENT_ID", "beesmart-video-publisher")

	opts := paho.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)

	client := paho.NewClient(opts)

	// Connect asynchronously so that video app doesn't hang if broker is offline
	go func() {
		token := client.Connect()
		if token.Wait() && token.Error() != nil {
			fmt.Printf("\n[MQTT Warning] Connection failed (will auto-reconnect): %v\n", token.Error())
		} else {
			fmt.Printf("\n[MQTT Info] Connected to broker at %s\n", broker)
		}
	}()

	return &Client{
		client: client,
		topic:  topic,
	}, nil
}

// PublishStats sends the given stats to the MQTT channel
func (c *Client) PublishStats(active, up, down int, avgPath, avgSpeed, avgDist, fps float64) {
	payload := TelemetryPayload{
		ActiveBees:    active,
		CountUp:       up,
		CountDown:     down,
		AvgPathLength: avgPath,
		AvgSpeed:      avgSpeed,
		AvgDistance:   avgDist,
		FPS:           fps,
		Timestamp:     time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("\n[MQTT Error] Marshal error: %v\n", err)
		return
	}

	// Publish asynchronously
	token := c.client.Publish(c.topic, 0, false, data)
	go func() {
		if token.Wait() && token.Error() != nil {
			fmt.Printf("\n[MQTT Error] Publish failed: %v\n", token.Error())
		}
	}()
}
