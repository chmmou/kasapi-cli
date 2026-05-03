package account

import (
	"context"
	"fmt"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup: a fake Caller can
// return a *soap.Response decoded from a fixture.
type Caller interface {
	Call(ctx context.Context, action string, params map[string]any) (*soap.Response, error)
}

// Client groups the read endpoints for accounts: get_accounts,
// get_accountsettings, get_accountresources.
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller. It does not
// retain ownership of c.
func NewClient(c Caller) *Client { return &Client{API: c} }

// List calls get_accounts and decodes the response into a slice of
// Account values.
func (c *Client) List(ctx context.Context) ([]Account, error) {
	resp, err := c.API.Call(ctx, "get_accounts", nil)
	if err != nil {
		return nil, err
	}
	accs, err := DecodeAccounts(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("account: get_accounts: %w", err)
	}
	return accs, nil
}

// Settings calls get_accountsettings and decodes the response into the
// AccountSettings struct for the authenticated login.
func (c *Client) Settings(ctx context.Context) (AccountSettings, error) {
	resp, err := c.API.Call(ctx, "get_accountsettings", nil)
	if err != nil {
		return AccountSettings{}, err
	}
	s, err := DecodeAccountSettings(resp.Body.ReturnInfo)
	if err != nil {
		return AccountSettings{}, fmt.Errorf("account: get_accountsettings: %w", err)
	}
	return s, nil
}

// Resources calls get_accountresources and decodes the response into
// the AccountResources struct.
func (c *Client) Resources(ctx context.Context) (AccountResources, error) {
	resp, err := c.API.Call(ctx, "get_accountresources", nil)
	if err != nil {
		return AccountResources{}, err
	}
	r, err := DecodeAccountResources(resp.Body.ReturnInfo)
	if err != nil {
		return AccountResources{}, fmt.Errorf("account: get_accountresources: %w", err)
	}
	return r, nil
}
