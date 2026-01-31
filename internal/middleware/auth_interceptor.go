package middleware

import (
	"context"
	"strings"

	"github.com/fekuna/omnipos-pkg/logger"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor handles authentication for gRPC requests
type AuthInterceptor struct {
	jwtHelper       *JWTHelper
	logger          logger.ZapLogger
	publicEndpoints map[string]bool
}

// NewAuthInterceptor creates a new authentication interceptor
// publicEndpoints: map of method names (e.g., "/user.v1.MerchantService/LoginMerchant") that don't require authentication
func NewAuthInterceptor(jwtHelper *JWTHelper, log logger.ZapLogger, publicEndpoints map[string]bool) *AuthInterceptor {
	return &AuthInterceptor{
		jwtHelper:       jwtHelper,
		logger:          log,
		publicEndpoints: publicEndpoints,
	}
}

// Unary returns a unary server interceptor for authentication
func (a *AuthInterceptor) Unary() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Skip authentication for public endpoints
		if a.publicEndpoints[method] {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		// Try to get metadata from incoming context first (from grpc-gateway)
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok || len(md) == 0 {
			// If not in incoming context, try outgoing context
			md, ok = metadata.FromOutgoingContext(ctx)
		}

		if !ok || len(md) == 0 {
			a.logger.Warn("no metadata found in request context")
			return status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Debug: Log all metadata keys
		a.logger.Info("🔍 INSPECTING METADATA KEYS")
		for key, values := range md {
			a.logger.Info("metadata key found", zap.String("key", key), zap.Strings("values", values))
		}

		// Get authorization header
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			// Try grpcgateway-authorization (grpc-gateway specific)
			authHeaders = md.Get("grpcgateway-authorization")
		}

		if len(authHeaders) == 0 {
			a.logger.Warn("no authorization header found in metadata")
			return status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Extract token from "Bearer <token>"
		authHeader := authHeaders[0]
		if !strings.HasPrefix(authHeader, "Bearer ") {
			a.logger.Warn("invalid authorization header format", zap.String("header", authHeader))
			return status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate token and extract merchant ID
		merchantID, err := a.jwtHelper.ExtractMerchantID(token)
		if err != nil {
			a.logger.Warn("token validation failed", zap.Error(err))
			if err == ErrExpiredToken {
				return status.Error(codes.Unauthenticated, "token has expired")
			}
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		a.logger.Debug("authentication successful", zap.String("merchant_id", merchantID))

		// Add merchant ID to outgoing metadata for internal service
		outgoingMD := metadata.Pairs(
			"x-merchant-id", merchantID,
		)

		// Merge with existing outgoing metadata if any
		if existingMD, ok := metadata.FromOutgoingContext(ctx); ok {
			outgoingMD = metadata.Join(existingMD, outgoingMD)
		}

		ctx = metadata.NewOutgoingContext(ctx, outgoingMD)

		// Call the actual gRPC method
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// HTTPHeaderMatcher allows custom HTTP headers to be passed to gRPC metadata
func HTTPHeaderMatcher(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "authorization":
		return "authorization", true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}
