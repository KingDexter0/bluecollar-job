package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ObjectStore interface {
	Put(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)
	SignedURL(ctx context.Context, documentRef string) (string, error)
}

type LocalObjectStore struct {
	basePath string
}

func NewLocalObjectStore(basePath string) *LocalObjectStore {
	if strings.TrimSpace(basePath) == "" {
		basePath = "./var/uploads"
	}
	return &LocalObjectStore{basePath: basePath}
}

func (s *LocalObjectStore) Put(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	cleanKey := filepath.Clean(strings.TrimPrefix(key, "/"))
	if cleanKey == "." || strings.HasPrefix(cleanKey, "..") {
		return "", fmt.Errorf("invalid object key")
	}
	target := filepath.Join(s.basePath, cleanKey)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return "", err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := io.Copy(file, reader); err != nil {
		return "", err
	}
	return "local://" + cleanKey, nil
}

func (s *LocalObjectStore) SignedURL(ctx context.Context, documentRef string) (string, error) {
	if !strings.HasPrefix(documentRef, "local://") {
		return "", fmt.Errorf("unsupported local document reference")
	}
	return documentRef, nil
}

type LinodeObjectStore struct {
	bucket   string
	region   string
	endpoint string
}

func NewLinodeObjectStore(bucket, region, endpoint string) *LinodeObjectStore {
	return &LinodeObjectStore{bucket: bucket, region: region, endpoint: endpoint}
}

func (s *LinodeObjectStore) Put(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	return "", fmt.Errorf("linode object storage provider is configured as a placeholder")
}

func (s *LinodeObjectStore) SignedURL(ctx context.Context, documentRef string) (string, error) {
	return "", fmt.Errorf("linode object storage provider is configured as a placeholder")
}
