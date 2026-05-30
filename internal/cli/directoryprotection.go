package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/directoryprotection"
)

// NewDirectoryProtectionCmd returns the "kasapi-cli directoryprotection"
// subcommand tree: the list read endpoint plus the add / update /
// delete write endpoints (add/update/delete_directoryprotection).
//
// The KAS endpoint `get_directoryprotection` returns one entry per
// (path, user) tuple, so a directory with N users surfaces as N rows;
// for that reason the read side is exposed as a list with an optional
// `--path` filter rather than the usual list+get pair. A protection
// entry is likewise identified on the write side by the (path, user)
// pair, taken as the two positional arguments of add/update/delete.
//
// update and delete are gated by the #109 confirmation prompt: update
// replaces the access password (the previous one is unrecoverable) and
// delete revokes access, so both can lock users out. add is reversible
// (delete undoes it) and is not prompted.
func NewDirectoryProtectionCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "directoryprotection",
		Short: "Inspect and manage directory (htaccess) protections (get/add/update/delete_directoryprotection)",
	}
	cmd.AddCommand(
		newDirectoryProtectionListCmd(opts),
		newDirectoryProtectionAddCmd(opts),
		newDirectoryProtectionUpdateCmd(opts),
		newDirectoryProtectionDeleteCmd(opts),
	)
	return cmd
}

func newDirectoryProtectionListCmd(opts *RootOptions) *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List directory protections, optionally filtered by --path",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_directoryprotection", func(c *api.Client, ctx context.Context) (directoryprotection.DirectoryProtectionList, error) {
			return directoryprotection.NewClient(c).List(ctx, path)
		}),
	}
	cmd.Flags().StringVar(&path, "path", "",
		"directory path to filter on (e.g. /protected/directory/); empty returns every protection")
	return cmd
}

// dpIdent renders the (path, user) identity of a protection entry for
// the confirm prompt / audit target and the success line, so the
// directory and the affected user are both visible (a path can carry
// several protected users).
func dpIdent(path, user string) string {
	return path + " (user " + user + ")"
}

func newDirectoryProtectionAddCmd(opts *RootOptions) *cobra.Command {
	var password, authname string
	cmd := &cobra.Command{
		Use:   "add <path> <user> --password <pw> [--authname <name>]",
		Short: "Protect a path for a user (add_directoryprotection)",
		Args:  cobra.ExactArgs(2),
		RunE: runWriteE(opts, func(args []string) (writeSpec, error) {
			path, user := args[0], args[1]
			if password == "" {
				return writeSpec{}, fmt.Errorf("--password is required")
			}
			s := directoryprotection.Spec{User: user, Path: path, Password: password, AuthName: authname}
			return writeSpec{
				action:      "add_directoryprotection",
				destructive: false,
				confirm:     ConfirmAction{Verb: "create", Resource: "directory protection", ID: dpIdent(path, user)},
				params:      directoryprotection.AddParams(s),
				dispatch: func(c *api.Client, ctx context.Context) (string, error) {
					if _, derr := directoryprotection.NewClient(c).Add(ctx, s); derr != nil {
						return "", derr
					}
					return "created directory protection " + dpIdent(path, user), nil
				},
			}, nil
		}),
	}
	cmd.Flags().StringVar(&password, "password", "", "access password for the protected user (required)")
	cmd.Flags().StringVar(&authname, "authname", "", "htaccess realm label shown in the browser auth dialog (optional; directory_authname)")
	return cmd
}

func newDirectoryProtectionUpdateCmd(opts *RootOptions) *cobra.Command {
	var password, authname string
	cmd := &cobra.Command{
		Use:   "update <path> <user> [--password <pw>] [--authname <name>]",
		Short: "Replace the password and/or realm of a directory protection (update_directoryprotection)",
		Args:  cobra.ExactArgs(2),
	}
	cmd.RunE = runWriteE(opts, func(args []string) (writeSpec, error) {
		path, user := args[0], args[1]
		// Only the explicitly-set flags are sent (keyed on cobra
		// Changed): an omitted --password keeps the current one rather
		// than resetting it, and an omitted --authname keeps the realm.
		fields := map[string]string{}
		if cmd.Flags().Changed("password") {
			fields[directoryprotection.FieldPassword] = password
		}
		if cmd.Flags().Changed("authname") {
			fields[directoryprotection.FieldAuthName] = authname
		}
		if len(fields) == 0 {
			return writeSpec{}, fmt.Errorf("at least one field flag (--password/--authname) is required")
		}
		return writeSpec{
			action:      "update_directoryprotection",
			destructive: true,
			confirm:     ConfirmAction{Verb: "replace the settings of", Resource: "directory protection", ID: dpIdent(path, user)},
			params:      directoryprotection.UpdateParams(path, user, fields),
			dispatch: func(c *api.Client, ctx context.Context) (string, error) {
				if derr := directoryprotection.NewClient(c).Update(ctx, path, user, fields); derr != nil {
					return "", derr
				}
				return "updated directory protection " + dpIdent(path, user), nil
			},
		}, nil
	})
	cmd.Flags().StringVar(&password, "password", "", "replacement access password (sent as directory_password)")
	cmd.Flags().StringVar(&authname, "authname", "", "replacement htaccess realm label (directory_authname)")
	return cmd
}

func newDirectoryProtectionDeleteCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <path> <user>",
		Short: "Revoke a user's directory protection on a path (delete_directoryprotection)",
		Args:  cobra.ExactArgs(2),
		RunE: runWriteE(opts, func(args []string) (writeSpec, error) {
			path, user := args[0], args[1]
			return writeSpec{
				action:      "delete_directoryprotection",
				destructive: true,
				confirm:     ConfirmAction{Verb: "delete", Resource: "directory protection", ID: dpIdent(path, user)},
				params:      directoryprotection.DeleteParams(path, user),
				dispatch: func(c *api.Client, ctx context.Context) (string, error) {
					if derr := directoryprotection.NewClient(c).Delete(ctx, path, user); derr != nil {
						return "", derr
					}
					return "deleted directory protection " + dpIdent(path, user), nil
				},
			}, nil
		}),
	}
}
