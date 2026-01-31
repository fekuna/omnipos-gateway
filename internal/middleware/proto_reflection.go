package middleware

import (
	"fmt"
	"sync"

	authv1 "github.com/fekuna/omnipos-proto/proto/auth/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var (
	publicEndpointsCache     map[string]bool
	publicEndpointsCacheLock sync.RWMutex
)

// DiscoverPublicEndpoints scans all registered gRPC services and builds a map of public endpoints
// by reading the custom (auth.v1.public_endpoint) option from proto method definitions.
func DiscoverPublicEndpoints() (map[string]bool, error) {
	publicEndpointsCacheLock.Lock()
	defer publicEndpointsCacheLock.Unlock()

	// Return cached result if already computed
	if publicEndpointsCache != nil {
		return publicEndpointsCache, nil
	}

	publicEndpoints := make(map[string]bool)

	// Iterate through all registered services
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			service := services.Get(i)
			methods := service.Methods()

			for j := 0; j < methods.Len(); j++ {
				method := methods.Get(j)

				// Build the full method name in gRPC format: /package.Service/Method
				fullMethodName := fmt.Sprintf("/%s/%s", service.FullName(), method.Name())

				// Check if the method has the public_endpoint option set to true
				if isPublic := isPublicEndpoint(method); isPublic {
					publicEndpoints[fullMethodName] = true
				}
			}
		}
		return true
	})

	// Cache the result
	publicEndpointsCache = publicEndpoints
	return publicEndpoints, nil
}

// isPublicEndpoint checks if a method has the (auth.v1.public_endpoint) option set to true
func isPublicEndpoint(method protoreflect.MethodDescriptor) bool {
	opts := method.Options()
	if opts == nil {
		return false
	}

	// Get the extension value for E_PublicEndpoint
	if proto.HasExtension(opts, authv1.E_PublicEndpoint) {
		publicEndpoint := proto.GetExtension(opts, authv1.E_PublicEndpoint)
		if val, ok := publicEndpoint.(bool); ok {
			return val
		}
	}

	return false
}
