package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/config"
	"github.com/chmmou/kasapi-cli/internal/session"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/transport"
)

// NewConfigCmd returns the "kasapi-cli config" subcommand tree:
//   - init (interactive bootstrap of the first profile)
//   - show (resolved effective config with auth_data redacted)
//   - path (resolved config-file path)
//   - add-profile (interactive bootstrap of an additional profile)
//   - use-profile (switch default_profile + server-side revoke)
//   - list-profiles (alphabetical listing, default marked, auth_data redacted)
func NewConfigCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and bootstrap the kasapi-cli configuration",
	}
	cmd.AddCommand(
		newConfigInitCmd(opts),
		newConfigShowCmd(opts),
		newConfigPathCmd(opts),
		newConfigAddProfileCmd(opts),
		newConfigUseProfileCmd(opts),
		newConfigListProfilesCmd(opts),
	)
	return cmd
}

func newConfigPathCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the resolved config-file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := resolveConfigPath(opts.ConfigPath)
			if err != nil {
				return UserError(err, "")
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), path); err != nil {
				return UserError(err, "")
			}
			return nil
		},
	}
}

func newConfigShowCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the resolved effective config (auth_data redacted)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigShow(opts, cmd.OutOrStdout())
		},
	}
}

func runConfigShow(opts *RootOptions, out io.Writer) error {
	path, err := resolveConfigPath(opts.ConfigPath)
	if err != nil {
		return UserError(err, "")
	}
	cfg, loadErr := config.Load(path)
	if loadErr != nil && !errors.Is(loadErr, config.ErrNoConfig) {
		return UserError(loadErr, "load config")
	}
	creds, resolveErr := cfg.Resolve(config.EnvFromOS(), config.Override{
		Profile:  opts.Profile,
		Login:    opts.Login,
		AuthData: opts.AuthData,
		AuthType: opts.AuthType,
	})
	w := &writeErr{w: out}
	w.printf("config_path: %s\n", path)
	if errors.Is(loadErr, config.ErrNoConfig) {
		w.printf("config_file: <not found>\n")
	} else {
		w.printf("default_profile: %q\n", cfg.DefaultProfile)
		w.printf("profiles: %s\n", profileNames(cfg))
	}
	if opts.Profile != "" {
		w.printf("selected_profile: %q\n", opts.Profile)
	}
	if resolveErr != nil {
		w.printf("resolved: error: %v\n", resolveErr)
	} else {
		w.printf("resolved: %s\n", creds.String())
	}
	if w.err != nil {
		return UserError(w.err, "")
	}
	return nil
}

func newConfigInitCmd(opts *RootOptions) *cobra.Command {
	var (
		profileName string
		force       bool
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactively create or replace a profile in the config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cio := defaultConfigIO()
			cio.In = cmd.InOrStdin()
			cio.Out = cmd.OutOrStdout()
			return runConfigInit(opts.ConfigPath, profileName, force, cio)
		},
	}
	cmd.Flags().StringVar(&profileName, "name", "main", "profile name to write (the persistent --profile flag selects which profile is *used* at runtime)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing profile of the same name")
	return cmd
}

// configIO bundles the user-interaction surface of `config init` so
// tests can drive prompts and password input without a real terminal.
type configIO struct {
	In           io.Reader
	Out          io.Writer
	IsTTY        func() bool
	ReadPassword func() (string, error)
}

func defaultConfigIO() configIO {
	//nolint:gosec // G115: file descriptors fit in int on every platform Go targets; term.IsTerminal/term.ReadPassword take int.
	stdinFD := int(os.Stdin.Fd())
	return configIO{
		In:    os.Stdin,
		Out:   os.Stdout,
		IsTTY: func() bool { return term.IsTerminal(stdinFD) },
		ReadPassword: func() (string, error) {
			b, err := term.ReadPassword(stdinFD)
			if _, perr := fmt.Fprintln(os.Stdout); perr != nil && err == nil {
				err = perr
			}
			return string(b), err
		},
	}
}

