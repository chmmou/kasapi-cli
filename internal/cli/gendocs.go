package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// NewGenDocsCmd returns the hidden "kasapi-cli gen-docs <out-dir>"
// subcommand. It walks the assembled command tree (cmd.Root()) and
// writes one Markdown file per command into <out-dir> using
// cobra/doc.GenMarkdownTree, so docs/cli/ stays in sync with the
// actual flag and subcommand surface.
//
// DisableAutoGenTag is set on the root before generation so the output
// is reproducible — without it cobra appends a per-run timestamp to
// every file and `make docs` produces churn even when the CLI surface
// has not changed.
func NewGenDocsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gen-docs <out-dir>",
		Short: "Generate the Markdown CLI reference under <out-dir>",
		Long: "Walk the kasapi-cli command tree and write one Markdown file " +
			"per command into <out-dir>. Used by `make docs` to regenerate " +
			"docs/cli/. Hidden because it is a build-time helper, not a " +
			"user-facing operation.",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := args[0]
			if err := os.MkdirAll(out, 0o755); err != nil {
				return fmt.Errorf("gen-docs: mkdir %s: %w", out, err)
			}
			root := cmd.Root()
			root.DisableAutoGenTag = true
			if err := doc.GenMarkdownTree(root, out); err != nil {
				return fmt.Errorf("gen-docs: write %s: %w", out, err)
			}
			return nil
		},
	}
}
