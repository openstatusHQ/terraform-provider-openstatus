package client

import (
	"context"
	"encoding/json"

	hreq "github.com/imroc/req/v3"
)

func GetMonitor(ctx context.Context, c *hreq.Client, id string) (*MonitorRequest, error) {

	request := c.Get("monitor/" + id).Do()

	if request.Err != nil {
		return nil, request.Err
	}
	var monitor MonitorRequest
	if err := json.NewDecoder(request.Body).Decode(&monitor); err != nil {
		return nil, err
	}

	return &monitor, nil
}
