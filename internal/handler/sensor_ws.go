package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"CentralServer/internal/auth"
	"CentralServer/internal/telemetry"

	"github.com/gorilla/websocket"
)

const DateTimeLayout = "2006/01/02 15:04:05"

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleSensorWS(hub *telemetry.Hub, validator *auth.TokenValidator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			token = r.Header.Get("Authorization")
		}

		claims, err := validator.ParseAndValidate(token)
		if err != nil {
			http.Error(w, "Unauthorized "+err.Error(), http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		conn.SetReadLimit(4096) // Limit the size of incoming messages to 4KB
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var reading telemetry.SensorReading
			if err := json.Unmarshal(msg, &reading); err != nil {
				continue
			}

			reading.DeviceID = claims.DeviceID
			if reading.Timestamp == "" {
				reading.Timestamp = time.Now().Format(DateTimeLayout)
			}

			hub.Push(reading)
		}
	}
}
