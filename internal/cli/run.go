package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
)

// runListE returns a cobra RunE closure that follows the canonical
// read-subcommand shape: build the API client, run the supplied
// fetch, wrap a transport error as APIError(action), and render the
// result. It is the no-positional-argument variant — used by `list`,
// `info`, `space`, `space-detail`, `settings`, `resources`, and the
// other no-arg subcommands.
//
// fetch owns the module client construction and the actual call;
// callers typically pass a one-line closure such as
//
//	func(c *api.Client, ctx context.Context) (ddns.DDNSUserList, error) {
//	    return ddns.NewClient(c).List(ctx)
//	}
//
// Flag-derived parameters are passed in via the closure's capture.
func runListE[T any](
	opts *RootOptions,
	action string,
	fetch func(c *api.Client, ctx context.Context) (T, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		c, err := BuildAPIClient(opts)
		if err != nil {
			return err
		}
		v, err := fetch(c, cmd.Context())
		if err != nil {
			return APIError(err, action)
		}
		if err := Render(cmd.OutOrStdout(), opts.Output, v); err != nil {
			return UserError(err, "render")
		}
		return nil
	}
}

// runGetE is the cobra.ExactArgs(1) variant of runListE — fetch
// additionally receives args[0]. The standard caller passes the
// module's Client.Get method curried through a closure:
//
//	func(c *api.Client, ctx context.Context, arg string) (ddns.DDNSUser, error) {
//	    return ddns.NewClient(c).Get(ctx, arg)
//	}
func runGetE[T any](
	opts *RootOptions,
	action string,
	fetch func(c *api.Client, ctx context.Context, arg string) (T, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		c, err := BuildAPIClient(opts)
		if err != nil {
			return err
		}
		v, err := fetch(c, cmd.Context(), args[0])
		if err != nil {
			return APIError(err, action)
		}
		if err := Render(cmd.OutOrStdout(), opts.Output, v); err != nil {
			return UserError(err, "render")
		}
		return nil
	}
}
