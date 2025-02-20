package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// MockConfig creates a test configuration
func MockConfig() *Config {
	return &Config{
		API: APIConfig{
			Host:    "localhost",
			Port:    "8080",
			Timeout: 30,
		},
		Server: ServerConfig{
			Host: "localhost",
			Port: "8443",
		},
		Models: []ModelConfig{
			{
				String:      "default_model",
				Width:       512,
				Height:      512,
				Steps:       20,
				Cfgscale:    7.0,
				Loras:       "",
				LoraWeights: 0.0,
				Options:     map[string]interface{}{},
			},
		},
	}
}

// TestGetNewSession tests the session retrieval functionality
func TestGetNewSession(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/API/GetNewSession" {
			t.Errorf("Expected path /API/GetNewSession, got %s", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Return a mock session response
		resp := SessionResponse{
			SessionID: "test-session-123",
			UserID:    "test-user-456",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create a test client
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	config := MockConfig()
	config.API.Host = server.URL[7:] // Remove "http://" prefix
	config.API.Port = ""

	client := NewClient(config, logger)

	// Test the method
	sessionID, err := client.getNewSession()
	if err != nil {
		t.Fatalf("getNewSession failed: %v", err)
	}

	if sessionID != "test-session-123" {
		t.Errorf("Expected sessionID to be 'test-session-123', got '%s'", sessionID)
	}
}

// TestGenerateImage tests the complete image generation flow
func TestGenerateImage(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/API/GetNewSession":
			// Return a mock session
			json.NewEncoder(w).Encode(SessionResponse{SessionID: "test-session-123"})

		case "/API/GenerateText2Image":
			// Verify request body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("Failed to read request body: %v", err)
			}

			var reqBody map[string]interface{}
			if err := json.Unmarshal(body, &reqBody); err != nil {
				t.Fatalf("Failed to parse request body: %v", err)
			}

			if reqBody["prompt"] != "test prompt" {
				t.Errorf("Expected prompt 'test prompt', got '%v'", reqBody["prompt"])
			}

			// Return mock image response
			json.NewEncoder(w).Encode(ImageResponse{Images: []string{"images/test.png"}})

		case "/images/test.png":
			// Return a mock image (simple 1x1 black PNG)
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})

		default:
			t.Errorf("Unexpected request to path: %s", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create a test client
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	config := MockConfig()
	config.API.Host = server.URL[7:] // Remove "http://" prefix
	config.API.Port = ""

	client := NewClient(config, logger)

	// Test the method
	imageData, err := client.GenerateImage("test prompt", 1)
	if err != nil {
		t.Fatalf("GenerateImage failed: %v", err)
	}

	// Verify we got some image data back
	if len(imageData) == 0 {
		t.Errorf("Expected non-empty image data")
	}
}

// TestUploadGeneratedImage tests the image upload functionality
func TestUploadGeneratedImage(t *testing.T) {
	// Setup test server for the HTTPS upload endpoint
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/image" {
			t.Errorf("Expected path /image, got %s", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("Failed to parse multipart form: %v", err)
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("Failed to get form file: %v", err)
		}
		defer file.Close()

		fileData, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("Failed to read file data: %v", err)
		}

		if !bytes.Equal(fileData, []byte("test image data")) {
			t.Errorf("Expected file content 'test image data', got '%s'", string(fileData))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a test client
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	config := MockConfig()
	config.Server.Host = server.URL[8:] // Remove "https://" prefix
	config.Server.Port = ""

	client := NewClient(config, logger)

	// Create a custom HTTP client that uses the test server's TLS certificate
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client.httpClient.Transport = transport

	// Test the method
	err := client.UploadGeneratedImage([]byte("test image data"))
	if err != nil {
		t.Fatalf("UploadGeneratedImage failed: %v", err)
	}
}

// TestGenerateImageInvalidModel tests error handling with invalid model ID
func TestGenerateImageInvalidModel(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	config := MockConfig()
	client := NewClient(config, logger)

	// Test with invalid model ID
	_, err := client.GenerateImage("test prompt", 0)
	if err == nil {
		t.Errorf("Expected error with invalid model ID, got nil")
	}

	_, err = client.GenerateImage("test prompt", 999)
	if err == nil {
		t.Errorf("Expected error with non-existent model ID, got nil")
	}
}

// TestConfig represents minimal config needed for tests
// type Config struct {
// 	API    APIConfig
// 	Server ServerConfig
// 	Models []ModelConfig
// }

// type APIConfig struct {
// 	Host    string
// 	Port    string
// 	Timeout int
// }

// type ServerConfig struct {
// 	Host string
// 	Port string
// }

// type ModelConfig struct {
// 	String      string
// 	Width       int
// 	Height      int
// 	Steps       int
// 	Cfgscale    float64
// 	Loras       string
// 	LoraWeights float64
// 	Options     map[string]interface{}
// }
