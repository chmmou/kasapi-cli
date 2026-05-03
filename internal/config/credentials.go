package config

import (
	"fmt"
	"os"
	"strings"
)

// Credentials are the resolved values for a single KAS API call.
type Credentials struct {
	Login    string
	AuthData string
	AuthType string
}

// String returns a representation safe for logs: AuthData is redacted.
func (c Credentials) String() string {
	if c.AuthData == "" {
		return fmt.Sprintf("config.Credentials{Login:%q AuthType:%q AuthData:<unset>}", c.Login, c.AuthType)
	}
	return fmt.Sprintf("config.Credentials{Login:%q AuthType:%q AuthData:<redacted %d bytes>}", c.Login, c.AuthType, len(c.AuthData))
}

// Override captures values supplied on the command line. Empty fields
// mean the flag was not given.
type Override struct {
	Profile  string
	Login    string
	AuthData string
	AuthType string
}

// Env captures the environment variables consulted during Resolve.
type Env struct {
	Login    string
	AuthData string
	AuthType string
}

// EnvFromOS reads the relevant environment variables from os.Getenv:
// KAS_LOGIN, KAS_AUTHDATA, KAS_AUTHTYPE.
func EnvFromOS() Env {
	return Env{
		Login:    os.Getenv("KAS_LOGIN"),
		AuthData: os.Getenv("KAS_AUTHDATA"),
		AuthType: os.Getenv("KAS_AUTHTYPE"),
	}
}

// Resolve applies the precedence flag > env > config(profile) >
// config(default_profile) and returns the credentials for a single
// call. The receiver may be nil — in that case only flags and env are
// consulted, and supplying ov.Profile is rejected.
func (c *Config) Resolve(env Env, ov Override) (Credentials, error) {
	var prof Profile
	if c != nil {
		name := ov.Profile
		if name == "" {
			name = c.DefaultProfile
		}
		if name != "" {
			p, ok := c.Profiles[name]
			if !ok {
				return Credentials{}, fmt.Errorf("config: profile %q not defined", name)
			}
			prof = p
		}
	} else if ov.Profile != "" {
		return Credentials{}, fmt.Errorf("config: --profile %q given but no config file loaded", ov.Profile)
	}

	cred := Credentials{
		Login:    pick(ov.Login, env.Login, prof.Login),
		AuthData: pick(ov.AuthData, env.AuthData, prof.AuthData),
		AuthType: pick(ov.AuthType, env.AuthType, prof.AuthType),
	}
	return cred, cred.validate()
}

func pick(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (c Credentials) validate() error {
	var missing []string
	if c.Login == "" {
		missing = append(missing, "login (--login or KAS_LOGIN)")
	}
	if c.AuthData == "" {
		missing = append(missing, "auth_data (--auth-data or KAS_AUTHDATA)")
	}
	if c.AuthType == "" {
		missing = append(missing, "auth_type (--auth-type or KAS_AUTHTYPE)")
	}
	if len(missing) > 0 {
		return fmt.Errorf("config: missing credentials: %s", strings.Join(missing, ", "))
	}
	if c.AuthType != AuthPlain && c.AuthType != AuthSession {
		return fmt.Errorf("config: auth_type %q must be %q or %q", c.AuthType, AuthPlain, AuthSession)
	}
	return nil
}
