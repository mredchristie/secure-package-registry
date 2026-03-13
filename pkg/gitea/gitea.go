package gitea

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Account struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

type Config struct {
	BaseURL  string             `json:"base_url"`
	Accounts map[string]Account `json:"accounts"`
}

func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing gitea config: %w", err)
	}
	if cfg.Accounts == nil {
		return nil, fmt.Errorf("gitea config has no accounts")
	}
	return &cfg, nil
}

type Client struct {
	baseURL    string
	config     *Config
	httpClient *http.Client
}

func NewClient(config *Config) *Client {
	return &Client{
		baseURL: config.BaseURL,
		config:  config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) Account(role string) (Account, error) {
	acct, ok := c.config.Accounts[role]
	if !ok {
		return Account{}, fmt.Errorf("gitea account %q not found in config", role)
	}
	return acct, nil
}

func (c *Client) NpmRegistry(role string) (*NpmRegistry, error) {
	acct, err := c.Account(role)
	if err != nil {
		return nil, err
	}
	return &NpmRegistry{
		client: c,
		owner:  acct.Username,
		token:  acct.Token,
	}, nil
}
