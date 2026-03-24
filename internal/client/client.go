package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const DefaultBaseURL = "https://api.openstatus.dev/rpc"
const DefaultRESTBaseURL = "https://api.openstatus.dev/v1"

type ProviderConfig struct {
	Client *Client
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("openstatus API error (%s): %s", e.Code, e.Message)
}

type Client struct {
	baseURL     string
	restBaseURL string
	apiKey      string
	httpClient  *http.Client
}

func New(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	restBaseURL := strings.TrimSuffix(baseURL, "/rpc") + "/v1"
	return &Client{
		baseURL:     baseURL,
		restBaseURL: restBaseURL,
		apiKey:      apiKey,
		httpClient:  &http.Client{},
	}
}

func (c *Client) Do(ctx context.Context, path string, reqBody, respBody any) error {
	return c.doRequest(ctx, http.MethodPost, c.baseURL+path, reqBody, respBody)
}

// DoREST sends a request to the REST v1 API with the given HTTP method and path.
func (c *Client) DoREST(ctx context.Context, method, path string, reqBody, respBody any) error {
	return c.doRequest(ctx, method, c.restBaseURL+path, reqBody, respBody)
}

func (c *Client) doRequest(ctx context.Context, method, url string, reqBody, respBody any) error {
	var bodyReader io.Reader
	if reqBody != nil {
		body, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-openstatus-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr APIError
		if json.Unmarshal(respBytes, &apiErr) == nil && apiErr.Code != "" {
			return &apiErr
		}
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBytes))
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}
