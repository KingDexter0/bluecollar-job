package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	statusOTPTTL          = 5 * time.Minute
	statusOTPRetryLimit   = 3
	statusOTPRateLimit    = 5
	statusOTPRateLimitTTL = time.Hour
)

type StatusOTPService interface {
	Generate(ctx context.Context, phoneNumber string) (*StatusOTPResult, error)
	Verify(ctx context.Context, phoneNumber, transactionID, otp string) error
}

type StatusOTPResult struct {
	PhoneNumber    string    `json:"phone_number"`
	TransactionID  string    `json:"transaction_id"`
	ExpiresAt      time.Time `json:"expires_at"`
	OTPForLocalDev string    `json:"otp_for_local_dev,omitempty"`
}

type redisStatusOTPService struct {
	redis  RedisStore
	pepper string
}

type storedStatusOTP struct {
	TransactionID string    `json:"transaction_id"`
	OTPHash       string    `json:"otp_hash"`
	Attempts      int       `json:"attempts"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewRedisStatusOTPService(client RedisStore, pepper string) StatusOTPService {
	return &redisStatusOTPService{redis: client, pepper: pepper}
}

func (s *redisStatusOTPService) Generate(ctx context.Context, phoneNumber string) (*StatusOTPResult, error) {
	phoneNumber = strings.TrimSpace(phoneNumber)
	if phoneNumber == "" {
		return nil, fmt.Errorf("%w: phone_number is required", ErrInvalidInput)
	}

	rateCount, err := s.redis.Incr(ctx, rateLimitKey(phoneNumber)).Result()
	if err != nil {
		return nil, err
	}
	if rateCount == 1 {
		if err := s.redis.Expire(ctx, rateLimitKey(phoneNumber), statusOTPRateLimitTTL).Err(); err != nil {
			return nil, err
		}
	}
	if rateCount > statusOTPRateLimit {
		return nil, fmt.Errorf("%w: OTP rate limit exceeded", ErrConflict)
	}

	otp, err := generateSixDigitOTP()
	if err != nil {
		return nil, err
	}
	transactionID, err := generateTransactionID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	stored := storedStatusOTP{
		TransactionID: transactionID,
		OTPHash:       hashOTP(transactionID, otp, s.pepper),
		Attempts:      0,
		CreatedAt:     now,
	}
	encoded, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}
	if err := s.redis.Set(ctx, statusOTPKey(phoneNumber), encoded, statusOTPTTL).Err(); err != nil {
		return nil, err
	}

	return &StatusOTPResult{
		PhoneNumber:    phoneNumber,
		TransactionID:  transactionID,
		ExpiresAt:      now.Add(statusOTPTTL),
		OTPForLocalDev: otp,
	}, nil
}

func (s *redisStatusOTPService) Verify(ctx context.Context, phoneNumber, transactionID, otp string) error {
	phoneNumber = strings.TrimSpace(phoneNumber)
	transactionID = strings.TrimSpace(transactionID)
	otp = strings.TrimSpace(otp)
	if phoneNumber == "" || transactionID == "" || otp == "" {
		return fmt.Errorf("%w: phone_number, transaction_id, and otp are required", ErrInvalidInput)
	}

	key := statusOTPKey(phoneNumber)
	value, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrNotFound
		}
		return err
	}

	var stored storedStatusOTP
	if err := json.Unmarshal(value, &stored); err != nil {
		return err
	}
	if stored.Attempts >= statusOTPRetryLimit {
		return fmt.Errorf("%w: OTP retry limit exceeded", ErrConflict)
	}
	if stored.TransactionID != transactionID {
		if err := s.incrementAttempts(ctx, key, stored); err != nil {
			return err
		}
		return fmt.Errorf("%w: invalid OTP", ErrInvalidInput)
	}

	expectedHash := hashOTP(transactionID, otp, s.pepper)
	if subtle.ConstantTimeCompare([]byte(expectedHash), []byte(stored.OTPHash)) != 1 {
		if err := s.incrementAttempts(ctx, key, stored); err != nil {
			return err
		}
		return fmt.Errorf("%w: invalid OTP", ErrInvalidInput)
	}

	return s.redis.Del(ctx, key).Err()
}

func (s *redisStatusOTPService) incrementAttempts(ctx context.Context, key string, stored storedStatusOTP) error {
	stored.Attempts++
	encoded, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	ttl, err := s.redis.TTL(ctx, key).Result()
	if err != nil {
		return err
	}
	if ttl <= 0 {
		ttl = statusOTPTTL
	}
	return s.redis.Set(ctx, key, encoded, ttl).Err()
}

func statusOTPKey(phoneNumber string) string {
	return AppStatusOTPKeyPrefix + strings.TrimSpace(phoneNumber)
}

func rateLimitKey(phoneNumber string) string {
	return WhatsAppRateKeyPrefix + strings.TrimSpace(phoneNumber)
}

func hashOTP(transactionID, otp, pepper string) string {
	sum := sha256.Sum256([]byte(transactionID + ":" + otp + ":" + pepper))
	return hex.EncodeToString(sum[:])
}

func generateSixDigitOTP() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func generateTransactionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
