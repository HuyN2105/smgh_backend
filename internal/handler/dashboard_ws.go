package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var dashboardUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Cho phép dashboard web kết nối từ mọi domain
		return true
	},
}

// Client đại diện cho một kết nối từ giao diện web/dashboard
type Client struct {
	hub  *DashboardHub
	conn *websocket.Conn
	send chan []byte
}

// DashboardHub quản lý danh sách các trình duyệt/app đang xem dữ liệu trực tiếp
type DashboardHub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewDashboardHub() *DashboardHub {
	return &DashboardHub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run lắng nghe các sự kiện thêm/xóa client và phân phối dữ liệu
func (h *DashboardHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastSensorData gửi gói dữ liệu cảm biến mới nhận được tới toàn bộ màn hình dashboard
func (h *DashboardHub) BroadcastSensorData(data any) {
	bytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("Lỗi serialize dữ liệu dashboard: %v", err)
		return
	}
	h.broadcast <- bytes
}

type DashboardHandler struct {
	hub *DashboardHub
}

func NewDashboardHandler(hub *DashboardHub) *DashboardHandler {
	return &DashboardHandler{hub: hub}
}

// ServeWS xử lý nâng cấp kết nối HTTP lên WebSocket cho dashboard frontend
func (h *DashboardHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := dashboardUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Nâng cấp WebSocket dashboard thất bại: %v", err)
		return
	}

	client := &Client{
		hub:  h.hub,
		conn: conn,
		send: make(chan []byte, 64),
	}
	h.hub.register <- client

	// Goroutine gửi dữ liệu từ kênh send xuống client qua socket
	go client.writePump()
	// Goroutine đọc để giữ kết nối và phát hiện khi client đóng tab/trình duyệt
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Gom các message còn tồn đọng trong queue gửi cùng lúc
			n := len(c.send)
			for range n {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
