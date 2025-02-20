package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"time"
)

type Client struct {
	config     *Config
	httpClient *http.Client
	logger     *slog.Logger
}

type SessionResponse struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
}

type ImageResponse struct {
	Images []string `json:"images"`
}

func NewClient(config *Config, logger *slog.Logger) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.API.Timeout) * time.Second,
		},
		logger: logger,
	}
}

// GenerateImage generates an image based on the provided prompt and model ID
// Returns the image data as a byte slice
func (c *Client) GenerateImage(prompt string, modelID int) ([]byte, error) {
	if modelID <= 0 || modelID > len(c.config.Models) {
		return nil, fmt.Errorf("invalid modelID: %d", modelID)
	}

	// Get session
	sessionID, err := c.getNewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %v", err)
	}

	// Generate image
	imageURL, err := c.generateImage(sessionID, prompt, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %v", err)
	}

	// Download image
	imageData, err := c.downloadImageBytes(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %v", err)
	}

	return imageData, nil
}

// UploadGeneratedImage uploads a previously generated image
func (c *Client) UploadGeneratedImage(imageData []byte) error {
	return c.uploadImageBytes(imageData)
}

func (c *Client) getNewSession() (string, error) {
	url := fmt.Sprintf("http://%s:%s/API/GetNewSession", c.config.API.Host, c.config.API.Port)

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", fmt.Errorf("session request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("session request returned non-OK status: %d", resp.StatusCode)
	}

	var sessionResp SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return "", fmt.Errorf("failed to decode session response: %w", err)
	}

	if sessionResp.SessionID == "" {
		return "", fmt.Errorf("received empty session ID")
	}

	return sessionResp.SessionID, nil
}

func (c *Client) generateImage(sessionID, prompt string, modelID int) (string, error) {
	model := c.config.Models[modelID-1]
	generateBody := map[string]interface{}{
		"session_id": sessionID,
		"images":     1,
		"prompt":     prompt,
		"model":      model.String,
		"width":      model.Width,
		"height":     model.Height,
		"steps":      model.Steps,
		"cfgscale":   model.Cfgscale,
	}

	if model.Loras != "" {
		generateBody["loras"] = model.Loras
		if generateBody["loras"] == "GyateGyate_pdxl_Incrs_v1" {
			generateBody["prompt"] = prompt + ", open mouth, smile, chibi, :d, :3"
		}
	}

	if model.LoraWeights != 0.0 {
		generateBody["loraweights"] = model.LoraWeights
	}

	for name, val := range model.Options {
		generateBody[name] = val
	}

	bodyJSON, err := json.Marshal(generateBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("http://%s:%s/API/GenerateText2Image", c.config.API.Host, c.config.API.Port)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("image generation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("image generation returned non-OK status: %d", resp.StatusCode)
	}

	var imageResp ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&imageResp); err != nil {
		return "", fmt.Errorf("failed to decode image response: %w", err)
	}

	if len(imageResp.Images) == 0 {
		return "", fmt.Errorf("no images returned from response")
	}

	return fmt.Sprintf("http://%s:%s/%s", c.config.API.Host, c.config.API.Port, imageResp.Images[0]), nil
}

// downloadImageBytes downloads an image and returns it as a byte slice
func (c *Client) downloadImageBytes(imageURL string) ([]byte, error) {
	resp, err := c.httpClient.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned non-OK status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// uploadImageBytes uploads image data directly without saving to disk first
func (c *Client) uploadImageBytes(imageData []byte) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a unique filename for the form field
	filename := time.Now().UTC().Format("20060102T150405Z") + "image.png"

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err = io.Copy(part, bytes.NewReader(imageData)); err != nil {
		return fmt.Errorf("failed to copy image data: %w", err)
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := fmt.Sprintf("https://%s:%s/image", c.config.Server.Host, c.config.Server.Port)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// TODO: For production, use proper certificate validation instead of InsecureSkipVerify
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: transport}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
