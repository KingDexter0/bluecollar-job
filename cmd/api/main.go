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
	"bluecollarjob/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stopRuntime := context.WithCancel(context.Background())
	defer stopRuntime()

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
	authService := service.NewAuthService(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTL)
	referralService := service.NewReferralService(repositories.Users, repositories.Referrals, repositories.Notifications, service.NewMockReferralPayoutGateway())
	workerService := service.NewWorkerService(repositories.Users, referralService)
	identityVerificationService := service.NewIdentityVerificationService(
		repositories.Users,
		repositories.IdentityVerifications,
		service.NewMockAadhaarGateway(),
		referralService,
	)
	jobService := service.NewJobService(repositories.Jobs)
	applicationService := service.NewApplicationService(repositories.Applications, repositories.Jobs, repositories.Users, repositories.Notifications)
	employerService := service.NewEmployerService(repositories.Employers, repositories.Jobs, authService)
	atsService := service.NewATSService(repositories.ATS, repositories.Notifications)
	adminService := service.NewAdminService(repositories.Admin, repositories.Referrals)
	whatsAppSender, err := buildWhatsAppSender(cfg)
	if err != nil {
		log.Fatalf("configure WhatsApp sender: %v", err)
	}
	objectStore, err := buildObjectStore(cfg)
	if err != nil {
		log.Fatalf("configure object storage: %v", err)
	}
	mediaDownloader := buildMediaDownloader(cfg, objectStore)
	messageDeduplicator := service.NewRedisWhatsAppMessageDeduplicator(redisClient, 48*time.Hour)
	notificationWorkerService := service.NewNotificationWorkerService(
		repositories.Notifications,
		whatsAppSender,
		service.NotificationWorkerConfig{
			WorkerCount:  cfg.NotificationWorker.Count,
			PollInterval: time.Duration(cfg.NotificationWorker.PollIntervalSeconds) * time.Second,
			BatchSize:    10,
		},
	)
	notificationQueryService := service.NewNotificationQueryService(repositories.Notifications)
	conversationStateService := service.NewRedisConversationStateService(redisClient)
	statusOTPService := service.NewRedisStatusOTPService(redisClient, cfg.JWTSecret)
	whatsAppBotService := service.NewWhatsAppBotService(repositories.Users, applicationService, jobService, identityVerificationService, referralService, conversationStateService, statusOTPService, whatsAppSender)

	workerHandler := handler.NewWorkerHandler(workerService)
	identityVerificationHandler := handler.NewIdentityVerificationHandler(identityVerificationService)
	jobHandler := handler.NewJobHandler(jobService)
	applicationHandler := handler.NewApplicationHandler(applicationService)
	employerHandler := handler.NewEmployerHandler(employerService)
	atsHandler := handler.NewATSHandler(atsService)
	devHandler := handler.NewDevHandler(notificationWorkerService, notificationQueryService, conversationStateService, statusOTPService, referralService)
	adminHandler := handler.NewAdminHandler(adminService, notificationQueryService, referralService)
	whatsAppHandler := handler.NewWhatsAppHandler(cfg.WhatsApp.VerifyToken, whatsAppBotService, messageDeduplicator, mediaDownloader, cfg.DocumentUpload.Enabled)
	referralHandler := handler.NewReferralHandler(referralService)

	if cfg.NotificationWorker.Enabled {
		notificationWorkerService.Start(ctx)
	}

	router := gin.New()
	metrics := appmiddleware.NewMetrics()
	publicLimiter := appmiddleware.NewRateLimiter(120, time.Minute)
	authLimiter := appmiddleware.NewRateLimiter(10, time.Minute)
	otpLimiter := appmiddleware.NewRateLimiter(12, time.Minute)
	webhookLimiter := appmiddleware.NewRateLimiter(120, time.Minute)

	router.Use(appmiddleware.RequestID())
	router.Use(appmiddleware.SecureHeaders())
	router.Use(appmiddleware.CORS(cfg.CORSAllowedOrigins, cfg.IsDevelopment()))
	router.Use(metrics.Middleware())
	router.Use(appmiddleware.RequestLogger(cfg.AppEnv))
	router.Use(appmiddleware.JSONRecovery())
	router.Use(appmiddleware.JSONErrorHandler())

	router.GET("/health", healthHandler.Check)
	router.GET("/ready", healthHandler.Ready)
	router.GET("/live", healthHandler.Live)
	router.GET("/metrics", metrics.Handler)

	api := router.Group("/api/v1")
	{
		api.POST("/workers", publicLimiter.Middleware("workers-create"), workerHandler.Create)
		api.GET("/workers/:id", publicLimiter.Middleware("workers-read"), workerHandler.GetByID)
		api.PATCH("/workers/:id/profile", workerHandler.UpdateProfile)
		api.GET("/workers/:id/referral", referralHandler.GetReferral)
		api.GET("/workers/:id/referrals", referralHandler.ListReferrals)
		api.GET("/workers/:id/referral-transactions", referralHandler.ListTransactions)

		api.POST("/workers/:id/identity/aadhaar/start", otpLimiter.Middleware("aadhaar-start"), identityVerificationHandler.StartAadhaar)
		api.POST("/workers/:id/identity/aadhaar/verify", otpLimiter.Middleware("aadhaar-verify"), identityVerificationHandler.VerifyAadhaar)
		api.POST("/workers/:id/identity/document", otpLimiter.Middleware("document-verify"), identityVerificationHandler.MarkDocumentUploaded)
		api.POST("/workers/:id/identity/skip", otpLimiter.Middleware("identity-skip"), identityVerificationHandler.MarkSkipped)
		api.GET("/workers/:id/identity/latest", identityVerificationHandler.GetLatest)

		api.GET("/jobs", publicLimiter.Middleware("jobs-list"), jobHandler.ListActive)
		api.GET("/jobs/:id", publicLimiter.Middleware("jobs-read"), jobHandler.GetByID)

		api.POST("/applications", publicLimiter.Middleware("applications-create"), applicationHandler.Create)
		api.GET("/applications/:id", applicationHandler.GetByID)
		api.POST("/applications/:id/interview/select-slot", atsHandler.SelectInterviewSlot)
		api.GET("/workers/:id/applications", applicationHandler.ListByWorker)

		api.POST("/employers/register", authLimiter.Middleware("employer-register"), employerHandler.Register)
		api.POST("/employers/login", authLimiter.Middleware("employer-login"), employerHandler.Login)

		api.GET("/whatsapp/webhook", whatsAppHandler.VerifyWebhook)
		api.POST("/whatsapp/webhook", webhookLimiter.Middleware("whatsapp-webhook"), whatsAppHandler.ReceiveWebhook)

		adminRoutes := api.Group("/admin")
		adminRoutes.Use(appmiddleware.AdminTokenAuth(cfg.AdminToken))
		{
			adminRoutes.GET("/summary", adminHandler.GetSummary)
			adminRoutes.GET("/notifications", adminHandler.ListNotifications)
			adminRoutes.GET("/referral-transactions", adminHandler.ListReferralTransactions)
			adminRoutes.POST("/referrals/process-payouts", adminHandler.ProcessReferralPayouts)
		}

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

		if cfg.IsDevelopment() {
			devRoutes := api.Group("/dev")
			{
				devRoutes.GET("/notifications", devHandler.ListNotifications)
				devRoutes.POST("/notifications/process-once", devHandler.ProcessNotificationsOnce)
				devRoutes.POST("/redis/state", devHandler.SetRedisState)
				devRoutes.GET("/redis/state/:phone", devHandler.GetRedisState)
				devRoutes.DELETE("/redis/state/:phone", devHandler.DeleteRedisState)
				devRoutes.POST("/status-otp/generate", devHandler.GenerateStatusOTP)
				devRoutes.POST("/status-otp/verify", devHandler.VerifyStatusOTP)
				devRoutes.POST("/referrals/process-payouts", devHandler.ProcessReferralPayouts)
			}
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
	stopRuntime()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}

	log.Println("api stopped")
}

