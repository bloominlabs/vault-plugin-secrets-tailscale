package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

func TestBackend_config_token(t *testing.T) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name                  string
		configData            *rootTokenConfig
		expectedWriteResponse map[string]interface{}
		expectedReadResponse  map[string]interface{}
	}{
		{
			"errorsWithEmptyRequest",
			nil,
			map[string]interface{}{"error": "Missing 'token' in configuration request"},
			map[string]interface{}{"error": "configuration does not exist. did you configure 'config/root'?"},
		},
		{
			"errorsWithEmptyToken",
			&rootTokenConfig{Tailnet: "test"},
			map[string]interface{}{"error": "Missing 'token' in configuration request"},
			map[string]interface{}{"error": "configuration does not exist. did you configure 'config/root'?"},
		},
		{
			"errorsWithEmptyTailnet",
			&rootTokenConfig{Token: "test"},
			map[string]interface{}{"error": "Missing 'tailnet' in configuration request"},
			map[string]interface{}{"error": "configuration does not exist. did you configure 'config/root'?"},
		},

		{
			"succeedsWithValidToken",
			&rootTokenConfig{Token: "test", Tailnet: "test"},
			nil,
			map[string]interface{}{"tailnet": "test", "token": "test"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			confReq := &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      "config/root",
				Storage:   config.StorageView,
				Data:      nil,
			}

			if testCase.configData != nil {
				var inInterface map[string]interface{}
				inrec, _ := json.Marshal(testCase.configData)
				json.Unmarshal(inrec, &inInterface)

				confReq.Data = inInterface
			}

			resp, err := b.HandleRequest(context.Background(), confReq)
			if err != nil {
				t.Fatal(err)
			}

			if testCase.expectedWriteResponse == nil {
				assert.Nil(t, resp)
			} else {
				assert.Equal(t, testCase.expectedWriteResponse, resp.Data)
			}

			confReq.Operation = logical.ReadOperation
			resp, err = b.HandleRequest(context.Background(), confReq)
			assert.Equal(t, testCase.expectedReadResponse, resp.Data)
		})
	}
}

const validTailscaleCapability = `{
	"capabilities": {
    "devices": {
      "create": {
        "reusable": false,
        "ephemeral": false
      }
    }
  }
}`

func TestBackend_roles(t *testing.T) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	compactedValidPolicy, err := compactJSON(validTailscaleCapability)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name                  string
		policy                map[string]interface{}
		expectedWriteResponse map[string]interface{}
		expectedReadResponse  map[string]interface{}
	}{
		{
			"succeedsWithNilPolicyDocument",
			nil,
			map[string]interface{}{"capabilities": ""},
			nil,
		},
		{
			"succeedsWithMissingPolicyDocument",
			map[string]interface{}{"capabilities": ""},
			map[string]interface{}{"capabilities": ""},
			nil,
		},
		{
			"failsWithInvalidJSONPolicyDocument",
			map[string]interface{}{"capabilities": "{'}"},
			map[string]interface{}{"error": "cannot parse capabilities: \"{'}\""},
			nil,
		},
		// TODO: add more validation to the parsed struct
		// {
		// 	"errorsWhenJSONIsntList",
		// 	map[string]interface{}{"capabilities": "{}"},
		// 	map[string]interface{}{"error": "cannot parse policy document: \"{'}\""},
		// 	nil,
		// },
		{
			"succeedsWithValidJSONCapabilities",
			map[string]interface{}{"capabilities": `[{"test": "test"}]`},
			map[string]interface{}{"capabilities": `[{"test":"test"}]`},
			map[string]interface{}{"capabilities": `[{"test":"test"}]`},
		},
		{
			"succeedsWithValidCapability",
			map[string]interface{}{"capabilities": validTailscaleCapability},
			map[string]interface{}{"capabilities": compactedValidPolicy},
			map[string]interface{}{"capabilities": compactedValidPolicy},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			confReq := &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      fmt.Sprintf("roles/%s", testCase.name),
				Storage:   config.StorageView,
				Data:      testCase.policy,
			}

			resp, err := b.HandleRequest(context.Background(), confReq)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, testCase.expectedWriteResponse, resp.Data)

			confReq = &logical.Request{
				Operation: logical.ReadOperation,
				Path:      fmt.Sprintf("roles/%s", testCase.name),
				Storage:   config.StorageView,
				Data:      testCase.policy,
			}
			resp, err = b.HandleRequest(context.Background(), confReq)
			if err != nil {
				t.Fatal(err)
			}

			var respData map[string]interface{} = nil
			if testCase.expectedReadResponse != nil {
				respData = resp.Data
			}
			assert.Equal(t, testCase.expectedReadResponse, respData)
		})
	}
}

