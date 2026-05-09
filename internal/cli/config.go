package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/chmmou/kasapi-cli/internal/config"
)

// NewConfigCmd returns the "kasapi-cli config" subcommand tree: init
// (interactive bootstrap), show (resolved effective config with
// auth_data redacted), and path (resolved config-file path).
func NewConfigCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and bootstrap the kasapi-cli configuration",
	}
	cmd.AddCommand(
		newConfigInitCmd(opts),
		newConfigShowCmd(opts),
		newConfigPathCmd(opts),
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
