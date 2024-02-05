package client

import (
	"context"

	hreq "github.com/imroc/req/v3"
)

type CreateMonitorRequest struct {
	Active      bool   `json:"active"`
	Body        string `json:"body"`
	Description string `json:"description"`
	Headers     []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"headers,omitempty"`
	Id          int64    `json:"id"`
	Method      string   `json:"method"`
	Name        string   `json:"name"`
	Periodicity string   `json:"periodicity"`
	Regions     []string `json:"regions"`
	Url         string   `json:"url"`
}

func CreateMonitor(ctx context.Context, c *hreq.Client, request CreateMonitorRequest) {
	// TODO - Implement this method

	err := c.Post("monitor").SetBody(&request).Do()

	if err != nil {
		panic(err)
	}
	return

}
