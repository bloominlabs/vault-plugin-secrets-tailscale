package tailscale

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	TAILSCALE_API_URL = "https://api.tailscale.com"
)

type Client struct {
	BaseURL string
	Tailnet string

	httpClient *http.Client
}

type Create struct {
	Reusable  bool `json:"reusable,omitempty"`
	Ephemeral bool `json:"ephemeral,omitempty"`
}

type Devices struct {
	Create Create `json:"create"`
}

// https://github.com/tailscale/tailscale/blob/main/api.md#post-apiv2tailnettailnetkeys---create-a-new-key-for-a-tailnet
type Capabilities struct {
	Devices Devices `json:"devices"`
}

type CreateAPIRequest struct {
	Capabilities Capabilities `json:"capabilities"`
}

type CreateAPIKeyResponse struct {
	ID           string       `json:"id"`
	Key          string       `json:"key"`
	Created      string       `json:"created"`
	Expires      string       `json:"expires"`
	Capabilities Capabilities `json:"capabilities"`
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errwrap.Wrapf("error attempting request: {{err}}", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 204 {
		defer resp.Body.Close()
		p, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to decode error response from tailscale. status_code: %d, err: %s",
				resp.StatusCode,
				err,
			)
		}
		return nil, errwrap.Wrapf(
			fmt.Sprintf(
				"error when performing request to tailscale. status_code: %d, err: %s",
				resp.StatusCode,
				string(p),
			),
			err,
		)
	}

	return resp, nil
}

func (c *Client) createAPIKey(ctx context.Context, capabilities Capabilities) (*CreateAPIKeyResponse, error) {
	url := fmt.Sprintf("%s/api/v2/tailnet/%s/keys", c.BaseURL, c.Tailnet)
	request := CreateAPIRequest{
		Capabilities: capabilities,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to create rotate-secret request. err %s", err))
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var decodedResponse CreateAPIKeyResponse
	err = json.NewDecoder(resp.Body).Decode(&decodedResponse)
	if err != nil {
		return nil, errwrap.Wrapf("fail to decode rotate secret response body: {{err}}", err)
	}

	return &decodedResponse, nil
}

func (c *Client) deleteAPIKey(ctx context.Context, keyID string) error {
	url := fmt.Sprintf("%s/api/v2/tailnet/%s/keys/%s", c.BaseURL, c.Tailnet, keyID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to create rotate-secret request. err %s", err))
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

type withHeader struct {
	http.Header
	rt http.RoundTripper
}

func WithHeader(rt http.RoundTripper) withHeader {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return withHeader{Header: make(http.Header), rt: rt}
}

func (h withHeader) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range h.Header {
		req.Header[k] = v
	}

	return h.rt.RoundTrip(req)
}

func createClientWithAPIKey(ctx context.Context, tailnet string, apiKey string) *Client {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	// tailscale uses basic authentication where the username is the api key
	rt := WithHeader(client.Transport)
	rt.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(apiKey+":")))
	client.Transport = rt

	return &Client{
		BaseURL:    TAILSCALE_API_URL,
		Tailnet:    tailnet,
		httpClient: client,
	}
}

func (b *backend) client(ctx context.Context, s logical.Storage) (*Client, error) {
	conf, err := b.readConfigToken(ctx, s)
	if err != nil {
		return nil, err
	}

	// TODO
	return createClientWithAPIKey(ctx, conf.Tailnet, conf.Token), nil
}
