package middleware

import (
	"context"
	"net/http"

	"google.golang.org/grpc/metadata"
)

// MetadataAnnotator is a custom annotator for grpc-gateway to pass HTTP headers to gRPC metadata
func MetadataAnnotator(ctx context.Context, req *http.Request) metadata.MD {
	md := make(metadata.MD)

	// Language
	if lang := req.Header.Get("Accept-Language"); lang != "" {
		md.Set("x-lang", lang)
	}

	// Timezone
	if tz := req.Header.Get("X-Timezone"); tz != "" {
		md.Set("x-timezone", tz)
	}

	return md
}