const validPolicy = `{"capabilities":{"devices":{"create":{"reusable":false,"ephemeral":false}}}}`

func TestBackend_creds_create(t *testing.T) {
	TAILSCALE_TOKEN := os.Getenv("TEST_TAILSCALE_TOKEN")
	TAILSCALE_TAILNET := os.Getenv("TEST_TAILSCALE_TAILNET")

	if TAILSCALE_TOKEN == "" {
		t.Skip("missing 'TEST_CLOUDFLARE_TOKEN'. skipping...")
	}

	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	var validCapabilties Capabilities
	err = json.Unmarshal([]byte(validPolicy), &validCapabilties)

	testCases := []struct {
		name               string
		capabilities       *Capabilities
		credsData          map[string]interface{}
		expectedCredsError map[string]interface{}
	}{
		{
			"succeedsWithValidPolicyDocument",
			&validCapabilties,
			nil,
			nil,
		},
		// TODO: add test for applying conditions to the api token
		//       https://api.cloudflare.com/#user-api-tokens-create-token
		// {
		// 	"succeedsWithValidPolicyDocument",
		// 	map[string]interface{}{"capabilities": validPolicy},
		// 	nil,
		// 	nil,
		// },
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			confReq := &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      "config/root",
				Storage:   config.StorageView,
				Data:      map[string]interface{}{"token": TAILSCALE_TOKEN, "tailnet": TAILSCALE_TAILNET},
			}
			resp, err := b.HandleRequest(context.Background(), confReq)
			if err != nil {
				t.Fatal(err)
			}

			c, err := b.(*backend).client(context.TODO(), config.StorageView)
			if err != nil {
				t.Fatal(err)
			}

			confReq = &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      fmt.Sprintf("roles/%s", testCase.name),
				Storage:   config.StorageView,
				Data:      nil,
			}
			if testCase.capabilities != nil {
				var inInterface map[string]interface{}
				inrec, _ := json.Marshal(testCase.capabilities)
				json.Unmarshal(inrec, &inInterface)

				confReq.Data = inInterface
			}
			resp, err = b.HandleRequest(context.Background(), confReq)
			if err != nil {
				t.Fatal(err)
			}

			confReq = &logical.Request{
				Operation: logical.ReadOperation,
				Path:      fmt.Sprintf("creds/%s", testCase.name),
				Storage:   config.StorageView,
				Data:      testCase.credsData,
			}
			resp, err = b.HandleRequest(context.Background(), confReq)
			if err != nil {
				t.Fatal(err)
			}

			if resp == nil {
				t.Fatalf("'creds/%s' did not return a response", testCase.name)
			}

			// Prevents the token from leaking if we expected an error, but the token
			// was created anyway
			tokenID, ok := resp.Data["id"].(string)
			if ok {
				defer func() {
					err := c.deleteAPIKey(context.TODO(), tokenID)
					if err != nil {
						t.Fatalf("failed to delete token '%s'. be sure it deleted in cloudflare", tokenID)
					}
				}()
			}

			if testCase.expectedCredsError != nil {
				assert.Equal(t, testCase.expectedCredsError, resp.Data)
				return
			}

			createdToken, err := c.getAPIKey(context.TODO(), tokenID)
			if err != nil {
				t.Fatalf("failed to get token '%s'. err: %s", tokenID, err)
			}

			assert.Equal(t, createdToken.Capabilities, *testCase.capabilities)
		})
	}
}
