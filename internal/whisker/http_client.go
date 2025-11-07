package whisker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

const (
	defaultWhiskerURL      = "http://127.0.0.1:8081"
	defaultWhiskerEndpoint = "/whisker-backend/flows"
)

// HTTPClient handles HTTP communication with the Whisker backend service
type HTTPClient struct {
	baseURL  string
	endpoint string
	client   *http.Client
}

// NewHTTPClient creates a new HTTP client for Whisker service
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		baseURL:  defaultWhiskerURL,
		endpoint: defaultWhiskerEndpoint,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetFlowLogs retrieves flow logs from Whisker service
func (h *HTTPClient) GetFlowLogs(ctx context.Context) ([]types.FlowLog, error) {
	url := h.baseURL + h.endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to Calico Whisker. Please ensure port-forward is running: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("whisker service returned status %d", resp.StatusCode)
	}

	var response types.FlowLogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Items, nil
}