func runConfigInit(configPath, profileName string, force bool, cio configIO) error {
	if !cio.IsTTY() {
		return UserError(errors.New("kasapi-cli config init requires an interactive terminal (stdin is not a TTY)"), "")
	}
	if profileName == "" {
		return UserError(errors.New("--name must not be empty"), "")
	}
	path, err := resolveConfigPath(configPath)
	if err != nil {
		return UserError(err, "")
	}
	cfg, err := config.Load(path)
	if err != nil && !errors.Is(err, config.ErrNoConfig) {
		return UserError(err, "load config")
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]config.Profile{}
	}
	if _, exists := cfg.Profiles[profileName]; exists && !force {
		return UserError(fmt.Errorf("profile %q already exists in %s (rerun with --force to overwrite)", profileName, path), "")
	}

	r := bufio.NewReader(cio.In)
	login, err := promptLine(r, cio.Out, "KAS login (e.g. w0000000): ")
	if err != nil {
		return UserError(err, "")
	}
	if login == "" {
		return UserError(errors.New("login is required"), "")
	}
	authType, err := promptAuthType(r, cio.Out)
	if err != nil {
		return UserError(err, "")
	}
	if _, perr := fmt.Fprint(cio.Out, "auth_data (input hidden): "); perr != nil {
		return UserError(perr, "")
	}
	authData, err := cio.ReadPassword()
	if err != nil {
		return UserError(err, "read password")
	}
	if authData == "" {
		return UserError(errors.New("auth_data is required"), "")
	}

	cfg.Profiles[profileName] = config.Profile{
		Login:    login,
		AuthData: authData,
		AuthType: authType,
	}
	if cfg.DefaultProfile == "" {
		ans, perr := promptLine(r, cio.Out, fmt.Sprintf("Set %q as default_profile? [Y/n]: ", profileName))
		if perr != nil {
			return UserError(perr, "")
		}
		if ans == "" || strings.EqualFold(ans, "y") || strings.EqualFold(ans, "yes") {
			cfg.DefaultProfile = profileName
		}
	}
	if err := cfg.Save(path); err != nil {
		return UserError(err, "")
	}
	if _, perr := fmt.Fprintf(cio.Out, "Wrote profile %q to %s\n", profileName, path); perr != nil {
		return UserError(perr, "")
	}
	return nil
}

func resolveConfigPath(configPath string) (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	return config.DefaultPath()
}

func promptLine(r *bufio.Reader, out io.Writer, prompt string) (string, error) {
	if _, err := fmt.Fprint(out, prompt); err != nil {
		return "", err
	}
	line, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func promptAuthType(r *bufio.Reader, out io.Writer) (string, error) {
	for {
		v, err := promptLine(r, out, fmt.Sprintf("auth_type (%s|%s) [%s]: ", config.AuthSession, config.AuthPlain, config.AuthSession))
		if err != nil {
			return "", err
		}
		switch v {
		case "":
			return config.AuthSession, nil
		case config.AuthPlain, config.AuthSession:
			return v, nil
		}
		if _, err := fmt.Fprintf(out, "  invalid auth_type %q (want %s or %s)\n", v, config.AuthSession, config.AuthPlain); err != nil {
			return "", err
		}
	}
}

func profileNames(cfg *config.Config) string {
	if cfg == nil {
		return "[]"
	}
	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return "[" + strings.Join(names, ", ") + "]"
}

// revokeFunc abstracts the server-side session-invalidation call so
// tests can drive `use-profile` switches without an httptest server.
// Production wires it to revokeSession; tests inject a spy.
type revokeFunc func(ctx context.Context, login, token string) error

// revokeSession server-side invalidates one cached session token by
// driving the session.Client delete_session use case with that token.
// The KAS API identifies the session via the (login, token) tuple
// supplied as auth_data / auth_type=session — no extra parameters are
// required. This function owns only the outer-layer wiring (transport +
// StaticTokenSource); the action string and success contract live in
// internal/session so config use-profile and `sessions delete` share
// one source of truth.
//
// endpoint is empty in production (uses api.DefaultEndpoint); tests
// point it at an httptest.Server so the real soap.Decode + api.Call
// pipeline runs against canned fixtures.
//
// Errors (transport, decode, or KAS fault) are returned so callers can
// log or classify them; whether they are propagated is the caller's
// decision (config use-profile swallows them best-effort, `sessions
// delete` surfaces non-unknown_session faults).
func revokeSession(ctx context.Context, login, token, endpoint string, logger *slog.Logger) error {
	tr := transport.New()
	tr.Logger = logger
	c := api.New(tr, &api.StaticTokenSource{
		Login:    login,
		AuthData: token,
		AuthType: soap.AuthSession,
	})
	c.Logger = logger
	if endpoint != "" {
		c.Endpoint = endpoint
	}
	return session.NewClient(c).Delete(ctx)
}

func newConfigAddProfileCmd(opts *RootOptions) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "add-profile <name>",
		Short: "Interactively add a new profile to the config file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cio := defaultConfigIO()
			cio.In = cmd.InOrStdin()
			cio.Out = cmd.OutOrStdout()
			return runConfigAddProfile(opts.ConfigPath, args[0], force, cio)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing profile of the same name")
	return cmd
}

