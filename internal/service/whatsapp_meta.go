package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrTemporaryWhatsAppDelivery = errors.New("temporary WhatsApp delivery failure")

type WhatsAppButton struct {
	ID    string
	Title string
}

type WhatsAppTemplateSender interface {
	SendTemplateMessage(ctx context.Context, phoneNumber, templateName, languageCode string, parameters []string) error
}

type WhatsAppInteractiveSender interface {
	SendInteractiveButtons(ctx context.Context, phoneNumber, bodyText string, buttons []WhatsAppButton) error
}

type MetaWhatsAppConfig struct {
	AccessToken     string
	PhoneNumberID   string
	GraphAPIVersion string
	BaseURL         string
}

type MetaWhatsAppSender struct {
	cfg        MetaWhatsAppConfig
	httpClient *http.Client
}

func NewMetaWhatsAppSender(cfg MetaWhatsAppConfig) (*MetaWhatsAppSender, error) {
	cfg.AccessToken = strings.TrimSpace(cfg.AccessToken)
	cfg.PhoneNumberID = strings.TrimSpace(cfg.PhoneNumberID)
	cfg.GraphAPIVersion = strings.TrimSpace(cfg.GraphAPIVersion)
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("WhatsApp access token is required")
	}
	if cfg.PhoneNumberID == "" {
		return nil, fmt.Errorf("WhatsApp phone number ID is required")
	}
	if cfg.GraphAPIVersion == "" {
		cfg.GraphAPIVersion = "v20.0"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://graph.facebook.com"
	}
	return &MetaWhatsAppSender{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (s *MetaWhatsAppSender) SendMessage(ctx context.Context, phoneNumber, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("WhatsApp message body is required")
	}
	return s.send(ctx, map[string]any{
		"messaging_product": "whatsapp",
		"to":                metaRecipient(phoneNumber),
		"type":              "text",
		"text": map[string]any{
			"preview_url": false,
			"body":        message,
		},
	})
}

func (s *MetaWhatsAppSender) SendTextMessage(ctx context.Context, phoneNumber, message string) error {
	return s.SendMessage(ctx, phoneNumber, message)
}

func (s *MetaWhatsAppSender) SendTemplateMessage(ctx context.Context, phoneNumber, templateName, languageCode string, parameters []string) error {
	templateName = strings.TrimSpace(templateName)
	languageCode = strings.TrimSpace(languageCode)
	if templateName == "" {
		return fmt.Errorf("WhatsApp template name is required")
	}
	if languageCode == "" {
		languageCode = "en"
	}

	template := map[string]any{
		"name": templateName,
		"language": map[string]any{
			"code": languageCode,
		},
	}
	if len(parameters) > 0 {
		bodyParams := make([]map[string]any, 0, len(parameters))
		for _, parameter := range parameters {
			bodyParams = append(bodyParams, map[string]any{
				"type": "text",
				"text": parameter,
			})
		}
		template["components"] = []map[string]any{{
			"type":       "body",
			"parameters": bodyParams,
		}}
	}

	return s.send(ctx, map[string]any{
		"messaging_product": "whatsapp",
		"to":                metaRecipient(phoneNumber),
		"type":              "template",
		"template":          template,
	})
}

func (s *MetaWhatsAppSender) SendInteractiveButtons(ctx context.Context, phoneNumber, bodyText string, buttons []WhatsAppButton) error {
	bodyText = strings.TrimSpace(bodyText)
	if bodyText == "" {
		return fmt.Errorf("WhatsApp interactive body is required")
	}
	if len(buttons) == 0 || len(buttons) > 3 {
		return fmt.Errorf("WhatsApp interactive buttons must include 1 to 3 buttons")
	}
	apiButtons := make([]map[string]any, 0, len(buttons))
	for _, button := range buttons {
		id := strings.TrimSpace(button.ID)
		title := strings.TrimSpace(button.Title)
		if id == "" || title == "" {
			return fmt.Errorf("WhatsApp interactive button id and title are required")
		}
		apiButtons = append(apiButtons, map[string]any{
			"type": "reply",
			"reply": map[string]any{
				"id":    id,
				"title": title,
			},
		})
	}
	return s.send(ctx, map[string]any{
		"messaging_product": "whatsapp",
		"to":                metaRecipient(phoneNumber),
		"type":              "interactive",
		"interactive": map[string]any{
			"type": "button",
			"body": map[string]any{
				"text": bodyText,
			},
			"action": map[string]any{
				"buttons": apiButtons,
			},
		},
	})
}

func (s *MetaWhatsAppSender) send(ctx context.Context, payload map[string]any) error {
	recipient := strings.TrimSpace(stringFromAny(payload["to"]))
	if recipient == "" {
		return fmt.Errorf("recipient phone number is required")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.messagesURL(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+s.cfg.AccessToken)
	request.Header.Set("Content-Type", "application/json")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: provider request failed", ErrTemporaryWhatsAppDelivery)
	}
	defer response.Body.Close()

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}

	metaErr := decodeMetaAPIError(response.Body)
	message := "Meta WhatsApp API error"
	if metaErr.Code != 0 {
		message = fmt.Sprintf("%s: code %d", message, metaErr.Code)
	}
	if metaErr.Type != "" {
		message = fmt.Sprintf("%s type %s", message, metaErr.Type)
	}
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
		return fmt.Errorf("%w: %s status %d", ErrTemporaryWhatsAppDelivery, message, response.StatusCode)
	}
	return fmt.Errorf("%s status %d", message, response.StatusCode)
}

func (s *MetaWhatsAppSender) messagesURL() string {
	baseURL := strings.TrimRight(s.cfg.BaseURL, "/")
	version := strings.Trim(s.cfg.GraphAPIVersion, "/")
	return fmt.Sprintf("%s/%s/%s/messages", baseURL, version, s.cfg.PhoneNumberID)
}

type metaAPIErrorResponse struct {
	Error metaAPIError `json:"error"`
}

type metaAPIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
}

func decodeMetaAPIError(body io.Reader) metaAPIError {
	limited, _ := io.ReadAll(io.LimitReader(body, 4096))
	var response metaAPIErrorResponse
	if err := json.Unmarshal(limited, &response); err != nil {
		return metaAPIError{}
	}
	return response.Error
}

func metaRecipient(phoneNumber string) string {
	var builder strings.Builder
	for _, r := range phoneNumber {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func stringFromAny(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
