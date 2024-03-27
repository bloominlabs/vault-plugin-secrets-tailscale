package tailscale

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const configRootKey = "config/root"

func pathConfigToken(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: configRootKey,
		Fields: map[string]*framework.FieldSchema{
			"tailnet": {
				Type:        framework.TypeString,
				Description: "tailnet to make API request on behalf of",
			},
			"token": {
				Type:        framework.TypeString,
				Description: "token to authenticate API requests",
			},
			"client_id": {
				Type:        framework.TypeString,
				Description: "Tailscale OAuth Client ID with the 'devices' scope. Preferred over 'token' if both are specified",
			},
			"client_secret": {
				Type:        framework.TypeString,
				Description: "Tailscale OAuth Client Secret with the 'devices' scope. Preferred over 'token' if both are specified",
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation:   b.pathConfigTokenRead,
			logical.CreateOperation: b.pathConfigTokenWrite,
			logical.UpdateOperation: b.pathConfigTokenWrite,
			logical.DeleteOperation: b.pathConfigTokenDelete,
		},

		ExistenceCheck: b.configTokenExistenceCheck,
	}
}

func (b *backend) configTokenExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	entry, err := b.readConfigToken(ctx, req.Storage)
	if err != nil {
		return false, err
	}

	return entry != nil, nil
}

func (b *backend) readConfigToken(ctx context.Context, storage logical.Storage) (*rootTokenConfig, error) {
	entry, err := storage.Get(ctx, configRootKey)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	conf := &rootTokenConfig{}
	if err := entry.DecodeJSON(conf); err != nil {
		return nil, fmt.Errorf("error reading nomad access configuration: %w", err)
	}

	return conf, nil
}

func (b *backend) pathConfigTokenRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	conf, err := b.readConfigToken(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if conf == nil {
		return logical.ErrorResponse(
			fmt.Sprintf("configuration does not exist. did you configure '%s'?", configRootKey),
		), nil
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"tailnet":       conf.Tailnet,
			"token":         conf.Token,
			"client_id":     conf.ClientID,
			"client_secret": conf.ClientSecret,
		},
	}, nil
}

func (b *backend) pathConfigTokenWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	conf, err := b.readConfigToken(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if conf == nil {
		conf = &rootTokenConfig{}
	}

	tailnet, ok := data.GetOk("tailnet")
	if !ok {
		return logical.ErrorResponse("Missing 'tailnet' in configuration request"), nil
	}
	conf.Tailnet = tailnet.(string)

	clientID, clientIDOK := data.GetOk("client_id")
	clientSecret, clientSecretOK := data.GetOk("client_secret")
	token, tokenOk := data.GetOk("token")

	if (!clientIDOK || !clientSecretOK) && !tokenOk {
		return logical.ErrorResponse("Must have one of 'client_id' and 'client_secret' or 'token'"), nil
	}
	if tokenOk {
		conf.Token = token.(string)
	}
	if clientIDOK {
		conf.ClientID = clientID.(string)
	}
	if clientSecretOK {
		conf.ClientSecret = clientSecret.(string)
	}

	entry, err := logical.StorageEntryJSON(configRootKey, conf)
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"tailnet":       conf.Tailnet,
			"token":         conf.Token,
			"client_id":     conf.ClientID,
			"client_secret": conf.ClientSecret,
		},
	}, nil

}

func (b *backend) pathConfigTokenDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	if err := req.Storage.Delete(ctx, configRootKey); err != nil {
		return nil, err
	}
	return nil, nil
}

type rootTokenConfig struct {
	Token        string `json:"token,omitempty"`
	Tailnet      string `json:"tailnet,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}

const pathConfigTokenHelpSyn = `
Configure tailscale token and options used by vault
`

const pathConfigTokenHelpDesc = `
Will confugre this mount with the token used by Vault for all tailscale
operations on this mount. 

For instructions on how to get and/or create a tailscale api token see their
documentation at https://tailscale.com/kb/1101/api/.
`
