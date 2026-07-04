package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	WhatsAppStateKeyPrefix      = "wa_state:"
	WhatsAppRateKeyPrefix       = "wa_rate:"
	AppStatusOTPKeyPrefix       = "app_status_otp:"
	AadhaarOTPKeyPrefix         = "aadhaar_otp:"
	defaultConversationStateTTL = 24 * time.Hour
)

type ConversationState struct {
	PhoneNumber string          `json:"phone_number"`
	State       string          `json:"state"`
	Data        json.RawMessage `json:"data"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ConversationStateService interface {
	SetState(ctx context.Context, phoneNumber, state string, data json.RawMessage, ttl time.Duration) (*ConversationState, error)
	GetState(ctx context.Context, phoneNumber string) (*ConversationState, error)
	UpdateState(ctx context.Context, phoneNumber, state string, data json.RawMessage, ttl time.Duration) (*ConversationState, error)
	DeleteState(ctx context.Context, phoneNumber string) error
}

type redisConversationStateService struct {
	redis RedisStore
}

func NewRedisConversationStateService(client RedisStore) ConversationStateService {
	return &redisConversationStateService{redis: client}
}

func (s *redisConversationStateService) SetState(ctx context.Context, phoneNumber, state string, data json.RawMessage, ttl time.Duration) (*ConversationState, error) {
	phoneNumber = strings.TrimSpace(phoneNumber)
	state = strings.TrimSpace(state)
	if phoneNumber == "" || state == "" {
		return nil, fmt.Errorf("%w: phone_number and state are required", ErrInvalidInput)
	}
	if len(data) == 0 {
		data = json.RawMessage(`{}`)
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("%w: data must be valid JSON", ErrInvalidInput)
	}
	if ttl <= 0 {
		ttl = defaultConversationStateTTL
	}

	conversationState := &ConversationState{
		PhoneNumber: phoneNumber,
		State:       state,
		Data:        data,
		UpdatedAt:   time.Now().UTC(),
	}
	encoded, err := json.Marshal(conversationState)
	if err != nil {
		return nil, err
	}
	if err := s.redis.Set(ctx, conversationStateKey(phoneNumber), encoded, ttl).Err(); err != nil {
		return nil, err
	}
	return conversationState, nil
}

func (s *redisConversationStateService) GetState(ctx context.Context, phoneNumber string) (*ConversationState, error) {
	value, err := s.redis.Get(ctx, conversationStateKey(phoneNumber)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var conversationState ConversationState
	if err := json.Unmarshal(value, &conversationState); err != nil {
		return nil, err
	}
	return &conversationState, nil
}

func (s *redisConversationStateService) UpdateState(ctx context.Context, phoneNumber, state string, data json.RawMessage, ttl time.Duration) (*ConversationState, error) {
	return s.SetState(ctx, phoneNumber, state, data, ttl)
}

func (s *redisConversationStateService) DeleteState(ctx context.Context, phoneNumber string) error {
	if err := s.redis.Del(ctx, conversationStateKey(phoneNumber)).Err(); err != nil {
		return err
	}
	return nil
}

func conversationStateKey(phoneNumber string) string {
	return WhatsAppStateKeyPrefix + strings.TrimSpace(phoneNumber)
}
