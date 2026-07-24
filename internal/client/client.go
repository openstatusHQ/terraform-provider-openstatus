package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	monitorv1connect "buf.build/gen/go/openstatus/api/connectrpc/go/openstatus/monitor/v1/monitorv1connect"
	notificationv1connect "buf.build/gen/go/openstatus/api/connectrpc/go/openstatus/notification/v1/notificationv1connect"
	privatelocationv1connect "buf.build/gen/go/openstatus/api/connectrpc/go/openstatus/private_location/v1/private_locationv1connect"
	statuspagev1connect "buf.build/gen/go/openstatus/api/connectrpc/go/openstatus/status_page/v1/status_pagev1connect"

	"connectrpc.com/connect"
)

const DefaultBaseURL = "https://api.openstatus.dev/rpc"

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

// IsNotFound reports whether err is the API's not-found response.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == "not_found"
	}
	return connect.CodeOf(err) == connect.CodeNotFound
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client

	Monitor         monitorv1connect.MonitorServiceClient
	Notification    notificationv1connect.NotificationServiceClient
	StatusPage      statuspagev1connect.StatusPageServiceClient
	PrivateLocation privatelocationv1connect.PrivateLocationServiceClient
}

func authInterceptor(apiKey string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("x-openstatus-key", apiKey)
			return next(ctx, req)
		}
	}
}

func New(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	httpClient := &http.Client{}
	opts := []connect.ClientOption{
		connect.WithProtoJSON(),
		connect.WithInterceptors(authInterceptor(apiKey)),
	}

	return &Client{
		baseURL:         baseURL,
		apiKey:          apiKey,
		httpClient:      httpClient,
		Monitor:         monitorv1connect.NewMonitorServiceClient(httpClient, baseURL, opts...),
		Notification:    notificationv1connect.NewNotificationServiceClient(httpClient, baseURL, opts...),
		StatusPage:      statuspagev1connect.NewStatusPageServiceClient(httpClient, baseURL, opts...),
		PrivateLocation: privatelocationv1connect.NewPrivateLocationServiceClient(httpClient, baseURL, opts...),
	}
}

// Do is the pre-SDK JSON transport, retained until every package is migrated.
func (c *Client) Do(ctx context.Context, path string, reqBody, respBody any) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
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
