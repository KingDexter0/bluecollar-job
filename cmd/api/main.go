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
	"bluecollarjob/internal/repository"
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

	repositories := repository.NewPostgresRepositories(dbPool)
	authService := service.NewAuthService(cfg.JWTSecret, cfg.JWTIssuer)
	workerService := service.NewWorkerService(repositories.Users)
	identityVerificationService := service.NewIdentityVerificationService(
		repositories.Users,
		repositories.IdentityVerifications,
		service.NewMockAadhaarGateway(),
	)
	jobService := service.NewJobService(repositories.Jobs)
	applicationService := service.NewApplicationService(repositories.Applications, repositories.Jobs, repositories.Users)
	employerService := service.NewEmployerService(repositories.Employers, repositories.Jobs, authService)
	atsService := service.NewATSService(repositories.ATS, repositories.Notifications)

	workerHandler := handler.NewWorkerHandler(workerService)
	identityVerificationHandler := handler.NewIdentityVerificationHandler(identityVerificationService)
	jobHandler := handler.NewJobHandler(jobService)
	applicationHandler := handler.NewApplicationHandler(applicationService)
	employerHandler := handler.NewEmployerHandler(employerService)
	atsHandler := handler.NewATSHandler(atsService)

	router := gin.New()
	router.Use(appmiddleware.RequestLogger())
	router.Use(appmiddleware.JSONRecovery())
	router.Use(appmiddleware.JSONErrorHandler())

	router.GET("/health", healthHandler.Check)

	api := router.Group("/api/v1")
	{
		api.POST("/workers", workerHandler.Create)
		api.GET("/workers/:id", workerHandler.GetByID)
		api.PATCH("/workers/:id/profile", workerHandler.UpdateProfile)

		api.POST("/workers/:id/identity/aadhaar/start", identityVerificationHandler.StartAadhaar)
		api.POST("/workers/:id/identity/aadhaar/verify", identityVerificationHandler.VerifyAadhaar)
		api.POST("/workers/:id/identity/document", identityVerificationHandler.MarkDocumentUploaded)
		api.POST("/workers/:id/identity/skip", identityVerificationHandler.MarkSkipped)
		api.GET("/workers/:id/identity/latest", identityVerificationHandler.GetLatest)

		api.GET("/jobs", jobHandler.ListActive)
		api.GET("/jobs/:id", jobHandler.GetByID)

		api.POST("/applications", applicationHandler.Create)
		api.GET("/applications/:id", applicationHandler.GetByID)
		api.POST("/applications/:id/interview/select-slot", atsHandler.SelectInterviewSlot)
		api.GET("/workers/:id/applications", applicationHandler.ListByWorker)

		api.POST("/employers/register", employerHandler.Register)
		api.POST("/employers/login", employerHandler.Login)

		employerRoutes := api.Group("")
		employerRoutes.Use(appmiddleware.EmployerAuth(authService))
		{
			employerRoutes.GET("/employers/me", employerHandler.GetMe)
			employerRoutes.PATCH("/employers/me", employerHandler.UpdateMe)
			employerRoutes.POST("/employer/jobs", employerHandler.CreateJob)
			employerRoutes.GET("/employer/jobs", employerHandler.ListJobs)
			employerRoutes.GET("/employer/jobs/:id", employerHandler.GetJob)
			employerRoutes.PATCH("/employer/jobs/:id", employerHandler.UpdateJob)
			employerRoutes.PATCH("/employer/jobs/:id/status", employerHandler.UpdateJobStatus)
			employerRoutes.GET("/employer/applications", atsHandler.ListEmployerApplications)
			employerRoutes.GET("/employer/jobs/:id/applications", atsHandler.ListJobApplications)
			employerRoutes.GET("/employer/applications/:id", atsHandler.GetEmployerApplication)
			employerRoutes.PATCH("/employer/applications/:id/status", atsHandler.UpdateApplicationStatus)
			employerRoutes.POST("/employer/applications/:id/interview/direct", atsHandler.ScheduleDirectInterview)
			employerRoutes.POST("/employer/applications/:id/interview/slots", atsHandler.CreateInterviewSlots)
		}
	}

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
