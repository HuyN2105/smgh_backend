package telemetry

type SensorReading struct {
	DeviceID  string             `json:"device_id"`
	Timestamp string             `json:"timestamp"`
	Metrics   map[string]float64 `json:"metrics"`
}
