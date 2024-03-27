package tailscale

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"golang.org/x/oauth2/clientcredentials"
)

func pathCredsCreate(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "creds/" + framework.GenericNameRegex("role"),
		Fields: map[string]*framework.FieldSchema{
			"role": {
				Type:        framework.TypeString,
				Description: "Create a tailscale token from a Vault role",
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

	// if scopes are specified, try to create an api access token instead using the oauth client
	if len(roleEntry.Scopes) > 0 {
		conf, err := b.readConfigToken(ctx, req.Storage)
		if err != nil {
			return nil, err
		}

		if conf.ClientID == "" || conf.ClientSecret == "" {
			return logical.ErrorResponse(
				fmt.Sprintf("%s specifies a scope, but the backend was not configured with oauth credentials. Please reconfigure the backend to use scopes", role),
			), nil
		}

		credentials := clientcredentials.Config{
			ClientID:     conf.ClientID,
			ClientSecret: conf.ClientSecret,
			TokenURL:     conf.BaseURL + "/api/v2/oauth/token",
			Scopes:       roleEntry.Scopes,
		}

		token, err := credentials.Token(ctx)
		if err != nil {
			return logical.ErrorResponse(
				fmt.Sprintf("failed to generate token: %s", err),
			), nil
		}

		resp := b.Secret(SecretTokenType).Response(map[string]interface{}{
			"id":      "",
			"token":   token.AccessToken,
			"expires": token.Expiry,
		}, map[string]interface{}{
			"id":      "",
			"token":   token.AccessToken,
			"expires": token.Expiry,
		})
		TTL := time.Until(token.Expiry)
		resp.Secret.TTL = TTL
		resp.Secret.MaxTTL = TTL
		resp.Secret.Renewable = false
		return resp, nil

	}

	// Get the http client
	c, err := b.client(ctx, req.Storage)

	if err != nil {
		return nil, err
	}

	secret, metadata, err := c.CreateKey(ctx, roleEntry.Capabilities)
	if err != nil {
		return logical.ErrorResponse("failed to create token. err: %s", err), nil
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
	// Use the helper to create the secret
	resp := b.Secret(SecretTokenType).Response(map[string]interface{}{
		"id":           metadata.ID,
		"token":        secret,
		"expires":      metadata.Expires,
		"capabilities": roleEntry.Capabilities,
	}, map[string]interface{}{
		"id":      metadata.ID,
		"token":   secret,
		"expires": metadata.Expires,
	})
	resp.Secret.TTL = ttl
	resp.Secret.MaxTTL = time.Until(metadata.Expires)
	return resp, nil
}
