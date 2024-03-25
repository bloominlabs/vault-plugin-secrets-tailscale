package tailscale

import (
	"context"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/tailscale/tailscale-client-go/tailscale"
)

func (b *backend) client(ctx context.Context, s logical.Storage) (*tailscale.Client, error) {
	conf, err := b.readConfigToken(ctx, s)
	if err != nil {
		return nil, err
	}

	options := []tailscale.ClientOption{}
	if conf.ClientID != "" && conf.ClientSecret != "" {
		options = []tailscale.ClientOption{tailscale.WithOAuthClientCredentials(conf.ClientID, conf.ClientSecret, []string{"devices"})}
	}

	return tailscale.NewClient(conf.Token, conf.Tailnet, options...)
}
