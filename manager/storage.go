package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path"
	"time"
)

type StorageClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewStorageClient(baseURL, apiKey string) *StorageClient {
	return &StorageClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

const bucketName = "backtest-reports"

func (s *StorageClient) Upload(userID, jobID, filename string, data []byte) error {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s/%s/%s", s.baseURL, bucketName, userID, jobID, filename)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("", filename)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("write form data: %w", err)
	}
	writer.Close()

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

type signedURLResponse struct {
	SignedURL string `json:"signedUrl"`
}

func (s *StorageClient) CreateSignedURL(userID, jobID, filename string, expiresIn int) (string, error) {
	filePath := path.Join(userID, jobID, filename)
	url := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s", s.baseURL, bucketName, filePath)

	payload, _ := json.Marshal(map[string]int{"expiresIn": expiresIn})
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sign URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("sign URL failed (%d): %s", resp.StatusCode, string(body))
	}

	var result signedURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return s.baseURL + "/storage/v1" + result.SignedURL, nil
}

func (s *StorageClient) RemoveFolder(userID, jobID string) {
	folderPath := path.Join(userID, jobID)
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, bucketName, folderPath)

	payload, _ := json.Marshal(map[string][]string{"prefixes": {folderPath + "/"}})
	req, err := http.NewRequest("DELETE", url, bytes.NewReader(payload))
	if err != nil {
		log.Printf("storage remove: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("storage remove: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("storage remove failed (%d): %s", resp.StatusCode, string(body))
	}
}
