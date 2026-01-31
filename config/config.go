package config

type Config struct {
	Server       ServerConfig
	HTTP         HTTPConfig
	GRPCServices GRPCServicesConfig
	Logger       LoggerConfig
	JWT          JWTConfig
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
	UserServiceAddr string
	// Add more service addresses as needed
	// ProductServiceAddr string
	// OrderServiceAddr   string
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
			UserServiceAddr: getEnvRequired("USER_GRPC_ADDR"),
			// Add more service addresses as needed
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
	}
	return cfg, nil
}
