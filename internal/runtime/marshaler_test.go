package runtime

import (
	"encoding/json"
	"testing"
)

func TestCustomMarshaler_Marshal(t *testing.T) {
	cm := NewCustomMarshaler()

	// Test data (mimics a proto struct or simple map)
	input := map[string]string{
		"foo": "bar",
	}

	data, err := cm.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Define expected structure
	type StandardResponse struct {
		Status  int             `json:"status"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}

	var resp StandardResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if resp.Status != 200 {
		t.Errorf("Expected status 200, got %d", resp.Status)
	}
	if resp.Message != "success" {
		t.Errorf("Expected message 'success', got '%s'", resp.Message)
	}

	// Check data content
	var innerData map[string]string
	if err := json.Unmarshal(resp.Data, &innerData); err != nil {
		t.Fatalf("Failed to unmarshal inner data: %v", err)
	}

	if innerData["foo"] != "bar" {
		t.Errorf("Expected data.foo to be 'bar', got '%s'", innerData["foo"])
	}
}

func TestCustomMarshaler_Marshal_Error(t *testing.T) {
	cm := NewCustomMarshaler()

	// Simulate a grpc-gateway error map (Code 5 = NOT_FOUND -> HTTP 404)
	input := map[string]interface{}{
		"code":    5,
		"message": "merchant not found",
	}

	data, err := cm.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Define expected structure
	type StandardResponse struct {
		Status  int             `json:"status"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}

	var resp StandardResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Expect 404
	if resp.Status != 404 {
		t.Errorf("Expected status 404, got %d", resp.Status)
	}
	if resp.Message != "merchant not found" {
		t.Errorf("Expected message 'merchant not found', got '%s'", resp.Message)
	}
	if string(resp.Data) != "null" {
		t.Errorf("Expected data to be null, got %s", resp.Data)
	}
}
