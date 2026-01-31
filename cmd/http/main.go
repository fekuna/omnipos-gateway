package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fekuna/omnipos-gateway/config"
	"github.com/fekuna/omnipos-gateway/internal/middleware"
	"github.com/fekuna/omnipos-gateway/internal/swagger"
	"github.com/fekuna/omnipos-pkg/logger"
	userv1 "github.com/fekuna/omnipos-proto/proto/user/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	// Load environment variables
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Initialize logger
	loggerCfg := logger.ZapLoggerConfig{
		IsDevelopment:     cfg.Server.AppEnv == "dev",
		Level:             cfg.Logger.Level,
		Encoding:          cfg.Logger.Encoding,
		DisableCaller:     cfg.Logger.DisableCaller,
		DisableStacktrace: cfg.Logger.DisableStacktrace,
	}

	log := logger.NewZapLogger(&loggerCfg)
	defer log.Sync()

	log.Info("Logger initialized")

	// Create context
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Initialize JWT helper
	jwtHelper := middleware.NewJWTHelper(cfg.JWT.SecretKey)
	log.Info("JWT helper initialized")

	// Discover public endpoints from proto definitions
	publicEndpoints, err := middleware.DiscoverPublicEndpoints()
	if err != nil {
		log.Fatal("failed to discover public endpoints", zap.Error(err))
	}
	log.Info("Discovered public endpoints from proto definitions", zap.Int("count", len(publicEndpoints)))
	for endpoint := range publicEndpoints {
		log.Debug("public endpoint", zap.String("method", endpoint))
	}

	// Initialize auth interceptor with proto-based public endpoints
	authInterceptor := middleware.NewAuthInterceptor(jwtHelper, log, publicEndpoints)
	log.Info("Auth interceptor initialized")

	// Initialize grpc-gateway mux with custom header matcher and metadata annotator
	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(middleware.HTTPHeaderMatcher),
		runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
			md := metadata.MD{}
			// Explicitly forward Authorization header
			if auth := req.Header.Get("Authorization"); auth != "" {
				md.Set("authorization", auth)
			}
			return md
		}),
	)

	// gRPC dial options with authentication interceptor
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(authInterceptor.Unary()),
	}

	// Register user service handler (auto-generated from proto annotations!)
	log.Info("Connecting to user service", zap.String("addr", cfg.GRPCServices.UserServiceAddr))
	err = userv1.RegisterMerchantServiceHandlerFromEndpoint(
		ctx,
		mux,
		cfg.GRPCServices.UserServiceAddr,
		opts,
	)
	if err != nil {
		log.Fatal("failed to register user service handler", zap.Error(err))
	}

	// TODO: Register other service handlers here
	// Example:
	// err = productv1.RegisterProductServiceHandlerFromEndpoint(ctx, mux, cfg.GRPCServices.ProductServiceAddr, opts)

	log.Info("User service handler registered")

	// Create HTTP handler using grpc-gateway mux
	httpMux := http.NewServeMux()

	// Register gRPC-Gateway routes
	httpMux.Handle("/", mux)

	// Initialize and register Swagger UI
	swaggerHandler := swagger.NewHandler(log)
	swaggerHandler.RegisterRoutes(httpMux)

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.HTTP.Port,
		Handler:      httpMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Info("grpc-gateway server started (zero routing logic!)", zap.String("port", cfg.HTTP.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Info("shutting down grpc-gateway server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", zap.Error(err))
	}

	log.Info("server shutdown complete")
}
