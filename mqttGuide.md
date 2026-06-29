# BeeSmartMQTT

This repo is a minimal MQTT publishing example plus a local Mosquitto broker. It is meant to be reused as a handoff document for another agent or another application that needs to publish messages to MQTT on a different topic.

## What Runs Here

- A Mosquitto broker on `localhost:1883`
- A Go publisher that connects to that broker and publishes every 5 seconds
- Anonymous access is enabled in the broker config, so no username or password is required for local testing

## Broker Configuration

The broker is defined in [docker-compose.yml](/home/emilio/Documents/BeeSmartMQTT/docker-compose.yml) and [mosquitto.conf](/home/emilio/Documents/BeeSmartMQTT/mosquitto.conf).

```yaml
services:
  mqtt:
    image: eclipse-mosquitto:2
    container_name: beesmart-mqtt
    ports:
      - "1883:1883"
    volumes:
      - ./mosquitto.conf:/mosquitto/config/mosquitto.conf:ro
```

```conf
listener 1883
allow_anonymous true
persistence false
```

Notes:
- `listener 1883` exposes the standard MQTT TCP port.
- `allow_anonymous true` allows clients to connect without credentials.
- `persistence false` keeps this broker disposable and simple for local testing.

## Go Publisher Behavior

The Go app is intentionally small. It shows the basic pattern another program can copy:

```go
broker := getenv("MQTT_BROKER", "tcp://localhost:1883")
topic := getenv("MQTT_TOPIC", "test/topic")
clientID := getenv("MQTT_CLIENT_ID", "go-mqtt-publisher")
```

```go
opts := mqtt.NewClientOptions()
opts.AddBroker(broker)
opts.SetClientID(clientID)
opts.SetAutoReconnect(true)
opts.SetConnectRetry(true)
```

```go
message := fmt.Sprintf("Hello MQTT #%d at %s", counter, time.Now().Format(time.RFC3339))
token := client.Publish(topic, 0, false, message)
```

The full program lives in [main.go](/home/emilio/Documents/BeeSmartMQTT/main.go).

## How Another Application Should Use This

To send messages from a different app:

1. Point the client at `tcp://localhost:1883` or whatever broker host you choose.
2. Use a different topic by setting `MQTT_TOPIC` or by hardcoding a new topic in your own client.
3. Keep `MQTT_CLIENT_ID` unique per running client.
4. Leave `MQTT_USERNAME` and `MQTT_PASSWORD` empty unless you change the broker to require auth.
5. Publish the payload format that your application needs; the broker does not care about the message body.

Example environment values for another publisher:

```bash
MQTT_BROKER=tcp://localhost:1883
MQTT_TOPIC=plants/bed1/moisture
MQTT_CLIENT_ID=irrigation-sensor-publisher
```

Example payload from another app:

```text
{"sensor":"bed1","moisture":42,"unit":"percent"}
```

## Start the Broker

Use the installed Docker Compose binary:

```bash
docker-compose up -d mqtt
```

The broker listens on `localhost:1883`.

## Run the Go Publisher

```bash
go run .
```

The default publish target is `test/topic`, but you can override it with `MQTT_TOPIC`.

## Watch Messages

Subscribe from another terminal:

```bash
docker-compose exec mqtt mosquitto_sub -h localhost -t test/topic -v
```

For a different topic, replace `test/topic` with the topic your other app uses.

## Quick Checklist For a New Agent

- Start the broker with `docker-compose up -d mqtt`
- Pick a topic that matches the new application
- Set `MQTT_BROKER`, `MQTT_TOPIC`, and `MQTT_CLIENT_ID`
- Publish a test message
- Subscribe to the same topic to verify delivery
