package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type ComponentStatus struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type HealthResponse struct {
	Status    string          `json:"status"`
	Postgres  ComponentStatus `json:"postgres"`
	Redis     ComponentStatus `json:"redis"`
	CheckedAt time.Time       `json:"checked_at"`
}

type HealthService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthService(db *pgxpool.Pool, redisClient *redis.Client) *HealthService {
	return &HealthService{
		db:    db,
		redis: redisClient,
	}
}

func (s *HealthService) Check(ctx context.Context) HealthResponse {
	postgresStatus := ComponentStatus{Status: "ok"}
	redisStatus := ComponentStatus{Status: "ok"}

	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := s.db.Ping(checkCtx); err != nil {
		postgresStatus = ComponentStatus{Status: "unavailable", Error: err.Error()}
	}

	if err := s.redis.Ping(checkCtx).Err(); err != nil {
		redisStatus = ComponentStatus{Status: "unavailable", Error: err.Error()}
	}

	apiStatus := "ok"
	if postgresStatus.Status != "ok" || redisStatus.Status != "ok" {
		apiStatus = "degraded"
	}

	return HealthResponse{
		Status:    apiStatus,
		Postgres:  postgresStatus,
		Redis:     redisStatus,
		CheckedAt: time.Now().UTC(),
	}
}
