package tailscale

import (
	"context"

	"github.com/davidsbond/tailscale-client-go/tailscale"
	"github.com/hashicorp/vault/sdk/logical"
)

func (b *backend) client(ctx context.Context, s logical.Storage) (*tailscale.Client, error) {
	conf, err := b.readConfigToken(ctx, s)
	if err != nil {
		return nil, err
	}

	return tailscale.NewClient(conf.Token, conf.Tailnet)
}
