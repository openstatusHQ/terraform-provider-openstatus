package client

import (
	"context"
	"encoding/json"

	hreq "github.com/imroc/req/v3"
)

type MonitorRequest struct {
	Active      bool   `json:"active"`
	Body        string `json:"body"`
	Description string `json:"description"`
	Headers     []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"headers,omitempty"`
	Id            int64           `json:"id"`
	Method        string          `json:"method"`
	Name          string          `json:"name"`
	Periodicity   string          `json:"periodicity"`
	Regions       []string        `json:"regions"`
	Url           string          `json:"url"`
	Public        bool            `json:"public"`
	Assertions    json.RawMessage `json:"assertions,omitempty"`
	Timeout       int             `json:"timeout"`
	DegradedAfter int             `json:"degradedAfter"`
	Type          string          `json:"jobType,omitempty"`
}

func CreateMonitor(ctx context.Context, c *hreq.Client, request MonitorRequest) (*MonitorRequest, error) {

	response := c.Post("monitor").SetBody(&request).Do()

	if response.Err != nil {
		return nil, response.Err

	}

	var monitor MonitorRequest
	if err := json.NewDecoder(response.Body).Decode(&monitor); err != nil {
		return nil, err
	}

	return &monitor, nil
}