func buildWhatsAppSender(cfg *config.Config) (service.WhatsAppSender, error) {
	if cfg.WhatsApp.Provider == "meta" {
		return service.NewMetaWhatsAppSender(service.MetaWhatsAppConfig{
			AccessToken:     cfg.WhatsApp.AccessToken,
			PhoneNumberID:   cfg.WhatsApp.PhoneNumberID,
			GraphAPIVersion: cfg.WhatsApp.GraphAPIVersion,
		})
	}
	return service.NewMockWhatsAppSender(), nil
}

func buildObjectStore(cfg *config.Config) (storage.ObjectStore, error) {
	switch cfg.ObjectStorage.Provider {
	case "linode", "s3":
		return storage.NewLinodeObjectStore(storage.LinodeObjectStoreConfig{
			Bucket:          cfg.ObjectStorage.Bucket,
			Region:          cfg.ObjectStorage.Region,
			Endpoint:        cfg.ObjectStorage.Endpoint,
			AccessKeyID:     cfg.ObjectStorage.AccessKeyID,
			SecretAccessKey: cfg.ObjectStorage.SecretAccessKey,
		})
	default:
		return storage.NewLocalObjectStore(cfg.ObjectStorage.LocalBasePath), nil
	}
}

func buildMediaDownloader(cfg *config.Config, objectStore storage.ObjectStore) service.MediaDownloader {
	if cfg.WhatsApp.Provider == "meta" {
		return service.NewMetaMediaDownloader(cfg.WhatsApp.AccessToken, cfg.WhatsApp.GraphAPIVersion, objectStore)
	}
	return service.NewMockMediaDownloader()
}
