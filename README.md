# omnipos-gateway

HTTP-to-gRPC Gateway for the OmniPOS system. Acts as a reverse proxy that translates RESTful HTTP API calls into gRPC requests to backend microservices.

## Architecture

The gateway serves as the entry point for HTTP/REST clients and forwards requests to backend gRPC services:

```
HTTP/REST Clients → omnipos-gateway (HTTP) → Backend gRPC Services → Database
                                                   ↓
                                            (user-service, product-service, etc.)
```

### Key Responsibilities

- **Protocol Translation**: Converts HTTP/JSON requests to gRPC calls
- **Service Routing**: Routes requests to appropriate backend services
- **Response Formatting**: Translates gRPC responses back to JSON
- **Health Checks**: Provides service health endpoints

### Project Structure

```
omnipos-gateway/
├── cmd/
│   └── http/              # HTTP server entry point
│       └── main.go
├── config/                # Configuration management
│   ├── config.go          # Config structs and loading
│   └── utils.go           # Environment variable helpers
├── internal/
│   └── client/            # gRPC client wrappers
│       └── user_client.go # User service gRPC client
├── .env                   # Environment variables (not in git)
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Features

- ✅ HTTP-to-gRPC reverse proxy
- ✅ gRPC client management for backend services
- ✅ Structured logging with Zap
- ✅ Environment-based configuration
- ✅ Graceful shutdown
- ✅ Health check endpoints
- ✅ Clean architecture separation

## Prerequisites

- Go 1.24 or higher
- Backend gRPC services running (e.g., omnipos-user-service)

## Setup

### 1. Install Dependencies

```bash
cd /home/fekuna/public_projects/omnipos/omnipos-gateway
go mod tidy
```

### 2. Configure Environment

Update the `.env` file with your gRPC service addresses:

```env
# HTTP Server Configuration
HTTP_PORT=:8081

# gRPC Service Addresses
USER_GRPC_ADDR=localhost:8080
# PRODUCT_GRPC_ADDR=localhost:8082
# ORDER_GRPC_ADDR=localhost:8083
```

## Development

### Running the Gateway

**Important**: Start your backend gRPC services first before running the gateway.

```bash
# Start user service first (in another terminal)
cd /home/fekuna/public_projects/omnipos/omnipos-user-service
go run cmd/grpc/main.go

# Then start the gateway
cd /home/fekuna/public_projects/omnipos/omnipos-gateway
make run
```

The gateway will start on port `:8081` by default.

### Building

```bash
make build
# Binary will be in ./bin/gateway
```

### Testing

```bash
make test
```

## API Endpoints

### Health Check

```bash
curl http://localhost:8081/health
```

Response:

```json
{
  "status": "healthy"
}
```

### Merchant Login (User Service Proxy)

```bash
curl -X POST http://localhost:8081/api/v1/merchant/login \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "1234567890",
    "pin": "1234"
  }'
```

Response:

```json
{
  "accessToken": "...",
  "refreshToken": "..."
}
```

## Adding New Service Integrations

To add a new backend gRPC service:

### 1. Add Service Address to Config

**config/config.go:**

```go
type GRPCServicesConfig struct {
    UserServiceAddr    string
    ProductServiceAddr string  // Add new service
}

// In Load():
GRPCServices: GRPCServicesConfig{
    UserServiceAddr:    getEnvRequired("USER_GRPC_ADDR"),
    ProductServiceAddr: getEnvRequired("PRODUCT_GRPC_ADDR"),  // Add new service
},
```

**.env:**

```env
PRODUCT_GRPC_ADDR=localhost:8082
```

### 2. Create gRPC Client Wrapper

**internal/client/product_client.go:**

```go
package client

import (
    productv1 "github.com/fekuna/omnipos-proto/proto/product/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

type ProductServiceClient struct {
    conn   *grpc.ClientConn
    client productv1.ProductServiceClient
}

func NewProductServiceClient(addr string) (*ProductServiceClient, error) {
    conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, err
    }

    return &ProductServiceClient{
        conn:   conn,
        client: productv1.NewProductServiceClient(conn),
    }, nil
}

func (c *ProductServiceClient) Close() error {
    return c.conn.Close()
}

func (c *ProductServiceClient) GetClient() productv1.ProductServiceClient {
    return c.client
}
```

### 3. Initialize Client and Add Routes in main.go

```go
// Initialize product service client
productClient, err := client.NewProductServiceClient(cfg.GRPCServices.ProductServiceAddr)
if err != nil {
    log.Fatal("failed to connect to product service", zap.Error(err))
}
defer productClient.Close()

// Add product routes
mux.HandleFunc("/api/v1/products", func(w http.ResponseWriter, r *http.Request) {
    // Your HTTP-to-gRPC proxy logic here
})
```

## Environment Variables

| Variable         | Description                 | Default         | Required |
| ---------------- | --------------------------- | --------------- | -------- |
| `APP_NAME`       | Application name            | omnipos-gateway | No       |
| `APP_ENV`        | Environment (dev/prod)      | local           | No       |
| `PRIVATE_KEY`    | API key                     | -               | Yes      |
| `HTTP_PORT`      | HTTP server port            | :8081           | No       |
| `USER_GRPC_ADDR` | User service gRPC address   | -               | Yes      |
| `LOG_LEVEL`      | Log level                   | info            | No       |
| `LOG_ENCODING`   | Log encoding (json/console) | json            | No       |

## Architecture Notes

### Why Manual Proxy Instead of grpc-gateway Auto-Generation?

The current implementation uses manual HTTP-to-gRPC proxying instead of grpc-gateway's auto-generated code because:

1. **Proto files need grpc-gateway annotations**: The backend service proto files don't have grpc-gateway annotations yet
2. **Simpler setup**: Manual proxying is straightforward and doesn't require proto modifications
3. **Full control**: Easier to add custom middleware, validation, and error handling

**Future Migration**: When proto files are updated with grpc-gateway annotations and `.gw.pb.go` files are generated, the gateway can be updated to use auto-generated handlers for zero-maintenance API exposure.

### No Database Connection

The gateway does **not** connect to any database. All data operations are handled by the backend gRPC services. The gateway's sole purpose is protocol translation (HTTP ↔ gRPC).

## Troubleshooting

### Gateway can't connect to backend service

**Error**: `failed to connect to user service`

**Solution**: Ensure the backend gRPC service is running:

```bash
cd /home/fekuna/public_projects/omnipos/omnipos-user-service
go run cmd/grpc/main.go
```

Check that the `USER_GRPC_ADDR` in `.env` matches the service address.

### Port already in use

**Error**: `bind: address already in use`

**Solution**: Change `HTTP_PORT` in `.env` to a different port, or kill the process using port 8081:

```bash
lsof -ti:8081 | xargs kill
```

## License

Private - OmniPOS Project
