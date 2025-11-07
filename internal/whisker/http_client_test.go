package whisker

import (
	"testing"
)

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient()
	
	if client == nil {
		t.Fatal("Expected HTTPClient to be created, got nil")
	}
	
	if client.baseURL != defaultWhiskerURL {
		t.Errorf("Expected baseURL to be %s, got %s", defaultWhiskerURL, client.baseURL)
	}
	
	if client.endpoint != defaultWhiskerEndpoint {
		t.Errorf("Expected endpoint to be %s, got %s", defaultWhiskerEndpoint, client.endpoint)
	}
	
	if client.client == nil {
		t.Error("Expected HTTP client to be initialized, got nil")
	}
}
