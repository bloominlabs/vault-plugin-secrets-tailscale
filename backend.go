package tailscale

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

// backend wraps the backend framework and adds a map for storing key value pairs
type backend struct {
	*framework.Backend
}

var _ logical.Factory = Factory

// Factory configures and returns Mock backends
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b, err := newBackend()
	if err != nil {
		return nil, err
	}

	if conf == nil {
		return nil, fmt.Errorf("configuration passed into backend is nil")
	}

	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

func newBackend() (*backend, error) {
	b := &backend{}

	b.Backend = &framework.Backend{
		Help:        strings.TrimSpace(backendHelp),
		BackendType: logical.TypeLogical,
		Paths: framework.PathAppend(
			b.paths(),
		),
		Secrets: []*framework.Secret{
			secretToken(b),
		},
	}

	return b, nil
}

func (b *backend) paths() []*framework.Path {
	return []*framework.Path{
		pathConfigToken(b),
		pathCredsCreate(b),
		pathRoles(b),
		pathListRoles(b),
		pathConfigLease(b),
	}
}

const backendHelp = `
  The tailscale backend generates [tailscale auth key
  based](https://tailscale.com/kb/1085/auth-keys/) on [tailscale
  capabilities](https://github.com/tailscale/tailscale/blob/main/api.md#post-apiv2tailnettailnetkeys---create-a-new-key-for-a-tailnet).
  Tailscale tokens automatically expire after 90 days, but the backend provides
  as configurable TTL update to 90 days.

	After mounting this backend, a root token (with the ability to generate
	tokens) must be configured with the 'config/token' path and policies must
	be written using the "roles/" endpoints before any access keys can be
	generated.
`
