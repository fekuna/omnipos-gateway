package runtime

import (
	"encoding/json"
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
)

// CustomMarshaler is a custom marshaler that wraps the response in a standard format.
// It embeds runtime.JSONPb to leverage the default Protobuf JSON marshaling.
type CustomMarshaler struct {
	runtime.JSONPb
}

// NewCustomMarshaler creates a new CustomMarshaler with default options.
func NewCustomMarshaler() *CustomMarshaler {
	return &CustomMarshaler{
		JSONPb: runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: true, // Emit default values
				UseProtoNames:   true, // Use snake_case as in proto definitions
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true, // Ignore unknown fields
			},
		},
	}
}

// Marshal wraps the default JSONPb marshaling with a standard response envelope.
func (c *CustomMarshaler) Marshal(v interface{}) ([]byte, error) {
	// Check if this is an error response from grpc-gateway
	// The default error handler passes a map[string]interface{} with specific fields
	if errMap, ok := v.(map[string]interface{}); ok {
		// Verify if it looks like a grpc-gateway error (has "code" and "message")
		if code, ok := errMap["code"]; ok {
			if msg, ok := errMap["message"]; ok {
				// It's likely an error. Code is usually float64 in JSON map or int
				statusCode := 200 // Default

				// Safely cast code
				switch val := code.(type) {
				case int:
					// If it's a gRPC status code, we need to map it to HTTP
					// But usually passing through Marshaler comes from runtime.HTTPError
					// which might pass the gRPC code (usually int 0-16).
					// Wait, grpc-gateway's DefaultHTTPErrorHandler constructs a map where "code" is the gRPC code.
					// We need to map gRPC code to HTTP status.
					statusCode = runtime.HTTPStatusFromCode(codes.Code(val))
				case float64:
					statusCode = runtime.HTTPStatusFromCode(codes.Code(int(val)))
				case int32:
					statusCode = runtime.HTTPStatusFromCode(codes.Code(val))
				}

				// The user wants:
				// "status": <HTTP CODE>
				// "message": <ERROR MSG>
				// "data": null

				return json.Marshal(map[string]interface{}{
					"status":  statusCode,
					"message": msg,
					"data":    nil,
				})
			}
		}
	}

	// Handle *status.Status directly if passed
	if s, ok := v.(*status.Status); ok {
		statusCode := runtime.HTTPStatusFromCode(codes.Code(s.Code))
		return json.Marshal(map[string]interface{}{
			"status":  statusCode,
			"message": s.Message,
			"data":    nil,
		})
	}

	// First, marshal the original value using the standard JSONPb marshaler.
	// This ensures we respect all Protobuf JSON mapping rules (snake_case, enums as strings, etc.)
	data, err := c.JSONPb.Marshal(v)
	if err != nil {
		return nil, err
	}

	// Define the standard response structure
	type StandardResponse struct {
		Status  int             `json:"status"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}

	// Create the wrapped response
	response := StandardResponse{
		Status:  200,       // Default status for successful successful gRPC calls handled here
		Message: "success", // Default message
		Data:    data,
	}

	// Marshal the wrapped response
	return json.Marshal(response)
}

// CustomEncoder wraps the writer to encode responses using CustomMarshaler.
type CustomEncoder struct {
	w io.Writer
	m *CustomMarshaler
}

// Encode encodes the val to the writer using CustomMarshaler.
func (e *CustomEncoder) Encode(v interface{}) error {
	data, err := e.m.Marshal(v)
	if err != nil {
		return err
	}
	_, err = e.w.Write(data)
	return err
}

// NewEncoder returns a new encoder.
func (c *CustomMarshaler) NewEncoder(w io.Writer) runtime.Encoder {
	return &CustomEncoder{w: w, m: c}
}

// NewDecoder returns a new decoder.
// We just use the embedded JSONPb's NewDecoder for requests.
func (c *CustomMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return c.JSONPb.NewDecoder(r)
}

// ContentType returns the content type for this marshaler.
func (c *CustomMarshaler) ContentType(v interface{}) string {
	return "application/json"
}
