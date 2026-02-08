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
	customRuntime "github.com/fekuna/omnipos-gateway/internal/runtime"
	"github.com/fekuna/omnipos-gateway/internal/swagger"
	"github.com/fekuna/omnipos-pkg/cache"
	"github.com/fekuna/omnipos-pkg/logger"
	auditv1 "github.com/fekuna/omnipos-proto/proto/audit/v1"
	customerv1 "github.com/fekuna/omnipos-proto/proto/customer/v1"
	orderv1 "github.com/fekuna/omnipos-proto/proto/order/v1"
	paymentv1 "github.com/fekuna/omnipos-proto/proto/payment/v1"
	productv1 "github.com/fekuna/omnipos-proto/proto/product/v1"
	storev1 "github.com/fekuna/omnipos-proto/proto/store/v1"
	userv1 "github.com/fekuna/omnipos-proto/proto/user/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	// Load environment variables from .env file (will not override existing env vars)
	if err := godotenv.Load(); err != nil {
		// handle error if .env file is missing, which is fine for docker
	}

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
		runtime.WithMarshalerOption(runtime.MIMEWildcard, customRuntime.NewCustomMarshaler()),
		runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
			// Get standard metadata from our custom annotator (lang, timezone)
			md := middleware.MetadataAnnotator(ctx, req)

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
	log.Info("Connecting to merchant service", zap.String("addr", cfg.GRPCServices.MerchantServiceAddr))
	err = userv1.RegisterMerchantServiceHandlerFromEndpoint(
		ctx,
		mux,
		cfg.GRPCServices.MerchantServiceAddr,
		opts,
	)
	if err != nil {
		log.Fatal("failed to register user service handler", zap.Error(err))
	}

	// Register RoleService
	err = userv1.RegisterRoleServiceHandlerFromEndpoint(
		ctx,
		mux,
		cfg.GRPCServices.MerchantServiceAddr, // RoleService is hosted in User Service (MerchantServiceAddr)
		opts,
	)
	if err != nil {
		log.Fatal("failed to register role service handler", zap.Error(err))
	}

	// Register UserService
	err = userv1.RegisterUserServiceHandlerFromEndpoint(
		ctx,
		mux,
		cfg.GRPCServices.MerchantServiceAddr, // UserService is hosted in User Service (MerchantServiceAddr)
		opts,
	)
	if err != nil {
		log.Fatal("failed to register user service handler (staff)", zap.Error(err))
	}

	// Register other service handlers here
	// Product Service
	log.Info("Connecting to product service", zap.String("addr", cfg.GRPCServices.ProductServiceAddr))

	// Register ProductService
	err = productv1.RegisterProductServiceHandlerFromEndpoint(
		ctx,
		mux,
		cfg.GRPCServices.ProductServiceAddr,
		opts,
	)
	if err != nil {
		log.Fatal("failed to register product service handler", zap.Error(err))
	}

	// Register CategoryService
	err = productv1.RegisterCategoryServiceHandlerFromEndpoint(
		ctx,
		mux,
		cfg.GRPCServices.ProductServiceAddr,
		opts,
	)
	if err != nil {
		log.Fatal("failed to register category service handler", zap.Error(err))
	}

	// Register InventoryService
	err = productv1.RegisterInventoryServiceHandlerFromEndpoint(
		ctx,
		mux,
		cfg.GRPCServices.ProductServiceAddr,
		opts,
	)
	if err != nil {
		log.Fatal("failed to register inventory service handler", zap.Error(err))
	}

	// Register ProductVariantService
	err = productv1.RegisterProductVariantServiceHandlerFromEndpoint(
		ctx,
		mux,
		cfg.GRPCServices.ProductServiceAddr,
		opts,
	)
	if err != nil {
		log.Fatal("failed to register product variant service handler", zap.Error(err))
	}

	log.Info("Product service handlers registered")

	// Register OrderService
	log.Info("Connecting to order service", zap.String("addr", cfg.GRPCServices.OrderServiceAddr))
	err = orderv1.RegisterOrderServiceHandlerFromEndpoint(
		ctx,
		mux,
		cfg.GRPCServices.OrderServiceAddr,
		opts,
	)
	if err != nil {
		log.Fatal("failed to register order service handler", zap.Error(err))
	}
	log.Info("Order service handler registered")

	// Register CustomerService
	log.Info("Connecting to customer service", zap.String("addr", cfg.GRPCServices.CustomerServiceAddr))
	err = customerv1.RegisterCustomerServiceHandlerFromEndpoint(
		ctx,
		mux,
		cfg.GRPCServices.CustomerServiceAddr,
		opts,
	)
	if err != nil {
		log.Fatal("failed to register customer service handler", zap.Error(err))
	}
	log.Info("Customer service handler registered")

	// Register PaymentService
	log.Info("Connecting to payment service", zap.String("addr", cfg.GRPCServices.PaymentServiceAddr))
	err = paymentv1.RegisterPaymentServiceHandlerFromEndpoint(
		ctx,
		mux,
		cfg.GRPCServices.PaymentServiceAddr,
		opts,
	)
	if err != nil {
		log.Fatal("failed to register payment service handler", zap.Error(err))
	}
	log.Info("Payment service handler registered")

	// Register StoreService
	log.Info("Connecting to store service", zap.String("addr", cfg.GRPCServices.StoreServiceAddr))
	err = storev1.RegisterStoreServiceHandlerFromEndpoint(
		ctx,
		mux,
		cfg.GRPCServices.StoreServiceAddr,
		opts,
	)
	if err != nil {
		log.Fatal("failed to register store service handler", zap.Error(err))
	}
	log.Info("Store service handler registered")

	// Register AuditService
	log.Info("Connecting to audit service", zap.String("addr", cfg.GRPCServices.AuditServiceAddr))
	err = auditv1.RegisterAuditServiceHandlerFromEndpoint(
		ctx,
		mux,
		cfg.GRPCServices.AuditServiceAddr,
		opts,
	)
	if err != nil {
		log.Fatal("failed to register audit service handler", zap.Error(err))
	}
	log.Info("Audit service handler registered")

	log.Info("User service handler registered")

	// Create HTTP handler using grpc-gateway mux
	httpMux := http.NewServeMux()

	// Register gRPC-Gateway routes
	httpMux.Handle("/", mux)

	// Initialize and register Swagger UI
	swaggerHandler := swagger.NewHandler(log)
	swaggerHandler.RegisterRoutes(httpMux)

	// Initialize Redis client
	redisClient, err := cache.NewRedisClient(&cfg.Redis)
	if err != nil {
		log.Fatal("failed to initialize redis client", zap.Error(err))
	}
	defer redisClient.Close()
	log.Info("Redis client initialized")

	// Initialize Rate Limiter
	rateLimiter := middleware.NewRateLimiter(redisClient, cfg.RateLimit, log)

	// Apply CORS middleware and Rate Limiter
	// Order: CORS -> RateLimit -> Mux
	handler := middleware.CORS(rateLimiter.Limit(httpMux))

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.HTTP.Port,
		Handler:      handler,
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
