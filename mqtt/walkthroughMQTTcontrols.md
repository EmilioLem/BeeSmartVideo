# Walkthrough - MQTT Telemetry Integration

We have successfully implemented MQTT telemetry support in `BeeSmartVideo`. Telemetry publishes processing metrics in real time (Active Bee counts, Flow directions, average speeds, etc.) to an MQTT broker.

## Changes Made

### 1. MQTT Package
- [NEW] [mqtt.go](file:///home/emilio/Documents/BeeSmartVideo/mqtt/mqtt.go)
  - Added a non-blocking MQTT client wrapper.
  - Formats metrics into a JSON structure matching the web dashboard layout.
  - Establishes connection asynchronously (in a background goroutine) to ensure the application starts up and processes frames even if the MQTT broker is offline.
  - Dispatches telemetry payloads in an asynchronous goroutine to prevent frame rate drops.

### 2. Interactive TUI Option
- [MODIFY] [menu.go](file:///home/emilio/Documents/BeeSmartVideo/menu/menu.go)
  - Added `EnableTelemetry` parameter to the `Options` configuration.
  - Included a new interactive confirmation option `"Enable MQTT Telemetry"` using `huh`.
  - Configured `loadSettings` and `saveSettings` to correctly load/save this field in `settings.json`.

### 3. Application Integration
- [MODIFY] [main.go](file:///home/emilio/Documents/BeeSmartVideo/main.go)
  - Imported the new `BeeSmartVideo/mqtt` package.
  - Conditioned MQTT client initialization on the `EnableTelemetry` flag.
  - Injected telemetry publishing within the 10-frame logic callback to keep the data in sync with the web dashboard stats.

---

## How to Test

### 1. Configure the Broker Connection
By default, the MQTT package reads the broker location from environment variables. If they are not specified, it falls back to standard local testing defaults:
- `MQTT_BROKER` (Default: `tcp://localhost:1883`)
- `MQTT_TOPIC` (Default: `beesmart/telemetry`)
- `MQTT_CLIENT_ID` (Default: `beesmart-video-publisher`)

### 2. Start a Local MQTT Broker
If you have Docker installed, you can launch a local broker in one line:
```bash
docker run -d --name mosquitto -p 1883:1883 eclipse-mosquitto:2 sh -c "echo 'listener 1883\nallow_anonymous true' > /mosquitto/config/mosquitto.conf && exec /usr/sbin/mosquitto -c /mosquitto/config/mosquitto.conf"
```

### 3. Subscribe to the Telemetry Topic
Open a second terminal window to listen for incoming telemetry messages:
```bash
docker exec -it mosquitto mosquitto_sub -h localhost -t beesmart/telemetry -v
```

### 4. Run BeeSmartVideo
Run the application and toggle telemetry to `Yes`:
```bash
go run main.go
```
Once the video starts processing, you will see telemetry JSON payloads matching the format below being published:
```json
{
  "activeBees": 5,
  "countUp": 12,
  "countDown": 7,
  "avgPathLength": 45.2,
  "avgSpeed": 3.4,
  "avgDistance": 88.1,
  "fps": 29.8,
  "timestamp": "2026-06-22T15:40:00-06:00"
}
```
