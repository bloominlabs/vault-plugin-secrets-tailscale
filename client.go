package tailscale

import (
	"cmp"
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/vault/sdk/logical"
	"golang.org/x/oauth2/clientcredentials"
	"tailscale.com/client/tailscale"
)

func (b *backend) client(ctx context.Context, s logical.Storage) (*tailscale.Client, error) {
	tailscale.I_Acknowledge_This_API_Is_Unstable = true

	conf, err := b.readConfigToken(ctx, s)
	if err != nil {
		return nil, err
	}

	baseURL := cmp.Or(os.Getenv("TS_BASE_URL"), "https://api.tailscale.com")
	var tsClient *tailscale.Client
	if conf.ClientID != "" && conf.ClientSecret != "" {
		credentials := clientcredentials.Config{
			ClientID:     conf.ClientID,
			ClientSecret: conf.ClientSecret,
			TokenURL:     baseURL + "/api/v2/oauth/token",
			Scopes:       []string{"devices"},
		}
		tsClient = tailscale.NewClient("-", nil)
		ctx := context.Background()
		tsClient.HTTPClient = credentials.Client(ctx)
		tsClient.BaseURL = baseURL
	} else if conf.Token != "" {
		tsClient = tailscale.NewClient(conf.Tailnet, tailscale.APIKey(conf.Token))
	}

	if tsClient == nil {
		return nil, fmt.Errorf("could not create ts client. neither 'client_id' and 'client_secret' or 'token' are specified")
	}

	return tsClient, nil
}
