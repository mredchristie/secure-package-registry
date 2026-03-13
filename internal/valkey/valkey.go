package valkey

import (
	"context"
	"fmt"

	"git.duti.dev/secure-package-registry/pkg/gitea"

	vk "github.com/valkey-io/valkey-go"
)

type Client struct {
	vk vk.Client
}

func NewClient(addr string) (*Client, error) {
	c, err := vk.NewClient(vk.ClientOption{
		InitAddress: []string{addr},
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to valkey: %w", err)
	}
	return &Client{vk: c}, nil
}

func (c *Client) Close() {
	c.vk.Close()
}

func (c *Client) GetGiteaConfig(ctx context.Context) (*gitea.Config, error) {
	data, err := c.vk.Do(ctx, c.vk.B().Get().Key("gitea:config").Build()).AsBytes()
	if err != nil {
		return nil, fmt.Errorf("reading gitea:config from valkey: %w", err)
	}
	return gitea.ParseConfig(data)
}
