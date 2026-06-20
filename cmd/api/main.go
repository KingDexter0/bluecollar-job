package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bluecollarjob/internal/cache"
	"bluecollarjob/internal/config"
	"bluecollarjob/internal/database"
	"bluecollarjob/internal/handler"
	appmiddleware "bluecollarjob/internal/middleware"
	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()

	dbPool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer dbPool.Close()

	redisClient := cache.NewRedisClient(cfg.Redis)
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("close redis: %v", err)
		}
	}()

	healthService := service.NewHealthService(dbPool, redisClient)
	healthHandler := handler.NewHealthHandler(healthService)

	router := gin.New()
	router.Use(appmiddleware.RequestLogger())
	router.Use(appmiddleware.JSONRecovery())
	router.Use(appmiddleware.JSONErrorHandler())

	router.GET("/health", healthHandler.Check)
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, appmiddleware.ErrorResponse{
			Error: appmiddleware.ErrorBody{
				Code:    "not_found",
				Message: "route not found",
			},
		})
	})

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api listening on :%s", cfg.AppPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}

	log.Println("api stopped")
}
