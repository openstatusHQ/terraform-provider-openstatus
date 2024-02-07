package client

import (
	"context"

	hreq "github.com/imroc/req/v3"
)

func DeleteMonitor(ctx context.Context, c *hreq.Client, id string) *error {

	req := c.Delete("monitor/" + id).Do()

	if req.Err != nil {
		return &req.Err

	}
	return nil

}
