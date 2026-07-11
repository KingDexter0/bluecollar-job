package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type MediaDownloader interface {
	DocumentRef(ctx context.Context, mediaID string) (string, error)
}

type MediaObjectStore interface {
	Put(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)
}

type MockMediaDownloader struct{}

func NewMockMediaDownloader() *MockMediaDownloader {
	return &MockMediaDownloader{}
}

func (d *MockMediaDownloader) DocumentRef(ctx context.Context, mediaID string) (string, error) {
	mediaID = strings.TrimSpace(mediaID)
	if mediaID == "" {
		return "", fmt.Errorf("media id is required")
	}
	return "mock-media:" + mediaID, nil
}

type MetaMediaDownloader struct {
	accessToken     string
	graphAPIVersion string
	objectStore     MediaObjectStore
	httpClient      *http.Client
	baseURL         string
	maxBytes        int64
}

func NewMetaMediaDownloader(accessToken, graphAPIVersion string, objectStore MediaObjectStore) *MetaMediaDownloader {
	return &MetaMediaDownloader{
		accessToken:     strings.TrimSpace(accessToken),
		graphAPIVersion: strings.TrimSpace(graphAPIVersion),
		objectStore:     objectStore,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		baseURL:         "https://graph.facebook.com",
		maxBytes:        10 << 20,
	}
}

func (d *MetaMediaDownloader) DocumentRef(ctx context.Context, mediaID string) (string, error) {
	mediaID = strings.TrimSpace(mediaID)
	if mediaID == "" {
		return "", fmt.Errorf("media id is required")
	}
	if d.objectStore == nil {
		return "", fmt.Errorf("object storage is required for Meta media downloads")
	}
	if d.accessToken == "" {
		return "", fmt.Errorf("WhatsApp access token is required for Meta media downloads")
	}
	metadata, err := d.fetchMetadata(ctx, mediaID)
	if err != nil {
		return "", err
	}
	if metadata.FileSize > 0 && metadata.FileSize > d.maxBytes {
		return "", fmt.Errorf("Meta media file exceeds maximum allowed size")
	}
	content, contentType, err := d.fetchContent(ctx, metadata.URL)
	if err != nil {
		return "", err
	}
	defer content.Close()

	if metadata.MimeType != "" {
		contentType = metadata.MimeType
	}
	key := fmt.Sprintf("worker-documents/%s/%s%s", time.Now().UTC().Format("2006/01/02"), randomHex(12), extensionFromContentType(contentType))
	return d.objectStore.Put(ctx, key, io.LimitReader(content, d.maxBytes+1), contentType)
}

type metaMediaMetadata struct {
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	FileSize int64  `json:"file_size"`
	ID       string `json:"id"`
}

func (d *MetaMediaDownloader) fetchMetadata(ctx context.Context, mediaID string) (*metaMediaMetadata, error) {
	version := strings.Trim(d.graphAPIVersion, "/")
	if version == "" {
		version = "v20.0"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(d.baseURL, "/")+"/"+version+"/"+mediaID, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+d.accessToken)
	response, err := d.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("Meta media metadata request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("Meta media metadata request failed with status %d", response.StatusCode)
	}
	var metadata metaMediaMetadata
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("Meta media metadata response is invalid")
	}
	if strings.TrimSpace(metadata.URL) == "" {
		return nil, fmt.Errorf("Meta media metadata did not include a media URL")
	}
	return &metadata, nil
}

func (d *MetaMediaDownloader) fetchContent(ctx context.Context, mediaURL string) (io.ReadCloser, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return nil, "", err
	}
	request.Header.Set("Authorization", "Bearer "+d.accessToken)
	response, err := d.httpClient.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf("Meta media download failed")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_ = response.Body.Close()
		return nil, "", fmt.Errorf("Meta media download failed with status %d", response.StatusCode)
	}
	return response.Body, response.Header.Get("Content-Type"), nil
}

func randomHex(bytesCount int) string {
	buffer := make([]byte, bytesCount)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}

func extensionFromContentType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	default:
		return filepath.Ext(contentType)
	}
}
