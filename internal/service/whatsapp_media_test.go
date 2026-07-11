package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetaMediaDownloaderStoresDownloadedMedia(t *testing.T) {
	store := &fakeMediaObjectStore{}
	var metadataAuth string
	var contentAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v20.0/media-123":
			metadataAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"url":"` + "http://" + r.Host + `/media-content","mime_type":"image/jpeg","file_size":12,"id":"media-123"}`))
		case "/media-content":
			contentAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("fake-content"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	downloader := NewMetaMediaDownloader("test-token", "v20.0", store)
	downloader.baseURL = server.URL

	ref, err := downloader.DocumentRef(context.Background(), "media-123")
	if err != nil {
		t.Fatalf("document ref: %v", err)
	}
	if ref != "s3://bucket/"+store.key {
		t.Fatalf("unexpected ref %q", ref)
	}
	if metadataAuth != "Bearer test-token" || contentAuth != "Bearer test-token" {
		t.Fatalf("expected bearer auth on metadata/content requests")
	}
	if store.contentType != "image/jpeg" {
		t.Fatalf("expected image/jpeg, got %s", store.contentType)
	}
	if !strings.HasSuffix(store.key, ".jpg") || !strings.HasPrefix(store.key, "worker-documents/") {
		t.Fatalf("unexpected object key %s", store.key)
	}
	if store.body != "fake-content" {
		t.Fatalf("unexpected body %q", store.body)
	}
}

func TestMetaMediaDownloaderRejectsOversizedMedia(t *testing.T) {
	store := &fakeMediaObjectStore{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"url":"http://example.invalid/media","mime_type":"image/jpeg","file_size":999999999,"id":"media-123"}`))
	}))
	defer server.Close()

	downloader := NewMetaMediaDownloader("test-token", "v20.0", store)
	downloader.baseURL = server.URL
	downloader.maxBytes = 10

	_, err := downloader.DocumentRef(context.Background(), "media-123")
	if err == nil || !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("expected max size error, got %v", err)
	}
	if store.key != "" {
		t.Fatalf("object store should not be called for oversized media")
	}
}

type fakeMediaObjectStore struct {
	key         string
	contentType string
	body        string
}

func (s *fakeMediaObjectStore) Put(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	s.key = key
	s.contentType = contentType
	s.body = string(bytes.TrimSpace(body))
	return "s3://bucket/" + key, nil
}
