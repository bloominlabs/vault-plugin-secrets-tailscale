# Vault Secrets Plugin - Tailscale

[Vault][vault] secrets plugins to simplying creation, management, and
revocation of [Tailscale][tailscale] API tokens.

## Usage

### Setup Endpoint

1. Download and enable plugin locally (TODO)

2. Configure the plugin

   ```
   vault write /tailscale/config/root tailnet=<tailnet> token=<token>
   ```

3. Add one or more policies

### Configure Policies

```
# NOTE: this policy will not work and is just an example
vault write /tailscale/roles/<role-name> capabilities=-<<EOF
{
  "devices": {
    "create": {
      "reusable": false,
      "ephemeral": false
    }
  }
}
EOF
```

you can then read from the role using

```
vault read /tailscale/creds/<role-name>
```

### Generate a new Token

To generate a new token:

[Create a new tailscale policy](#configure-policies) and perform a 'read' operation on the `creds/<role-name>` endpoint.

```bash
# To read data using the api
$ vault read tailscale/role/single-use
Key                Value
---                -----
lease_id           tailscale/creds/test/yfF2qCtSvKSakATS89va1Var
lease_duration     768h
lease_renewable    false
capabilities       map[devices:map[create:map[]]]
expires            2022-03-27T03:13:45Z
id                 koD1dv6CNTRL
token              <token>
```

## Development

The provided [Earthfile] ([think makefile, but using
docker](https://earthly.dev)) is used to build, test, and publish the plugin.
See the build targets for more information. Common targets include

```bash
# build a local version of the plugin
$ earthly +build

# execute integration tests
#
# use https://developers.cloudflare.com/api/tokens/create to create a token
# with 'User:API Tokens:Edit' permissions
$ TEST_CLOUDFLARE_TOKEN=<YOUR_CLOUDFLARE_TOKEN> earthly --secret TEST_CLOUDFLARE_TOKEN +test

# start vault and enable the plugin locally
earthly +dev
```

[vault]: https://www.vaultproject.io/
[tailscale]: https://tailscale.com/
[earthfile]: ./Earthfile
