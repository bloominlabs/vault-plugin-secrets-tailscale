package tailscale

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/tailscale/tailscale-client-go/tailscale"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func pathListRoles(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "roles/?$",

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ListOperation: b.pathRoleList,
		},

		HelpSynopsis:    pathListRolesHelpSyn,
		HelpDescription: pathListRolesHelpDesc,
	}
}

func pathRoles(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "roles/" + framework.GenericNameWithAtRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Name of the role",
				DisplayAttrs: &framework.DisplayAttributes{
					Name: "Role Name",
				},
			},

			"capabilities": &framework.FieldSchema{
				Type: framework.TypeString,
				Description: `Capabilities the API token derived from this role will
        inherit
        (https://github.com/tailscale/tailscale/blob/main/api.md#post-apiv2tailnettailnetkeys---create-a-new-key-for-a-tailnet)`,
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.DeleteOperation: b.pathRolesDelete,
			logical.ReadOperation:   b.pathRolesRead,
			logical.UpdateOperation: b.pathRolesWrite,
		},

		HelpSynopsis:    pathRolesHelpSyn,
		HelpDescription: pathRolesHelpDesc,
	}
}

func (b *backend) pathRoleList(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	entries, err := req.Storage.List(ctx, "role/")
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(entries), nil
}

func (b *backend) pathRolesDelete(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	err := req.Storage.Delete(ctx, "role/"+d.Get("name").(string))
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathRolesRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	entry, err := b.roleRead(ctx, req.Storage, d.Get("name").(string))
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	var respPolicy map[string]interface{}
	inrec, _ := json.Marshal(entry)
	json.Unmarshal(inrec, &respPolicy)

	return &logical.Response{
		Data: respPolicy,
	}, nil
}

func (b *backend) pathRolesWrite(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	var resp logical.Response

	roleName := d.Get("name").(string)
	if roleName == "" {
		return logical.ErrorResponse("missing role name"), nil
	}

	roleEntry, err := b.roleRead(ctx, req.Storage, roleName)
	if err != nil {
		return nil, err
	}
	if roleEntry == nil {
		roleEntry = &tailscaleRoleEntry{}
	}

	if capabilitiesRaw, ok := d.GetOk("capabilities"); ok {
		var capabilities tailscale.KeyCapabilities
		s, ok := d.Get("capabilities").(string)
		if ok != true {
			return logical.ErrorResponse(fmt.Sprintf("cannot parse capabilities. raw: %q, err: %s", capabilitiesRaw.(string), err)), nil
		}

		err := json.Unmarshal([]byte(s), &capabilities)
		if err != nil {
			return logical.ErrorResponse(fmt.Sprintf("cannot parse capabilities. raw: %q, err: %s", capabilitiesRaw.(string), err)), nil
		}
		roleEntry.Capabilities = capabilities
	}

	var respData map[string]interface{}
	marshalledRole, err := json.Marshal(roleEntry)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(marshalledRole), &respData)
	if err != nil {
		return nil, err
	}

	entry, err := logical.StorageEntryJSON("role/"+roleName, respData)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, fmt.Errorf("nil result when writing to storage")
	}
	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	resp.Data = respData

	return &resp, nil
}

func (b *backend) roleRead(ctx context.Context, s logical.Storage, roleName string) (*tailscaleRoleEntry, error) {
	if roleName == "" {
		return nil, fmt.Errorf("missing role name")
	}
	entry, err := s.Get(ctx, "role/"+roleName)
	if err != nil {
		return nil, err
	}
	var roleEntry tailscaleRoleEntry
	if entry != nil {
		if err := entry.DecodeJSON(&roleEntry); err != nil {
			return nil, err
		}
		return &roleEntry, nil
	}

	return nil, nil
}

type tailscaleRoleEntry struct {
	Capabilities tailscale.KeyCapabilities `json:"capabilities"` // JSON-serialized capabilities to attach to tokens.
}

func compactJSON(input string) (string, error) {
	var compacted bytes.Buffer
	err := json.Compact(&compacted, []byte(input))
	return compacted.String(), err
}

const pathListRolesHelpSyn = `List the existing roles in this backend`

const pathListRolesHelpDesc = `Roles will be listed by the role name.`

const pathRolesHelpSyn = `
Read, write and reference tailscale capabilities that token can be made for.
`

const pathRolesHelpDesc = `
This path allows you to read and write capabilities that are used to
create tailscale tokens.`
