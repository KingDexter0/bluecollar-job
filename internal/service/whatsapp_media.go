package service

import (
	"context"
	"fmt"
	"strings"
)

type MediaDownloader interface {
	DocumentRef(ctx context.Context, mediaID string) (string, error)
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
}

func NewMetaMediaDownloader(accessToken, graphAPIVersion string) *MetaMediaDownloader {
	return &MetaMediaDownloader{
		accessToken:     strings.TrimSpace(accessToken),
		graphAPIVersion: strings.TrimSpace(graphAPIVersion),
	}
}

func (d *MetaMediaDownloader) DocumentRef(ctx context.Context, mediaID string) (string, error) {
	mediaID = strings.TrimSpace(mediaID)
	if mediaID == "" {
		return "", fmt.Errorf("media id is required")
	}
	// The production storage layer should resolve and download this media ID only
	// when DOCUMENT_UPLOAD_ENABLED=true. PostgreSQL stores this stable reference,
	// not raw files or provider access URLs.
	return "meta-media:" + mediaID, nil
}
