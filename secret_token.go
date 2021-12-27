package tailscale

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	SecretTokenType = "token"
)

func secretToken(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: SecretTokenType,
		Fields: map[string]*framework.FieldSchema{
			"token": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "tailscale API token",
			},
			"id": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "ID of the API Token",
			},
		},

		Revoke: b.secretTokenRevoke,
	}
}

func (b *backend) secretTokenRevoke(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	c, err := b.client(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("error getting tailscale client")
	}

	id, ok := req.Secret.InternalData["id"]
	if !ok {
		return nil, fmt.Errorf("id is missing on the lease")
	}

	b.Logger().Info(fmt.Sprintf("Revoking tailscale token (%s)...", id))
	err = c.deleteAPIKey(ctx, id.(string))
	if err != nil {
		return logical.ErrorResponse(fmt.Sprintf("failed to revoke cloudflare token (%s). err: %s", id, err)), nil
	}

	return nil, nil
}
