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
	"os"
	"path/filepath"
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

func (c *Client) GenerateImage(prompt string, modelID int) error {
	// Get session
	sessionID, err := c.getNewSession()
	if err != nil {
		return fmt.Errorf("failed to get session: %v", err)
	}

	// Generate image
	imageURL, err := c.generateImage(sessionID, prompt, modelID)
	if err != nil {
		return fmt.Errorf("failed to generate image: %v", err)
	}

	// Download and save image
	fileName, err := c.downloadImage(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %v", err)
	}
	defer os.Remove(fileName)

	// Upload image
	if err := c.uploadImage(fileName); err != nil {
		return fmt.Errorf("failed to upload image: %v", err)
	}

	return nil
}

func (c *Client) getNewSession() (string, error) {
	url := fmt.Sprintf("http://%s:%s/API/GetNewSession", c.config.API.Host, c.config.API.Port)

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var sessionResp SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return "", err
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
		return "", err
	}

	url := fmt.Sprintf("http://%s:%s/API/GenerateText2Image", c.config.API.Host, c.config.API.Port)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(bodyJSON))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var imageResp ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&imageResp); err != nil {
		return "", err
	}

	if len(imageResp.Images) == 0 {
		return "", fmt.Errorf("no images returned from response")
	}

	return fmt.Sprintf("http://%s:%s/%s", c.config.API.Host, c.config.API.Port, imageResp.Images[0]), nil
}

func (c *Client) downloadImage(imageURL string) (string, error) {
	resp, err := c.httpClient.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	fileName := time.Now().UTC().Format("20060102T150405Z") + "image.png"
	out, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return fileName, nil
}

func (c *Client) uploadImage(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return err
	}

	if _, err = io.Copy(part, file); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s:%s/image", c.config.Server.Host, c.config.Server.Port)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: transport}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed with status: %d", resp.StatusCode)
	}

	return nil
}
