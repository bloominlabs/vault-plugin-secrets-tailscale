package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func pathCredsCreate(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "creds/" + framework.GenericNameRegex("role"),
		Fields: map[string]*framework.FieldSchema{
			"role": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Create a cloudflare token from a Vault role",
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation: b.pathCredsRead,
		},
	}
}

func (b *backend) pathCredsRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	role := d.Get("role").(string)

	roleEntry, err := b.roleRead(ctx, req.Storage, role)
	if err != nil {
		return logical.ErrorResponse(fmt.Sprintf("err while getting role configuration for '%s'. err: %s", role, err)), nil
	}
	if roleEntry == nil {
		return logical.ErrorResponse(fmt.Sprintf("could not find entry for role '%s', did you configure it?", role)), nil
	}

	var capabilities Capabilities
	if roleEntry.Capabilities != "" {
		err = json.Unmarshal([]byte(roleEntry.Capabilities), &capabilities)
		if err != nil {
			return logical.ErrorResponse(
				"failed to marshal '%s' into a tailscale capabilities. ensure your configuration is correct",
				roleEntry.Capabilities,
			), nil
		}
	}

	// Get the http client
	c, err := b.client(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	lease, err := b.LeaseConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if lease == nil {
		lease = &configLease{}
	}

	ttl, _, err := framework.CalculateTTL(b.System(), 0, lease.TTL, 0, lease.MaxTTL, 0, time.Time{})
	if err != nil {
		return logical.ErrorResponse("failed to calculate ttl. err: %w", err), nil
	}

	createdToken, err := c.createAPIKey(ctx, capabilities)
	if err != nil {
		return logical.ErrorResponse("failed to create token. err: %s", err), nil
	}
	expires, err := time.Parse(time.RFC3339, createdToken.Expires)
	if err != nil {
		return logical.ErrorResponse("failed to parse expiration date. err: %s", err), nil
	}

	// Use the helper to create the secret
	resp := b.Secret(SecretTokenType).Response(map[string]interface{}{
		"id":           createdToken.ID,
		"token":        createdToken.Key,
		"expires":      createdToken.Expires,
		"capabilities": roleEntry.Capabilities,
	}, map[string]interface{}{
		"id":      createdToken.ID,
		"token":   createdToken.Key,
		"expires": createdToken.Expires,
	})
	resp.Secret.TTL = ttl
	resp.Secret.MaxTTL = expires.Sub(time.Now())
	return resp, nil
}
