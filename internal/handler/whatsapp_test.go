package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

func TestWhatsAppWebhookVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewWhatsAppHandler("verify-token", &fakeWhatsAppBot{}, nil, nil, false)
	router := gin.New()
	router.GET("/api/v1/whatsapp/webhook", handler.VerifyWebhook)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/whatsapp/webhook?hub.mode=subscribe&hub.verify_token=verify-token&hub.challenge=challenge-123", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "challenge-123" {
		t.Fatalf("expected challenge response, got %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/whatsapp/webhook?hub.mode=subscribe&hub.verify_token=bad&hub.challenge=challenge-123", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for invalid token, got %d", response.Code)
	}
}

func TestParseMetaPayloadTextInteractiveAndMedia(t *testing.T) {
	textPayload := map[string]any{
		"entry": []any{map[string]any{"changes": []any{map[string]any{"value": map[string]any{"messages": []any{map[string]any{
			"from": "919876543210",
			"id":   "wamid.text",
			"type": "text",
			"text": map[string]any{"body": "menu"},
		}}}}}}},
	}
	message, err := parseWhatsAppMessage(textPayload)
	if err != nil {
		t.Fatalf("parse text: %v", err)
	}
	if message.PhoneNumber != "+919876543210" || message.Text != "menu" || message.MessageID != "wamid.text" {
		t.Fatalf("unexpected text message %#v", message)
	}

	buttonPayload := map[string]any{
		"entry": []any{map[string]any{"changes": []any{map[string]any{"value": map[string]any{"messages": []any{map[string]any{
			"from": "919876543210",
			"id":   "wamid.button",
			"type": "interactive",
			"interactive": map[string]any{
				"type":         "button_reply",
				"button_reply": map[string]any{"id": "1", "title": "Check Application Status"},
			},
		}}}}}}},
	}
	message, err = parseWhatsAppMessage(buttonPayload)
	if err != nil {
		t.Fatalf("parse button: %v", err)
	}
	if message.Text != "1" || message.MessageType != "interactive" {
		t.Fatalf("unexpected button message %#v", message)
	}

	mediaPayload := map[string]any{
		"entry": []any{map[string]any{"changes": []any{map[string]any{"value": map[string]any{"messages": []any{map[string]any{
			"from":     "919876543210",
			"id":       "wamid.media",
			"type":     "document",
			"document": map[string]any{"id": "media-123"},
		}}}}}}},
	}
	message, err = parseWhatsAppMessage(mediaPayload)
	if err != nil {
		t.Fatalf("parse media: %v", err)
	}
	if message.MediaRef == nil || *message.MediaRef != "media-123" {
		t.Fatalf("expected media ID, got %#v", message)
	}
}

func TestWhatsAppHandlerDuplicateMessageIgnored(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bot := &fakeWhatsAppBot{}
	handler := NewWhatsAppHandler("verify-token", bot, &fakeDeduplicator{seen: true}, nil, false)
	router := gin.New()
	router.POST("/api/v1/whatsapp/webhook", handler.ReceiveWebhook)

	body := `{"entry":[{"changes":[{"value":{"messages":[{"from":"919876543210","id":"wamid.dup","type":"text","text":{"body":"menu"}}]}}]}]}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/webhook", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"duplicate":true`) {
		t.Fatalf("expected duplicate ack, got %d %s", response.Code, response.Body.String())
	}
	if bot.called {
		t.Fatal("bot should not be called for duplicate message")
	}
}

func TestWhatsAppHandlerIgnoresStatusCallbacks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewWhatsAppHandler("verify-token", &fakeWhatsAppBot{}, nil, nil, false)
	router := gin.New()
	router.POST("/api/v1/whatsapp/webhook", handler.ReceiveWebhook)

	body := `{"entry":[{"changes":[{"value":{"statuses":[{"id":"wamid.sent","status":"sent"}]}}]}]}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/webhook", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"ignored":true`) {
		t.Fatalf("expected ignored ack, got %d %s", response.Code, response.Body.String())
	}
}

type fakeWhatsAppBot struct {
	called bool
}

func (b *fakeWhatsAppBot) HandleIncomingMessage(ctx context.Context, message service.IncomingWhatsAppMessage) (*service.BotReply, error) {
	b.called = true
	return &service.BotReply{PhoneNumber: message.PhoneNumber, Message: "ok", State: "test"}, nil
}

type fakeDeduplicator struct {
	seen bool
}

func (d *fakeDeduplicator) MarkProcessed(ctx context.Context, messageID string) (bool, error) {
	return !d.seen, nil
}
