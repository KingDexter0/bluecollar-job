package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetaWhatsAppSenderSendMessage(t *testing.T) {
	var requestPath string
	var authHeader string
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		authHeader = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.test"}]}`))
	}))
	defer server.Close()

	sender, err := NewMetaWhatsAppSender(MetaWhatsAppConfig{
		AccessToken:     "test-token",
		PhoneNumberID:   "12345",
		GraphAPIVersion: "v20.0",
		BaseURL:         server.URL,
	})
	if err != nil {
		t.Fatalf("new sender: %v", err)
	}

	if err := sender.SendMessage(context.Background(), "+919876543210", "Hello worker"); err != nil {
		t.Fatalf("send message: %v", err)
	}

	if requestPath != "/v20.0/12345/messages" {
		t.Fatalf("unexpected path %s", requestPath)
	}
	if authHeader != "Bearer test-token" {
		t.Fatalf("unexpected authorization header %q", authHeader)
	}
	if payload["messaging_product"] != "whatsapp" || payload["to"] != "919876543210" || payload["type"] != "text" {
		t.Fatalf("unexpected payload %#v", payload)
	}
}

func TestMetaWhatsAppSenderSanitizesProviderErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"bad token test-token aadhaar otp","type":"OAuthException","code":190}}`, http.StatusBadRequest)
	}))
	defer server.Close()

	sender, err := NewMetaWhatsAppSender(MetaWhatsAppConfig{
		AccessToken:     "test-token",
		PhoneNumberID:   "12345",
		GraphAPIVersion: "v20.0",
		BaseURL:         server.URL,
	})
	if err != nil {
		t.Fatalf("new sender: %v", err)
	}

	err = sender.SendMessage(context.Background(), "+919876543210", "Hello")
	if err == nil {
		t.Fatal("expected provider error")
	}
	lowered := strings.ToLower(err.Error())
	for _, forbidden := range []string{"test-token", "aadhaar", "otp", "bad token"} {
		if strings.Contains(lowered, forbidden) {
			t.Fatalf("error exposed sensitive value %q: %v", forbidden, err)
		}
	}
}

func TestRedisWhatsAppMessageDeduplicator(t *testing.T) {
	store := newFakeRedisStore()
	deduper := NewRedisWhatsAppMessageDeduplicator(store, 0)

	first, err := deduper.MarkProcessed(context.Background(), "wamid.123")
	if err != nil {
		t.Fatalf("first mark: %v", err)
	}
	second, err := deduper.MarkProcessed(context.Background(), "wamid.123")
	if err != nil {
		t.Fatalf("second mark: %v", err)
	}
	if !first || second {
		t.Fatalf("expected first=true second=false, got %v %v", first, second)
	}
}
