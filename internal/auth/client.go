package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/transport"
)

// DefaultEndpoint is the production KasAuth.php URL. Set Client.Endpoint
// to override (e.g. in tests against an httptest.Server).
const DefaultEndpoint = "https://kasapi.kasserver.com/soap/KasAuth.php"

// Options carries the optional KasAuth parameters. Lifetime in seconds
// (1..30000); zero means "let the server apply its default". OTP is the
// 2FA PIN; empty when 2FA is disabled. UpdateLifetime maps to
// session_update_lifetime ("Y" / "N"); leave nil to omit.
type Options struct {
	Lifetime       int
	UpdateLifetime *bool
	OTP            string
}

// Client posts KasAuth requests through the supplied transport and
// returns the credential token from the response. SOAP faults surface
// as *Error with the stable KAS error string in Code.
//
// A zero Client is unusable; obtain one with New.
type Client struct {
	Transport *transport.Client
	Login     string
	AuthData  string
	AuthType  soap.AuthType
	Options   Options
	Endpoint  string
}

// New returns a Client configured for the given credentials and options.
// Endpoint defaults to DefaultEndpoint.
func New(t *transport.Client, login, authData string, authType soap.AuthType, opts Options) *Client {
	return &Client{
		Transport: t,
		Login:     login,
		AuthData:  authData,
		AuthType:  authType,
		Options:   opts,
		Endpoint:  DefaultEndpoint,
	}
}

// GetCredentialToken posts the KasAuth request and returns the
// 40-character credential token. Auth-specific faults surface as
// *Error; transport-level errors are returned wrapped.
func (c *Client) GetCredentialToken(ctx context.Context) (string, error) {
	if c.Login == "" || c.AuthData == "" || c.AuthType == "" {
		return "", errors.New("auth: Client has empty credential fields")
	}
	var buf bytes.Buffer
	req := Request{
		Login:          c.Login,
		AuthType:       c.AuthType,
		AuthData:       c.AuthData,
		Lifetime:       c.Options.Lifetime,
		UpdateLifetime: c.Options.UpdateLifetime,
		OTP:            c.Options.OTP,
	}
	if err := EncodeRequest(&buf, req); err != nil {
		return "", fmt.Errorf("auth: encode: %w", err)
	}
	body, err := c.Transport.Do(ctx, c.Endpoint, buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("auth: transport: %w", err)
	}
	token, err := DecodeResponse(bytes.NewReader(body))
	if err != nil {
		var fe *soap.FaultError
		if errors.As(err, &fe) {
			return "", newError(c.Login, fe)
		}
		return "", fmt.Errorf("auth: decode: %w", err)
	}
	return token, nil
}
