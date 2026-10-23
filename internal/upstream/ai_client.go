package upstream

import (
	"CentralServer/internal/telemetry"
	"CentralServer/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type AIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAIClient(baseURL string) *AIClient {
	return &AIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    500,
				MaxConnsPerHost: 200,
				IdleConnTimeout: 90 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   2 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},
	}
}

// ForwardImageStream pipes the raw reader directly across the socket without allocating memory for the body
func (c *AIClient) ForwardImageStream(ctx context.Context, deviceID, contextType string, body io.Reader) (io.ReadCloser, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/predict/image", body)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Content-Type", contextType)
	req.Header.Set("X-Device-Id", deviceID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}

	return resp.Body, resp.StatusCode, nil
}

func (c *AIClient) SendSensorTelemetry(ctx context.Context, reading telemetry.SensorReading) error {
	payload, err := json.Marshal(reading)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/predict/sensor", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			utils.Logger{}.Error("AIClient.SendSensorTelemetry: failed to close response body")
		}
	}(resp.Body)

	if resp.StatusCode >= 400 {
		utils.Logger{}.Error("AIClient.SendSensorTelemetry: received error status code " + fmt.Sprint(resp.StatusCode))
		return fmt.Errorf("AIClient.SendSensorTelemetry: received error status code %d", resp.StatusCode)
	}
	return nil
}
