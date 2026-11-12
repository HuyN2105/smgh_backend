//
// Created by HuyN on 2026/10/17.
// Description: Định nghĩa các cấu trúc dữ liệu (struct) chuẩn cho gói tin cảm biến đầu vào (SensorPayload)
// và bản ghi lưu trữ nội bộ (SensorReading) với kiểu metrics động map[string]float64.
//

package telemetry

type SensorReading struct {
	DeviceID  string             `json:"device_id"`
	Timestamp string             `json:"timestamp"`
	Metrics   map[string]float64 `json:"metrics"`
}
