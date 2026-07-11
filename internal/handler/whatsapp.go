package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

type WhatsAppHandler struct {
	verifyToken           string
	bot                   service.WhatsAppBotService
	deduplicator          service.WhatsAppMessageDeduplicator
	mediaDownloader       service.MediaDownloader
	documentUploadEnabled bool
}

func NewWhatsAppHandler(verifyToken string, bot service.WhatsAppBotService, deduplicator service.WhatsAppMessageDeduplicator, mediaDownloader service.MediaDownloader, documentUploadEnabled bool) *WhatsAppHandler {
	return &WhatsAppHandler{
		verifyToken:           verifyToken,
		bot:                   bot,
		deduplicator:          deduplicator,
		mediaDownloader:       mediaDownloader,
		documentUploadEnabled: documentUploadEnabled,
	}
}

func (h *WhatsAppHandler) VerifyWebhook(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")
	if mode == "subscribe" && token != "" && token == h.verifyToken {
		c.String(http.StatusOK, challenge)
		return
	}
	writeError(c, http.StatusForbidden, "verification_failed", "invalid WhatsApp verify token")
}

func (h *WhatsAppHandler) ReceiveWebhook(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	message, err := parseWhatsAppMessage(payload)
	if err != nil {
		if err == service.ErrNotFound {
			c.JSON(http.StatusOK, gin.H{"ok": true, "ignored": true})
			return
		}
		writeError(c, http.StatusBadRequest, "invalid_webhook_payload", err.Error())
		return
	}
	if h.deduplicator != nil && message.MessageID != "" {
		isNew, err := h.deduplicator.MarkProcessed(c.Request.Context(), message.MessageID)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		if !isNew {
			c.JSON(http.StatusOK, gin.H{"ok": true, "duplicate": true})
			return
		}
	}
	if message.MediaRef != nil {
		ref := safeMediaReference(*message.MediaRef)
		if h.documentUploadEnabled && h.mediaDownloader != nil {
			downloadedRef, err := h.mediaDownloader.DocumentRef(c.Request.Context(), *message.MediaRef)
			if err != nil {
				writeServiceError(c, err)
				return
			}
			ref = downloadedRef
		}
		message.MediaRef = &ref
	}

	reply, err := h.bot.HandleIncomingMessage(c.Request.Context(), message)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "reply": reply})
}

func parseWhatsAppMessage(payload map[string]any) (service.IncomingWhatsAppMessage, error) {
	if message, ok := parseOpenWAMessage(payload); ok {
		return message, nil
	}
	if message, ok := parseMetaMessage(payload); ok {
		return message, nil
	}
	return service.IncomingWhatsAppMessage{}, service.ErrNotFound
}

func parseOpenWAMessage(payload map[string]any) (service.IncomingWhatsAppMessage, bool) {
	phone := firstNonEmptyString(payload, "phone_number", "phone", "from", "sender", "chatId")
	text := firstNonEmptyString(payload, "text", "body", "message", "caption")
	messageType := firstNonEmptyString(payload, "type", "message_type")
	mediaRef := firstNonEmptyString(payload, "media_ref", "mediaUrl", "media_url", "document_ref")

	if phone == "" {
		return service.IncomingWhatsAppMessage{}, false
	}
	if messageType == "" {
		messageType = "text"
	}
	phone = normalizeWebhookPhone(phone)
	var mediaRefPtr *string
	if mediaRef != "" {
		mediaRefPtr = &mediaRef
	}
	messageID := firstNonEmptyString(payload, "id", "message_id")
	return service.IncomingWhatsAppMessage{
		PhoneNumber: phone,
		Text:        text,
		MessageType: messageType,
		MessageID:   messageID,
		MediaRef:    mediaRefPtr,
	}, true
}

func parseMetaMessage(payload map[string]any) (service.IncomingWhatsAppMessage, bool) {
	entries, ok := payload["entry"].([]any)
	if !ok || len(entries) == 0 {
		return service.IncomingWhatsAppMessage{}, false
	}
	for _, entryValue := range entries {
		entry, ok := entryValue.(map[string]any)
		if !ok {
			continue
		}
		changes, ok := entry["changes"].([]any)
		if !ok {
			continue
		}
		for _, changeValue := range changes {
			change, ok := changeValue.(map[string]any)
			if !ok {
				continue
			}
			value, ok := change["value"].(map[string]any)
			if !ok {
				continue
			}
			messages, ok := value["messages"].([]any)
			if !ok || len(messages) == 0 {
				continue
			}
			messageMap, ok := messages[0].(map[string]any)
			if !ok {
				continue
			}
			phone := normalizeWebhookPhone(stringValue(messageMap["from"]))
			messageID := stringValue(messageMap["id"])
			messageType := stringValue(messageMap["type"])
			text := ""
			var mediaRef *string
			if textPayload, ok := messageMap["text"].(map[string]any); ok {
				text = stringValue(textPayload["body"])
			}
			if documentPayload, ok := messageMap["document"].(map[string]any); ok {
				ref := firstNonEmptyString(documentPayload, "id", "filename", "link")
				if ref != "" {
					mediaRef = &ref
				}
			}
			if imagePayload, ok := messageMap["image"].(map[string]any); ok {
				ref := firstNonEmptyString(imagePayload, "id", "link")
				if ref != "" {
					mediaRef = &ref
				}
			}
			if interactivePayload, ok := messageMap["interactive"].(map[string]any); ok {
				text = parseMetaInteractiveReply(interactivePayload)
				if messageType == "" {
					messageType = "interactive"
				}
			}
			if buttonPayload, ok := messageMap["button"].(map[string]any); ok && text == "" {
				text = firstNonEmptyString(buttonPayload, "payload", "text")
				if messageType == "" {
					messageType = "button"
				}
			}
			if phone != "" {
				if messageType == "" {
					messageType = "text"
				}
				return service.IncomingWhatsAppMessage{
					PhoneNumber: phone,
					Text:        text,
					MessageType: messageType,
					MessageID:   messageID,
					MediaRef:    mediaRef,
				}, true
			}
		}
	}
	return service.IncomingWhatsAppMessage{}, false
}

func parseMetaInteractiveReply(payload map[string]any) string {
	interactiveType := stringValue(payload["type"])
	switch interactiveType {
	case "button_reply":
		if reply, ok := payload["button_reply"].(map[string]any); ok {
			return firstNonEmptyString(reply, "id", "title")
		}
	case "list_reply":
		if reply, ok := payload["list_reply"].(map[string]any); ok {
			return firstNonEmptyString(reply, "id", "title", "description")
		}
	}
	return ""
}

func firstNonEmptyString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(stringValue(payload[key]))
		if value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func normalizeWebhookPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.TrimSuffix(phone, "@c.us")
	phone = strings.TrimSuffix(phone, "@s.whatsapp.net")
	if strings.HasPrefix(phone, "+") {
		return phone
	}
	digits := strings.Builder{}
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	cleaned := digits.String()
	if cleaned == "" {
		return ""
	}
	return "+" + cleaned
}

func safeMediaReference(mediaID string) string {
	mediaID = strings.TrimSpace(mediaID)
	if mediaID == "" {
		return ""
	}
	if strings.Contains(mediaID, ":") {
		return mediaID
	}
	return "wa-media:" + mediaID
}
