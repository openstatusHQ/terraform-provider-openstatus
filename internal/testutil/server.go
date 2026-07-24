package testutil

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	monitorv1connect "buf.build/gen/go/openstatus/api/connectrpc/go/openstatus/monitor/v1/monitorv1connect"
	notificationv1connect "buf.build/gen/go/openstatus/api/connectrpc/go/openstatus/notification/v1/notificationv1connect"
	privatelocationv1connect "buf.build/gen/go/openstatus/api/connectrpc/go/openstatus/private_location/v1/private_locationv1connect"
	statuspagev1connect "buf.build/gen/go/openstatus/api/connectrpc/go/openstatus/status_page/v1/status_pagev1connect"

	"connectrpc.com/connect"

	"terraform-provider-openstatus/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// APIKey is the token the fake server requires and the test provider sends.
const APIKey = "test-api-key"

func requireAPIKey() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Header().Get("x-openstatus-key") != APIKey {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing or invalid x-openstatus-key"))
			}
			return next(ctx, req)
		}
	}
}

// NewServer starts a fake OpenStatus API and returns it with its backing store.
func NewServer(t *testing.T) (*httptest.Server, *Fake) {
	t.Helper()

	fake := NewFake()
	opts := connect.WithInterceptors(requireAPIKey())

	mux := http.NewServeMux()
	mux.Handle(monitorv1connect.NewMonitorServiceHandler(&monitorService{f: fake}, opts))
	mux.Handle(notificationv1connect.NewNotificationServiceHandler(&notificationService{f: fake}, opts))
	mux.Handle(statuspagev1connect.NewStatusPageServiceHandler(&statusPageService{f: fake}, opts))
	mux.Handle(privatelocationv1connect.NewPrivateLocationServiceHandler(&privateLocationService{f: fake}, opts))

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, fake
}

// ProviderConfig returns an HCL provider block pointed at the fake server.
func ProviderConfig(server *httptest.Server) string {
	return `
provider "openstatus" {
  api_token = "` + APIKey + `"
  base_url  = "` + server.URL + `"
}
`
}

// ProviderFactories wires the real provider for use in acceptance tests.
func ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"openstatus": providerserver.NewProtocol6WithError(provider.New()()),
	}
}
