package tailscale

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/logical"
	"golang.org/x/oauth2/clientcredentials"
	"tailscale.com/client/tailscale"
)

const DEFAULT_BASE_URL = "https://api.tailscale.com"

func (b *backend) client(ctx context.Context, s logical.Storage) (*tailscale.Client, error) {
	tailscale.I_Acknowledge_This_API_Is_Unstable = true

	conf, err := b.readConfigToken(ctx, s)
	if err != nil {
		return nil, err
	}

	var tsClient *tailscale.Client
	if conf.ClientID != "" && conf.ClientSecret != "" {
		credentials := clientcredentials.Config{
			ClientID:     conf.ClientID,
			ClientSecret: conf.ClientSecret,
			TokenURL:     conf.BaseURL + "/api/v2/oauth/token",
			Scopes:       []string{"all"},
		}
		tsClient = tailscale.NewClient("-", nil)
		ctx := context.Background()
		tsClient.HTTPClient = credentials.Client(ctx)
		tsClient.BaseURL = conf.BaseURL
	} else if conf.Token != "" {
		tsClient = tailscale.NewClient(conf.Tailnet, tailscale.APIKey(conf.Token))
	}

	if tsClient == nil {
		return nil, fmt.Errorf("could not create ts client. neither 'client_id' and 'client_secret' or 'token' are specified")
	}

	return tsClient, nil
}