// runConfigAddProfile is structurally identical to runConfigInit but
// takes the profile name as a positional argument and never offers a
// default-profile prompt on a non-empty existing config. It still sets
// default_profile to the new name when the file had no default before,
// matching `init`'s behaviour for a fresh file.
func runConfigAddProfile(configPath, name string, force bool, cio configIO) error {
	if !cio.IsTTY() {
		return UserError(errors.New("kasapi-cli config add-profile requires an interactive terminal (stdin is not a TTY)"), "")
	}
	if name == "" {
		return UserError(errors.New("profile name must not be empty"), "")
	}
	path, err := resolveConfigPath(configPath)
	if err != nil {
		return UserError(err, "")
	}
	cfg, err := config.Load(path)
	if err != nil && !errors.Is(err, config.ErrNoConfig) {
		return UserError(err, "load config")
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]config.Profile{}
	}
	if _, exists := cfg.Profiles[name]; exists && !force {
		return UserError(fmt.Errorf("profile %q already exists in %s (rerun with --force to overwrite)", name, path), "")
	}

	r := bufio.NewReader(cio.In)
	login, err := promptLine(r, cio.Out, "KAS login (e.g. w0000000): ")
	if err != nil {
		return UserError(err, "")
	}
	if login == "" {
		return UserError(errors.New("login is required"), "")
	}
	authType, err := promptAuthType(r, cio.Out)
	if err != nil {
		return UserError(err, "")
	}
	if _, perr := fmt.Fprint(cio.Out, "auth_data (input hidden): "); perr != nil {
		return UserError(perr, "")
	}
	authData, err := cio.ReadPassword()
	if err != nil {
		return UserError(err, "read password")
	}
	if authData == "" {
		return UserError(errors.New("auth_data is required"), "")
	}

	cfg.Profiles[name] = config.Profile{
		Login:    login,
		AuthData: authData,
		AuthType: authType,
	}
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = name
	}
	if err := cfg.Save(path); err != nil {
		return UserError(err, "")
	}
	if _, perr := fmt.Fprintf(cio.Out, "Wrote profile %q to %s\n", name, path); perr != nil {
		return UserError(perr, "")
	}
	return nil
}

func newConfigUseProfileCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "use-profile <name>",
		Short: "Switch the persistent default_profile and invalidate the outgoing session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := buildLogger(opts.Verbose)
			storePath, err := session.PathFor(opts.ConfigPath)
			if err != nil {
				return UserError(err, "session store")
			}
			store, err := session.New(storePath)
			if err != nil {
				return UserError(err, "session store")
			}
			revoke := func(ctx context.Context, login, token string) error {
				return revokeSession(ctx, login, token, "", logger)
			}
			return runConfigUseProfile(cmd.Context(), opts.ConfigPath, args[0], revoke, store, logger, cmd.OutOrStdout())
		},
	}
}

