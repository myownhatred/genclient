// main.go
package main

import (
	"os"
)

func main() {
	// Initialize logger
	logger := initLogger()

	// Load configuration
	conf, err := LoadConfig("./config.yaml")
	if err != nil {
		logger.Error("Config load failed", "error", err)
		os.Exit(1)
	}

	// Create client instance
	client := NewClient(conf, logger)

	// Start the WebSocket client
	wsClient := NewWebSocketClient(conf, client, logger)
	wsClient.Start()
}
