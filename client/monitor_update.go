package client

import (
	"context"
	"encoding/json"

	hreq "github.com/imroc/req/v3"
)

func UpdateMonitor(ctx context.Context, c *hreq.Client, request MonitorRequest, id string) (*MonitorRequest, error) {

	response := c.Put("monitor/" + id).SetBody(&request).Do()

	if response.Err != nil {
		return nil, response.Err
	}
	var monitor MonitorRequest
	if err := json.NewDecoder(response.Body).Decode(&monitor); err != nil {
		return nil, err
	}
	return &monitor, nil
}
