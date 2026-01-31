module github.com/fekuna/omnipos-gateway

go 1.24.0

toolchain go1.24.12

replace github.com/fekuna/omnipos-pkg => ../omnipos-pkg

replace github.com/fekuna/omnipos-proto => ../omnipos-proto

require (
	github.com/fekuna/omnipos-pkg v0.0.0-00010101000000-000000000000
	github.com/fekuna/omnipos-proto v0.0.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.7
	github.com/joho/godotenv v1.5.1
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/envoyproxy/protoc-gen-validate v1.3.0 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260128011058-8636f8732409 // indirect
)
