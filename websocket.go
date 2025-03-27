package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime/multipart"
	"time"

	"github.com/gorilla/websocket"
)

type Model struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type WebSocketClient struct {
	config *Config
	client *Client
	logger *slog.Logger
	token  string
	models []Model
}

type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
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
		return fmt.Errorf("dial error: %w", err)
	}
	defer conn.Close()

	if err := w.authenticate(conn); err != nil {
		return fmt.Errorf("authentication error: %w", err)
	}

	if err := w.sendModels(conn); err != nil {
		return fmt.Errorf("models send error: %w", err)
	}

	go w.startPingLoop(conn)
	return w.handleMessages(conn)
}

func (w *WebSocketClient) authenticate(conn *websocket.Conn) error {
	authReq := WebSocketMessage{
		Type:    "auth",
		Payload: json.RawMessage(fmt.Sprintf(`{"password":"%s"}`, w.config.Server.Passcode)),
	}

	if err := w.writeJSON(conn, authReq); err != nil {
		return err
	}

	var response WebSocketMessage
	if err := conn.ReadJSON(&response); err != nil {
		return err
	}

	if response.Type != "auth_success" {
		return fmt.Errorf("auth failed")
	}

	var authResponse struct {
		Token string `josn:"token"`
	}
	if err := json.Unmarshal(response.Payload, &authResponse); err != nil {
		return err
	}
	w.token = authResponse.Token

	return nil

}

func (w *WebSocketClient) writeJSON(conn *websocket.Conn, v interface{}) error {
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	defer conn.SetWriteDeadline(time.Time{})
	return conn.WriteJSON(v)
}

func (w *WebSocketClient) requestModels(conn *websocket.Conn) error {
	req := WebSocketMessage{
		Type: "get_models",
	}
	return w.writeJSON(conn, req)
}

func (w *WebSocketClient) sendModels(conn *websocket.Conn) error {
	var models []map[string]interface{}
	for i, m := range w.config.Models {
		models = append(models, map[string]interface{}{
			"id":   i + 1,
			"name": m.Name,
		})
	}

	var msg struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	modelsJSON, err := json.Marshal(models)
	if err != nil {
		return err
	}
	msg.Type = "models_update"
	msg.Payload = modelsJSON

	modelsMSG, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, modelsMSG)
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
		var message WebSocketMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				return fmt.Errorf("connection closed: %w", err)
			}
			return err
		}

		switch message.Type {
		case "task":
			var task Tasukete
			if err := json.Unmarshal(message.Payload, &task); err != nil {
				w.logger.Error("Failed to unmarshal task", "error", err)
				continue
			}
			w.handleTask(conn, &task)

		case "models_update":
			var models []Model
			if err := json.Unmarshal(message.Payload, &models); err != nil {
				w.logger.Error("Failed to unmarshal models", "error", err)
				continue
			}
			w.models = models
			w.logger.Info("Models updated", "count", len(models))

		default:
			w.logger.Warn("Unknown message type", "type", message.Type)
		}
	}
}

func (w *WebSocketClient) handleTask(conn *websocket.Conn, task *Tasukete) {
	// Validate task
	if err := task.Validate(); err != nil {
		w.logger.Error("Invalid task received", "error", err)
		return
	}

	// Process task based on type
	switch task.Type {
	case TTI:
		w.handleTTITask(conn, task)
		// case LLM:
		// 	w.handleLLMTask(conn, task)
		// case Recon:
		// 	w.handleReconTask(conn, task)
	}
}

func (w *WebSocketClient) handleTTITask(conn *websocket.Conn, task *Tasukete) {
	// Update task status
	task.Status = StatusProcessing
	w.sendTaskUpdate(conn, task)

	// Generate image
	result, err := w.client.GenerateImage(task.Prompt, task.Model)
	if err != nil {
		task.Status = StatusFailed
		w.sendTaskUpdate(conn, task)
		return
	}

	// Send result
	if err := w.sendTaskResult(conn, task, result); err != nil {
		w.logger.Error("Failed to send task result", "error", err)
	}
}

func (w *WebSocketClient) sendTaskUpdate(conn *websocket.Conn, task *Tasukete) {
	msg := WebSocketMessage{
		Type:    "task_update",
		Payload: must(json.Marshal(task)),
	}
	if err := w.writeJSON(conn, msg); err != nil {
		w.logger.Error("Failed to send task update", "error", err)
	}
}

func (w *WebSocketClient) sendTaskResult(conn *websocket.Conn, task *Tasukete, result []byte) error {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add task metadata
	metadataField, err := writer.CreateFormField("task")
	if err != nil {
		return err
	}
	if err := json.NewEncoder(metadataField).Encode(task); err != nil {
		return err
	}

	// Add file
	fileField, err := writer.CreateFormFile("file", fmt.Sprintf("%s.png", task.UUID))
	if err != nil {
		return err
	}
	if _, err := fileField.Write(result); err != nil {
		return err
	}

	writer.Close()

	// Prepend the boundary to the message
	boundaryPrefix := []byte(fmt.Sprintf("Boundary: %s\n", writer.Boundary()))
	msg := append(boundaryPrefix, b.Bytes()...)

	// Send as binary WebSocket message
	return conn.WriteMessage(websocket.BinaryMessage, msg)
}

// Helper function for JSON marshaling
func must(data []byte, err error) json.RawMessage {
	if err != nil {
		panic(err)
	}
	return json.RawMessage(data)
}
