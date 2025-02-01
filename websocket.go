package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketClient struct {
	config *Config
	client *Client
	logger *slog.Logger
}

func NewWebSocketClient(config *Config, client *Client, logger *slog.Logger) *WebSocketClient {
	return &WebSocketClient{
		config: config,
		client: client,
		logger: logger,
	}
}

func (w *WebSocketClient) Start() {
	for {
		if err := w.connect(); err != nil {
			w.logger.Error("WebSocket connection failed", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}
	}
}

func (w *WebSocketClient) connect() error {
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	url := fmt.Sprintf("wss://%s:%s/ws", w.config.Server.Host, w.config.Server.Port)
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := w.authenticate(conn); err != nil {
		return err
	}

	if err := w.sendModels(conn); err != nil {
		return err
	}

	go w.startPingLoop(conn)
	return w.handleMessages(conn)
}

func (w *WebSocketClient) authenticate(conn *websocket.Conn) error {
	authJSON := fmt.Sprintf(`{"password":"%s"}`, w.config.Server.Passcode)
	return conn.WriteMessage(websocket.TextMessage, []byte(authJSON))
}

func (w *WebSocketClient) sendModels(conn *websocket.Conn) error {
	var models []map[string]interface{}
	for i, m := range w.config.Models {
		models = append(models, map[string]interface{}{
			"id":   i + 1,
			"name": m.Name,
		})
	}

	modelsJSON, err := json.Marshal(models)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, modelsJSON)
}

func (w *WebSocketClient) startPingLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
	}
}

func (w *WebSocketClient) handleMessages(conn *websocket.Conn) error {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				return fmt.Errorf("connection closed: %v", err)
			}
			return err
		}

		parts := strings.Split(string(message), "^")
		if len(parts) != 2 {
			w.logger.Error("Invalid message format")
			continue
		}

		modelID, err := strconv.Atoi(parts[0])
		if err != nil {
			w.logger.Error("Invalid model ID", "error", err)
			continue
		}

		if modelID < 1 || modelID > len(w.config.Models) {
			w.logger.Error("Model ID out of range")
			continue
		}

		prompt := parts[1]
		if err := w.client.GenerateImage(prompt, modelID); err != nil {
			w.logger.Error("Failed to generate image", "error", err)
		}
	}
}
