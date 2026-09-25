package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"industrial-noise-source-attribution/backend/internal/config"
	"industrial-noise-source-attribution/backend/internal/handler"
	"industrial-noise-source-attribution/backend/internal/middleware"
	"industrial-noise-source-attribution/backend/internal/repository"
	"industrial-noise-source-attribution/backend/internal/router"
	"industrial-noise-source-attribution/backend/internal/service"
	"industrial-noise-source-attribution/backend/internal/util"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	configuration, err := config.Load()
	if err != nil {
		return err
	}
	db, err := config.OpenDatabase(configuration)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := config.CloseDatabase(db); closeErr != nil {
			logger.Error("database close failed", "error", closeErr)
		}
	}()

	pointRepository := repository.NewMonitoringPointRepository(db)
	measurementRepository := repository.NewNoiseMeasurementRepository(db)
	sourceRepository := repository.NewSourceProfileRepository(db)
	runRepository := repository.NewAttributionRunRepository(db)
	supportRepository := repository.NewSupportRepository(db)

	pointHandler := handler.NewMonitoringPointHandler(service.NewMonitoringPointService(pointRepository))
	measurementHandler := handler.NewNoiseMeasurementHandler(service.NewNoiseMeasurementService(measurementRepository, pointRepository))
	sourceHandler := handler.NewSourceProfileHandler(service.NewSourceProfileService(sourceRepository))
	runHandler := handler.NewAttributionRunHandler(service.NewAttributionRunService(runRepository, measurementRepository, sourceRepository))
	supportHandler := handler.NewSupportHandler(service.NewAccessService(supportRepository, configuration.JWTSecret, configuration.JWTExpiry))

	authenticator := middleware.NewAuthenticator(configuration.JWTSecret)
	loginLimiter := middleware.NewRateLimiter(configuration.LoginLimitPerMinute)
	importLimiter := middleware.NewRateLimiter(configuration.ImportLimitPerMinute)
	attributionLimiter := middleware.NewRateLimiter(configuration.AttributionLimitPerMinute)

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(middleware.RequestID(), middleware.Recovery(logger), middleware.ErrorHandler(logger))
	engine.GET("/healthz", func(c *gin.Context) {
		if err := config.Ping(c.Request.Context(), db); err != nil {
			util.RespondError(c, util.NewError(http.StatusServiceUnavailable, "database_unavailable", "数据库暂不可用", nil))
			return
		}
		util.Respond(c, http.StatusOK, gin.H{
			"status": "healthy", "service": "industrial-noise-source-attribution",
			"product": "NoiseTrace 噪声源贡献归因台", "time": time.Now().UTC(),
		})
	})
	v1 := engine.Group("/api/v1")
	api := v1.Group("")
	api.Use(authenticator.RequireAuth(), middleware.AuditBoundary(logger))
	router.RegisterSupportRoutes(v1, api, supportHandler, loginLimiter)
	router.RegisterMonitoringPointRoutes(api, pointHandler)
	router.RegisterNoiseMeasurementRoutes(api, measurementHandler, importLimiter)
	router.RegisterSourceProfileRoutes(api, sourceHandler)
	router.RegisterAttributionRunRoutes(api, runHandler, attributionLimiter)
	engine.NoRoute(func(c *gin.Context) {
		util.RespondError(c, util.NotFound("请求路由不存在"))
	})

	server := &http.Server{
		Addr: ":" + configuration.Port, Handler: engine,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second,
		IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20,
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	serverError := make(chan error, 1)
	go func() { serverError <- server.ListenAndServe() }()
	logger.Info("server started", "port", configuration.Port, "database_driver", configuration.DBDriver)
	select {
	case listenErr := <-serverError:
		if !errors.Is(listenErr, http.ErrServerClosed) {
			return listenErr
		}
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), configuration.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
	}
	return nil
}
