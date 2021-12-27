package tailscale

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"os"
// 	"testing"
// 	"time"
//
// 	"github.com/hashicorp/vault/sdk/logical"
// 	"github.com/stretchr/testify/assert"
// )
//
// func TestBackend_config_token(t *testing.T) {
// 	CLOUDFLARE_TOKEN := os.Getenv("TEST_CLOUDFLARE_TOKEN")
//
// 	config := logical.TestBackendConfig()
// 	config.StorageView = &logical.InmemStorage{}
// 	b, err := Factory(context.Background(), config)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	CLOUDFLARE_TOKEN_ID := ""
//
// 	client, _ := createClient(CLOUDFLARE_TOKEN)
//
// 	if client != nil {
// 		resp, _ := client.VerifyAPIToken(context.TODO())
//
// 		CLOUDFLARE_TOKEN_ID = resp.ID
// 	}
//
// 	testCases := []struct {
// 		name                  string
// 		configData            *rootTokenConfig
// 		expectedWriteResponse map[string]interface{}
// 		expectedReadResponse  map[string]interface{}
// 	}{
// 		{
// 			"errorsWithEmptyToken",
// 			nil,
// 			map[string]interface{}{"error": "Missing 'token' in configuration request"},
// 			map[string]interface{}{"error": "configuration does not exist. did you configure 'config/token'?"},
// 		},
// 		{
// 			"errorsWithInvalidCredentials",
// 			&rootTokenConfig{Token: "test"},
// 			map[string]interface{}{"error": "encountered error when verifying token: HTTP status 400: Invalid request headers (6003)"},
// 			map[string]interface{}{"error": "configuration does not exist. did you configure 'config/token'?"},
// 		},
// 		// TODO: add test to ensure the backend errors if the provided token is expired
// 		// {
// 		// 	"errorsWithInvalidatedToken",
// 		// 	&tokenConfig{Token: CLOUDFLARE_TOKEN},
// 		// 	nil,
// 		// 	map[string]interface{}{"token": CLOUDFLARE_TOKEN},
// 		// },
// 		{
// 			"succeedsWithValidToken",
// 			&rootTokenConfig{Token: CLOUDFLARE_TOKEN},
// 			nil,
// 			map[string]interface{}{"id": CLOUDFLARE_TOKEN_ID, "token": CLOUDFLARE_TOKEN},
// 		},
// 	}
//
// 	for _, testCase := range testCases {
// 		t.Run(testCase.name, func(t *testing.T) {
// 			confReq := &logical.Request{
// 				Operation: logical.UpdateOperation,
// 				Path:      "config/token",
// 				Storage:   config.StorageView,
// 				Data:      nil,
// 			}
//
// 			if testCase.configData != nil {
// 				confReq.Data = map[string]interface{}{
// 					"token": testCase.configData.Token,
// 				}
// 			}
//
// 			resp, err := b.HandleRequest(context.Background(), confReq)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
//
// 			if testCase.expectedWriteResponse == nil {
// 				assert.Nil(t, resp)
// 			} else {
// 				assert.Equal(t, testCase.expectedWriteResponse, resp.Data)
// 			}
//
// 			confReq.Operation = logical.ReadOperation
// 			resp, err = b.HandleRequest(context.Background(), confReq)
//
// 			assert.Equal(t, testCase.expectedReadResponse, resp.Data)
// 		})
// 	}
// }
//
// func TestBackend_rotate_root(t *testing.T) {
// 	CLOUDFLARE_TOKEN := os.Getenv("TEST_CLOUDFLARE_TOKEN")
//
// 	if CLOUDFLARE_TOKEN == "" {
// 		t.Skip("missing 'TEST_CLOUDFLARE_TOKEN'. skipping...")
// 	}
//
// 	config := logical.TestBackendConfig()
// 	config.StorageView = &logical.InmemStorage{}
// 	b, err := Factory(context.Background(), config)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	delta, _ := time.ParseDuration("30m")
// 	deltaFromNow := time.Now().UTC().Truncate(time.Second).Add(delta)
// 	testCases := []struct {
// 		name string
// 	}{
// 		{
// 			"happyPath",
// 		},
// 	}
// 	for _, testCase := range testCases {
// 		t.Run(testCase.name, func(t *testing.T) {
// 			client, err := createClient(CLOUDFLARE_TOKEN)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
//
// 			// Cloudflare does not allow creating tokens that can create other
// 			// management tokens 'sub-token is not allowed to have permissions to
// 			// manage other tokens' and a token is not allowed to roll itself unless
// 			// it can manage other tokens. So instead we create an empty token and
// 			// inject the token id of the empty token into the plugin configuration
// 			// since the 'config/rotate-roll' rotates the id of the token.
// 			emptyToken := cloudflare.APIToken{
// 				Name:      fmt.Sprintf("vault-integration-test-%s-%d", testCase.name, time.Now().UnixNano()),
// 				ExpiresOn: &deltaFromNow,
// 				Policies:  []cloudflare.APITokenPolicies{},
// 			}
// 			emptyToken, tokenCleanup, err := testCreateToken(t, client, emptyToken)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			defer tokenCleanup()
//
// 			conf := &rootTokenConfig{TokenID: emptyToken.ID, Token: CLOUDFLARE_TOKEN}
// 			entry, err := logical.StorageEntryJSON(configTokenKey, conf)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			if err := config.StorageView.Put(context.TODO(), entry); err != nil {
// 				t.Fatal(err)
// 			}
//
// 			confReq := &logical.Request{
// 				Operation: logical.UpdateOperation,
// 				Path:      "config/rotate-root",
// 				Storage:   config.StorageView,
// 				Data:      map[string]interface{}{},
// 			}
//
// 			resp, err := b.HandleRequest(context.Background(), confReq)
// 			if err != nil || (resp != nil && resp.IsError()) {
// 				t.Fatalf("failed to rotate token: resp:%#v err:%s", resp, err)
// 			}
// 			createdTokenID := resp.Data["id"].(string)
//
// 			// Verify that the token configured in the backend is still valid
// 			bClient, err := b.(*backend).client(context.TODO(), config.StorageView)
// 			verifyResp, _ := bClient.VerifyAPIToken(context.TODO())
// 			assert.Equal(t, "active", verifyResp.Status)
// 			assert.Equal(t, createdTokenID, verifyResp.ID)
//
// 			// Verify the original token has been rotated invalid
// 			client, err = createClient(emptyToken.Value)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			_, err = client.VerifyAPIToken(context.TODO())
// 			assert.Equal(t, err.Error(), "HTTP status 401: Invalid API Token (1000)")
//
// 			assert.Equal(t, createdTokenID, emptyToken.ID)
// 		})
// 	}
// }
//
// const synaticallyValidPolicy = `[
// 	{
// 		"effect": "allow",
// 		"resources": {
// 			"com.cloudflare.api.account.zone.eb78d65290b24279ba6f44721b3ea3c4": "*",
// 			"com.cloudflare.api.account.zone.22b1de5f1c0e4b3ea97bb1e963b06a43": "*"
// 		},
// 		"permission_groups": [
// 			{
// 				"id": "c8fed203ed3043cba015a93ad1616f1f",
// 				"name": "Zone Read"
// 			},
// 			{
// 				"id": "82e64a83756745bbbb1c9c2701bf816b",
// 				"name": "DNS Read"
// 			}
// 		]
// 	}
// ]`
//
// func TestBackend_roles(t *testing.T) {
// 	config := logical.TestBackendConfig()
// 	config.StorageView = &logical.InmemStorage{}
// 	b, err := Factory(context.Background(), config)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	compactedValidPolicy, err := compactJSON(synaticallyValidPolicy)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	testCases := []struct {
// 		name                  string
// 		policy                map[string]interface{}
// 		expectedWriteResponse map[string]interface{}
// 		expectedReadResponse  map[string]interface{}
// 	}{
// 		{
// 			"succeedsWithNilPolicyDocument",
// 			nil,
// 			map[string]interface{}{"policy_document": ""},
// 			nil,
// 		},
// 		{
// 			"succeedsWithMissingPolicyDocument",
// 			map[string]interface{}{"policy_document": ""},
// 			map[string]interface{}{"policy_document": ""},
// 			nil,
// 		},
// 		{
// 			"succeedsWithInvalidJSONPolicyDocument",
// 			map[string]interface{}{"policy_document": "{'}"},
// 			map[string]interface{}{"error": "cannot parse policy document: \"{'}\""},
// 			nil,
// 		},
// 		// TODO: add more validation to the parsed struct
// 		// {
// 		// 	"errorsWhenJSONIsntList",
// 		// 	map[string]interface{}{"policy_document": "{}"},
// 		// 	map[string]interface{}{"error": "cannot parse policy document: \"{'}\""},
// 		// 	nil,
// 		// },
// 		{
// 			"succeedsWithValidJSONPolicyDocument",
// 			map[string]interface{}{"policy_document": `[{"test": "test"}]`},
// 			map[string]interface{}{"policy_document": `[{"test":"test"}]`},
// 			map[string]interface{}{"policy_document": `[{"test":"test"}]`},
// 		},
// 		{
// 			"succeedsWithValidPolicyDocument",
// 			map[string]interface{}{"policy_document": synaticallyValidPolicy},
// 			map[string]interface{}{"policy_document": compactedValidPolicy},
// 			map[string]interface{}{"policy_document": compactedValidPolicy},
// 		},
// 	}
//
// 	for _, testCase := range testCases {
// 		t.Run(testCase.name, func(t *testing.T) {
// 			confReq := &logical.Request{
// 				Operation: logical.UpdateOperation,
// 				Path:      fmt.Sprintf("roles/%s", testCase.name),
// 				Storage:   config.StorageView,
// 				Data:      testCase.policy,
// 			}
//
// 			resp, err := b.HandleRequest(context.Background(), confReq)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
//
// 			assert.Equal(t, testCase.expectedWriteResponse, resp.Data)
//
// 			confReq = &logical.Request{
// 				Operation: logical.ReadOperation,
// 				Path:      fmt.Sprintf("roles/%s", testCase.name),
// 				Storage:   config.StorageView,
// 				Data:      testCase.policy,
// 			}
// 			resp, err = b.HandleRequest(context.Background(), confReq)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
//
// 			var respData map[string]interface{} = nil
// 			if testCase.expectedReadResponse != nil {
// 				respData = resp.Data
// 			}
// 			assert.Equal(t, testCase.expectedReadResponse, respData)
// 		})
// 	}
// }
//
// const validPolicy = `
// [{"effect":"allow","resources":{"com.cloudflare.api.account.zone.a1e23bc2933e158857087ff3310c4e40":"*"},"permission_groups":[{"id":"4755a26eedb94da69e1066d98aa820be","name":"DNS Write"}]}]
// `
//
// func TestBackend_creds_create(t *testing.T) {
// 	CLOUDFLARE_TOKEN := os.Getenv("TEST_CLOUDFLARE_TOKEN")
//
// 	if CLOUDFLARE_TOKEN == "" {
// 		t.Skip("missing 'TEST_CLOUDFLARE_TOKEN'. skipping...")
// 	}
//
// 	config := logical.TestBackendConfig()
// 	config.StorageView = &logical.InmemStorage{}
// 	b, err := Factory(context.Background(), config)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	testCases := []struct {
// 		name               string
// 		rolesData          map[string]interface{}
// 		credsData          map[string]interface{}
// 		expectedCredsError map[string]interface{}
// 	}{
// 		{
// 			"succeedsWithValidPolicyDocument",
// 			map[string]interface{}{"policy_document": validPolicy},
// 			nil,
// 			nil,
// 		},
// 		// TODO: add test for applying conditions to the api token
// 		//       https://api.cloudflare.com/#user-api-tokens-create-token
// 		// {
// 		// 	"succeedsWithValidPolicyDocument",
// 		// 	map[string]interface{}{"policy_document": validPolicy},
// 		// 	nil,
// 		// 	nil,
// 		// },
// 	}
//
// 	for _, testCase := range testCases {
// 		t.Run(testCase.name, func(t *testing.T) {
// 			confReq := &logical.Request{
// 				Operation: logical.UpdateOperation,
// 				Path:      "config/token",
// 				Storage:   config.StorageView,
// 				Data:      map[string]interface{}{"token": CLOUDFLARE_TOKEN},
// 			}
// 			resp, err := b.HandleRequest(context.Background(), confReq)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
//
// 			c, err := b.(*backend).client(context.TODO(), config.StorageView)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
//
// 			confReq = &logical.Request{
// 				Operation: logical.UpdateOperation,
// 				Path:      fmt.Sprintf("roles/%s", testCase.name),
// 				Storage:   config.StorageView,
// 				Data:      testCase.rolesData,
// 			}
// 			resp, err = b.HandleRequest(context.Background(), confReq)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
//
// 			confReq = &logical.Request{
// 				Operation: logical.ReadOperation,
// 				Path:      fmt.Sprintf("creds/%s", testCase.name),
// 				Storage:   config.StorageView,
// 				Data:      testCase.credsData,
// 			}
// 			resp, err = b.HandleRequest(context.Background(), confReq)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
//
// 			if resp == nil {
// 				t.Fatalf("'creds/%s' did not return a response", testCase.name)
// 			}
//
// 			// Prevents the token from leaking if we expected an error, but the token
// 			// was created anyway
// 			tokenID, ok := resp.Data["id"].(string)
// 			if ok {
// 				defer func() {
// 					err := c.DeleteAPIToken(context.TODO(), tokenID)
// 					if err != nil {
// 						t.Fatalf("failed to delete token '%s'. be sure it deleted in cloudflare", tokenID)
// 					}
// 				}()
// 			}
//
// 			if testCase.expectedCredsError != nil {
// 				assert.Equal(t, testCase.expectedCredsError, resp.Data)
// 				return
// 			}
//
// 			createdToken, err := c.GetAPIToken(context.TODO(), tokenID)
// 			if err != nil {
// 				t.Fatalf("failed to get token '%s'. err: %s", tokenID, err)
// 			}
//
// 			var expectedPolicies []cloudflare.APITokenPolicies
// 			err = json.Unmarshal([]byte(testCase.rolesData["policy_document"].(string)), &expectedPolicies)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			expectedPolicies[0].ID = createdToken.Policies[0].ID
// 			assert.Equal(t, expectedPolicies, createdToken.Policies)
// 		})
// 	}
// }
