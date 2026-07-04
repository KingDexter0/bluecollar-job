package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisConversationStateSetGetUpdateDelete(t *testing.T) {
	ctx := context.Background()
	store := newFakeRedisStore()
	states := NewRedisConversationStateService(store)

	created, err := states.SetState(ctx, "+919876543210", "awaiting_role", json.RawMessage(`{"step":1}`), time.Minute)
	if err != nil {
		t.Fatalf("set state: %v", err)
	}
	if created.State != "awaiting_role" {
		t.Fatalf("expected awaiting_role, got %s", created.State)
	}

	got, err := states.GetState(ctx, "+919876543210")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if got.State != "awaiting_role" {
		t.Fatalf("expected awaiting_role, got %s", got.State)
	}

	updated, err := states.UpdateState(ctx, "+919876543210", "awaiting_zone", json.RawMessage(`{"step":2}`), time.Minute)
	if err != nil {
		t.Fatalf("update state: %v", err)
	}
	if updated.State != "awaiting_zone" {
		t.Fatalf("expected awaiting_zone, got %s", updated.State)
	}

	if err := states.DeleteState(ctx, "+919876543210"); err != nil {
		t.Fatalf("delete state: %v", err)
	}
	if _, err := states.GetState(ctx, "+919876543210"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestStatusOTPGenerateAndVerify(t *testing.T) {
	ctx := context.Background()
	otps := NewRedisStatusOTPService(newFakeRedisStore(), "test-pepper")

	generated, err := otps.Generate(ctx, "+919876543210")
	if err != nil {
		t.Fatalf("generate otp: %v", err)
	}
	if generated.TransactionID == "" || generated.OTPForLocalDev == "" {
		t.Fatalf("expected transaction and dev OTP, got %#v", generated)
	}

	if err := otps.Verify(ctx, "+919876543210", generated.TransactionID, generated.OTPForLocalDev); err != nil {
		t.Fatalf("verify otp: %v", err)
	}
	if err := otps.Verify(ctx, "+919876543210", generated.TransactionID, generated.OTPForLocalDev); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected otp deleted after success, got %v", err)
	}
}

func TestStatusOTPInvalidOTPRejected(t *testing.T) {
	ctx := context.Background()
	otps := NewRedisStatusOTPService(newFakeRedisStore(), "test-pepper")

	generated, err := otps.Generate(ctx, "+919876543210")
	if err != nil {
		t.Fatalf("generate otp: %v", err)
	}
	if err := otps.Verify(ctx, "+919876543210", generated.TransactionID, "000000"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input for wrong OTP, got %v", err)
	}
}

type fakeRedisStore struct {
	mu     sync.Mutex
	values map[string]fakeRedisValue
}

type fakeRedisValue struct {
	value     string
	expiresAt time.Time
}

func newFakeRedisStore() *fakeRedisStore {
	return &fakeRedisStore{values: map[string]fakeRedisValue{}}
}

func (s *fakeRedisStore) Get(ctx context.Context, key string) *redis.StringCmd {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	if !ok || value.expired() {
		delete(s.values, key)
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(value.value, nil)
}

func (s *fakeRedisStore) Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
	s.mu.Lock()
	defer s.mu.Unlock()
	var encoded string
	switch typed := value.(type) {
	case []byte:
		encoded = string(typed)
	case string:
		encoded = typed
	default:
		encoded = fmt.Sprint(typed)
	}
	s.values[key] = fakeRedisValue{value: encoded, expiresAt: expiryFromDuration(expiration)}
	return redis.NewStatusResult("OK", nil)
}

func (s *fakeRedisStore) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	s.mu.Lock()
	defer s.mu.Unlock()
	var deleted int64
	for _, key := range keys {
		if _, ok := s.values[key]; ok {
			deleted++
			delete(s.values, key)
		}
	}
	return redis.NewIntResult(deleted, nil)
}

func (s *fakeRedisStore) Incr(ctx context.Context, key string) *redis.IntCmd {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	if !ok || value.expired() {
		s.values[key] = fakeRedisValue{value: "1"}
		return redis.NewIntResult(1, nil)
	}
	var current int64
	_, _ = fmt.Sscanf(value.value, "%d", &current)
	current++
	value.value = fmt.Sprintf("%d", current)
	s.values[key] = value
	return redis.NewIntResult(current, nil)
}

func (s *fakeRedisStore) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	if !ok {
		return redis.NewBoolResult(false, nil)
	}
	value.expiresAt = expiryFromDuration(expiration)
	s.values[key] = value
	return redis.NewBoolResult(true, nil)
}

func (s *fakeRedisStore) TTL(ctx context.Context, key string) *redis.DurationCmd {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	if !ok || value.expired() {
		delete(s.values, key)
		return redis.NewDurationResult(-2*time.Second, nil)
	}
	if value.expiresAt.IsZero() {
		return redis.NewDurationResult(-1*time.Second, nil)
	}
	return redis.NewDurationResult(time.Until(value.expiresAt), nil)
}

func (v fakeRedisValue) expired() bool {
	return !v.expiresAt.IsZero() && time.Now().After(v.expiresAt)
}

func expiryFromDuration(duration time.Duration) time.Time {
	if duration <= 0 {
		return time.Time{}
	}
	return time.Now().Add(duration)
}