// runConfigUseProfile flips default_profile to name. Before writing it
// looks up the outgoing profile's login in sessions.toml and, if a
// non-expired token is cached, calls revoke to drop the session
// server-side. The on-disk cache entry is removed unconditionally
// afterwards because the local cache is the authoritative client-side
// state — a server-side revoke failure (network error,
// unknown_session fault) is logged but does not abort the switch.
func runConfigUseProfile(ctx context.Context, configPath, name string, revoke revokeFunc, store *session.Store, logger *slog.Logger, w io.Writer) error {
	if name == "" {
		return UserError(errors.New("profile name must not be empty"), "")
	}
	path, err := resolveConfigPath(configPath)
	if err != nil {
		return UserError(err, "")
	}
	cfg, err := config.Load(path)
	if err != nil {
		if errors.Is(err, config.ErrNoConfig) {
			return UserError(fmt.Errorf("no config file at %s (run `kasapi-cli config init` to create a profile interactively)", path), "")
		}
		return UserError(err, "load config")
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return UserError(fmt.Errorf("profile %q not defined in %s", name, path), "")
	}
	if name == cfg.DefaultProfile {
		if _, perr := fmt.Fprintf(w, "Profile %q is already the default in %s\n", name, path); perr != nil {
			return UserError(perr, "")
		}
		return nil
	}

	if outgoing := cfg.DefaultProfile; outgoing != "" {
		if prof, ok := cfg.Profiles[outgoing]; ok && prof.Login != "" {
			entry, lerr := store.Load(ctx, prof.Login)
			if lerr != nil {
				logger.Warn("config use-profile: session store load failed",
					"login", prof.Login, "err", lerr)
			}
			if entry != nil && entry.Token != "" {
				rerr := revoke(ctx, prof.Login, entry.Token)
				if rerr != nil {
					logger.Warn("config use-profile: server-side revoke failed (continuing)",
						"login", prof.Login, "err", rerr)
				}
				derr := store.Delete(ctx, prof.Login)
				if derr != nil {
					logger.Warn("config use-profile: session store delete failed",
						"login", prof.Login, "err", derr)
				}
				// Best-effort: a revoke/cache failure never aborts the
				// profile switch (see the function doc), but the message
				// must not imply a success that did not happen.
				var msg string
				switch {
				case rerr == nil && derr == nil:
					msg = fmt.Sprintf("Invalidated cached session for %q (login %s)\n", outgoing, prof.Login)
				case rerr != nil && derr == nil:
					msg = fmt.Sprintf("Cleared the local session cache for %q (login %s); server-side delete_session failed (see --verbose) — continuing\n", outgoing, prof.Login)
				case rerr == nil && derr != nil:
					msg = fmt.Sprintf("Revoked the server-side session for %q (login %s); the local cache could NOT be cleared (see --verbose) — continuing\n", outgoing, prof.Login)
				default:
					msg = fmt.Sprintf("Could not fully invalidate the cached session for %q (login %s): both server-side delete_session and local cache removal failed (see --verbose) — continuing\n", outgoing, prof.Login)
				}
				if _, perr := fmt.Fprint(w, msg); perr != nil {
					return UserError(perr, "")
				}
			}
		}
	}

	cfg.DefaultProfile = name
	if err := cfg.Save(path); err != nil {
		return UserError(err, "")
	}
	if _, perr := fmt.Fprintf(w, "Switched default_profile to %q in %s\n", name, path); perr != nil {
		return UserError(perr, "")
	}
	return nil
}

func newConfigListProfilesCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list-profiles",
		Short: "List configured profiles and their auth_type (auth_data redacted)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigListProfiles(opts.ConfigPath, cmd.OutOrStdout())
		},
	}
}

// runConfigListProfiles prints one line per configured profile,
// alphabetically sorted, prefixed with "* " for the default. auth_data
// is never written. A missing config file prints a discoverability
// hint pointing at `config init`, mirroring the first-run pathway
// used by BuildAPIClient (#138).
func runConfigListProfiles(configPath string, w io.Writer) error {
	path, err := resolveConfigPath(configPath)
	if err != nil {
		return UserError(err, "")
	}
	cfg, err := config.Load(path)
	if err != nil {
		if errors.Is(err, config.ErrNoConfig) {
			_, perr := fmt.Fprintf(w, "no profiles configured (run `kasapi-cli config init` to create one interactively)\n")
			if perr != nil {
				return UserError(perr, "")
			}
			return nil
		}
		return UserError(err, "load config")
	}
	if len(cfg.Profiles) == 0 {
		_, perr := fmt.Fprintf(w, "no profiles configured in %s (run `kasapi-cli config init` to create one interactively)\n", path)
		if perr != nil {
			return UserError(perr, "")
		}
		return nil
	}
	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	we := &writeErr{w: w}
	for _, n := range names {
		mark := "  "
		if n == cfg.DefaultProfile {
			mark = "* "
		}
		// Empty auth_type is rare (init / add-profile always write one
		// via promptAuthType), but possible if the TOML was edited by
		// hand. Show "no auth_type" so the user sees the real state
		// instead of an optimistic "(session)" default that may not
		// match what `cfg.Resolve` will accept.
		authType := cfg.Profiles[n].AuthType
		if authType == "" {
			authType = "no auth_type"
		}
		we.printf("%s%s (%s)\n", mark, n, authType)
	}
	if we.err != nil {
		return UserError(we.err, "")
	}
	return nil
}

// writeErr is a fmt.Fprintf wrapper that records the first error so a
// caller can issue a series of writes and only check err at the end.
type writeErr struct {
	w   io.Writer
	err error
}

func (we *writeErr) printf(format string, args ...any) {
	if we.err != nil {
		return
	}
	_, we.err = fmt.Fprintf(we.w, format, args...)
}
