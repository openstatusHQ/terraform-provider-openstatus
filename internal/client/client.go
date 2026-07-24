package client

import (
	"context"
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

// IsNotFound reports whether err is the API's not-found response.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return connect.CodeOf(err) == connect.CodeNotFound
}

type Client struct {
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
		Monitor:         monitorv1connect.NewMonitorServiceClient(httpClient, baseURL, opts...),
		Notification:    notificationv1connect.NewNotificationServiceClient(httpClient, baseURL, opts...),
		StatusPage:      statuspagev1connect.NewStatusPageServiceClient(httpClient, baseURL, opts...),
		PrivateLocation: privatelocationv1connect.NewPrivateLocationServiceClient(httpClient, baseURL, opts...),
	}
}
