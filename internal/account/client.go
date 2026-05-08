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

// List calls get_accounts without a filter and decodes the response
// into a slice of Account values. For a main login this returns every
// sub-account; for a sub-login it returns just the authenticated
// account.
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

// Get calls get_accounts with an account_login filter and returns the
// single matching Account. The KAS API still wraps the result in an
// array; we unwrap it here so callers do not have to. An empty array
// surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, login string) (Account, error) {
	if login == "" {
		return Account{}, fmt.Errorf("account: login is required")
	}
	resp, err := c.API.Call(ctx, "get_accounts", map[string]any{"account_login": login})
	if err != nil {
		return Account{}, err
	}
	accs, err := DecodeAccounts(resp.Body.ReturnInfo)
	if err != nil {
		return Account{}, fmt.Errorf("account: get_accounts: %w", err)
	}
	if len(accs) == 0 {
		return Account{}, fmt.Errorf("account: %q not found", login)
	}
	return accs[0], nil
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
