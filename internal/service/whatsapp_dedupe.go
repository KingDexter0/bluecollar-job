package service

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	WhatsAppMessageKeyPrefix = "wa_msg:"
	defaultWebhookDedupeTTL  = 48 * time.Hour
)

type WhatsAppMessageDeduplicator interface {
	MarkProcessed(ctx context.Context, messageID string) (bool, error)
}

type RedisWhatsAppMessageDeduplicator struct {
	redis RedisSetNXStore
	ttl   time.Duration
}

type RedisSetNXStore interface {
	RedisStore
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
}

func NewRedisWhatsAppMessageDeduplicator(redis RedisSetNXStore, ttl time.Duration) *RedisWhatsAppMessageDeduplicator {
	if ttl <= 0 {
		ttl = defaultWebhookDedupeTTL
	}
	return &RedisWhatsAppMessageDeduplicator{redis: redis, ttl: ttl}
}

func (d *RedisWhatsAppMessageDeduplicator) MarkProcessed(ctx context.Context, messageID string) (bool, error) {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return true, nil
	}
	return d.redis.SetNX(ctx, WhatsAppMessageKeyPrefix+messageID, "1", d.ttl).Result()
}
