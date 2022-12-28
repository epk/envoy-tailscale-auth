package tailscale

import (
	"context"

	ts "tailscale.com/client/tailscale"
	"tailscale.com/client/tailscale/apitype"
)

// LocalClient exposes subset of tailscale.LocalClient methods that are used by the authservice
type LocalClient interface {
	WhoIs(ctx context.Context, addr string) (*apitype.WhoIsResponse, error)
}

func NewLocalClient() LocalClient {
	return &impl{&ts.LocalClient{}}
}

// impl is a wrapper around tailscale.LocalClient that implements the LocalClient interface
type impl struct {
	*ts.LocalClient
}
