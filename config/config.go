package config

import (
	"github.com/fekuna/omnipos-pkg/cache"
)

type Config struct {
	Server       ServerConfig
	HTTP         HTTPConfig
	GRPCServices GRPCServicesConfig
	Logger       LoggerConfig
	JWT          JWTConfig
	Redis        cache.Config
	RateLimit    RateLimitConfig
}

type ServerConfig struct {
	AppName    string
	AppEnv     string
	PrivateKey string
}

type HTTPConfig struct {
	Port string
}

type GRPCServicesConfig struct {
	MerchantServiceAddr string
	ProductServiceAddr  string
	OrderServiceAddr    string
	CustomerServiceAddr string
	PaymentServiceAddr  string
	StoreServiceAddr    string
	AuditServiceAddr    string
}

type LoggerConfig struct {
	Level             string
	Encoding          string
	DisableCaller     bool
	DisableStacktrace bool
}

type JWTConfig struct {
	SecretKey string
}

type RateLimitConfig struct {
	Enabled     bool
	PublicRPS   int
	PublicBurst int
	AuthRPS     int
	AuthBurst   int
}

func Load() (Config, error) {
	cfg := Config{
		Server: ServerConfig{
			AppName:    getEnv("APP_NAME", "omnipos-gateway"),
			AppEnv:     getEnv("APP_ENV", "dev"),
			PrivateKey: getEnvRequired("PRIVATE_KEY"),
		},
		HTTP: HTTPConfig{
			Port: getEnv("HTTP_PORT", ":8081"),
		},
		GRPCServices: GRPCServicesConfig{
			MerchantServiceAddr: getEnv("MERCHANT_GRPC_ADDR", "localhost:8080"),
			ProductServiceAddr:  getEnv("PRODUCT_GRPC_ADDR", "localhost:8082"),
			OrderServiceAddr:    getEnv("ORDER_GRPC_ADDR", "localhost:8083"),
			CustomerServiceAddr: getEnv("CUSTOMER_GRPC_ADDR", "localhost:8084"),
			PaymentServiceAddr:  getEnv("PAYMENT_GRPC_ADDR", "localhost:50054"),
			StoreServiceAddr:    getEnv("STORE_GRPC_ADDR", "localhost:50055"),
			AuditServiceAddr:    getEnv("AUDIT_GRPC_ADDR", "localhost:8086"),
		},
		Logger: LoggerConfig{
			Level:             getEnv("LOG_LEVEL", "info"),
			Encoding:          getEnv("LOG_ENCODING", "json"),
			DisableCaller:     getBoolEnv("LOG_DISABLE_CALLER", false),
			DisableStacktrace: getBoolEnv("LOG_DISABLE_STACKTRACE", false),
		},
		JWT: JWTConfig{
			SecretKey: getEnvRequired("JWT_SECRET_KEY"),
		},
		Redis: cache.Config{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		RateLimit: RateLimitConfig{
			Enabled:     getBoolEnv("RATE_LIMIT_ENABLED", true),
			PublicRPS:   getEnvInt("RATE_LIMIT_PUBLIC_RPS", 10),
			PublicBurst: getEnvInt("RATE_LIMIT_PUBLIC_BURST", 20),
			AuthRPS:     getEnvInt("RATE_LIMIT_AUTH_RPS", 100),
			AuthBurst:   getEnvInt("RATE_LIMIT_AUTH_BURST", 200),
		},
	}
	return cfg, nil
}
