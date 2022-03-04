package tailscale

import (
	"context"
	"fmt"
	"time"

	"github.com/davidsbond/tailscale-client-go/tailscale"
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
			"expires": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Date the token expires",
			},
		},

		Revoke: b.secretTokenRevoke,
		Renew:  b.secretTokenRenew,
	}
}

func (b *backend) secretTokenRenew(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	lease, err := b.LeaseConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if lease == nil {
		lease = &configLease{}
	}

	expires, ok := req.Secret.InternalData["expires"]
	if !ok {
		return nil, fmt.Errorf("expiration time is missing on the lease")
	}

	expirationDate, err := time.Parse(time.RFC3339, expires.(string))
	if err != nil {
		return logical.ErrorResponse("failed to parse expiration date. err: %s", err), nil
	}

	resp := &logical.Response{Secret: req.Secret}
	resp.Secret.TTL = lease.TTL
	resp.Secret.MaxTTL = expirationDate.Sub(time.Now())
	return resp, nil
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
	err = c.DeleteKey(ctx, id.(string))
	if err != nil {
		// if we get 404 on deleting key, it is already deleted and we can ignore it
		if tailscale.IsNotFound(err) {
			return nil, nil
		}

		return logical.ErrorResponse(fmt.Sprintf("failed to revoke tailscale token (%s). err: %s", id, err)), nil
	}

	return nil, nil
}
