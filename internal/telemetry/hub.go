package telemetry

import (
	"CentralServer/utils"
	"context"
	"sync"
	"time"
)

type Store interface {
	InsertBatch(ctx context.Context, readings []SensorReading) error
}

type AIForwarder interface {
	SendSensorTelemetry(ctx context.Context, reading SensorReading) error
}

type Hub struct {
	store       Store
	ai          AIForwarder
	inboundChan chan SensorReading
	shutdown    chan struct{}
	wg          sync.WaitGroup
}

func NewHub(store Store, ai AIForwarder, bufferCap int) *Hub {
	return &Hub{
		store:       store,
		ai:          ai,
		inboundChan: make(chan SensorReading, bufferCap),
		shutdown:    make(chan struct{}),
	}
}

func (h *Hub) Start(workerCount int) {
	for i := 0; i < workerCount; i++ {
		h.wg.Add(1)
		go h.worker()
	}
}

func (h *Hub) Push(reading SensorReading) bool {
	select {
	case h.inboundChan <- reading:
		return true
	default:
		return false
	}
}

func (h *Hub) worker() {
	defer h.wg.Done()

	const maxBatchSize = 128
	batch := make([]SensorReading, 0, maxBatchSize)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := h.store.InsertBatch(ctx, batch); err != nil {
			utils.Logger{}.Error("Hub Worker: MongoDB batch insert failed: " + err.Error())
		}
		cancel()
		batch = batch[:0] // Retain memory allocations
	}

	for {
		select {
		case <-h.shutdown:
			flush()
			return
		case reading := <-h.inboundChan:
			batch = append(batch, reading)

			// Non-blocking trigger to AI server
			go func(rd SensorReading) {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()
				_ = h.ai.SendSensorTelemetry(ctx, rd)
			}(reading)

			if len(batch) >= maxBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (h *Hub) Stop() {
	close(h.shutdown)
	h.wg.Wait()
}
