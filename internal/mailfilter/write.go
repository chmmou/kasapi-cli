package mailfilter

import (
	"context"
	"errors"
	"strings"

	"github.com/chmmou/kasapi-cli/internal/kaswrite"
)

// ErrUnexpectedReturnString is the shared canonical post-call-contract
// sentinel, re-exported so errors.Is(err, mailfilter.ErrUnexpectedReturnString)
// keeps working and the slice stays self-describing. See
// kaswrite.ErrUnexpectedReturnString for the full contract.
var ErrUnexpectedReturnString = kaswrite.ErrUnexpectedReturnString

const (
	addAction    = "add_mailstandardfilter"
	deleteAction = "delete_mailstandardfilter"
)

// Field-prefixed constants are the KAS request keys add_mailstandardfilter
// and delete_mailstandardfilter accept. Both actions identify the target
// mailbox by mail_login; add additionally carries the joined filter chain
// in the single filter key.
//
// The filter chain is a semicolon-separated list of items where each item
// is either a bare filter id (e.g. "pdw") or "<filter-id>:<option>=<value>"
// (e.g. "spamc_move:move=Spam"). The captured add_mailstandardfilter
// request fixture is authoritative. delete_mailstandardfilter takes only
// mail_login and removes the whole configured chain in one shot — there is
// no per-item delete on the KAS API.
const (
	FieldMailLogin = "mail_login"
	FieldFilter    = "filter"
)

// JoinFilters builds the filter-chain string add_mailstandardfilter
// expects: semicolon-joined items, no leading or trailing separator. Empty
// items and items containing the separator itself are rejected because
// the KAS API has no escaping for either case and a typo would otherwise
// silently extend the chain.
func JoinFilters(items []string) (string, error) {
	if len(items) == 0 {
		return "", errors.New("mailfilter: at least one filter item is required")
	}
	for _, item := range items {
		if item == "" {
			return "", errors.New("mailfilter: filter item must not be empty")
		}
		if strings.Contains(item, ";") {
			return "", errors.New("mailfilter: filter item must not contain ';' (use a separate --filter)")
		}
	}
	return strings.Join(items, ";"), nil
}

// Add sets the standard-filter chain on a mail account
// (add_mailstandardfilter). The chain is sent verbatim as a single
// semicolon-joined string built from filterItems by JoinFilters; both
// mailLogin and a non-empty filterItems slice are validated before any
// SOAP call so the CLI can fail fast on an obviously incomplete request.
// Per the KAS docs the action replaces the configured chain wholesale.
func (cl *Client) Add(ctx context.Context, mailLogin string, filterItems []string) error {
	if mailLogin == "" {
		return errors.New("mailfilter: add_mailstandardfilter requires a non-empty mail login")
	}
	chain, err := JoinFilters(filterItems)
	if err != nil {
		return err
	}
	_, err = kaswrite.Call(ctx, cl.c, "mailfilter", addAction, AddParams(mailLogin, chain))
	return err
}

// AddParams builds the add_mailstandardfilter KAS request parameter map
// from an already-joined chain string. It is the single source of truth
// for the request shape so the CLI dry-run preview / audit record and
// the dispatched call cannot diverge. Callers obtain the chain from
// JoinFilters (which validates the items and rejects empties /
// embedded ';' separators) so an invalid slice cannot silently produce
// an empty chain string here.
func AddParams(mailLogin, chain string) map[string]any {
	return map[string]any{
		FieldMailLogin: mailLogin,
		FieldFilter:    chain,
	}
}

// Delete removes the entire standard-filter chain from a mail account
// (delete_mailstandardfilter). The KAS action takes only mail_login and
// drops every configured filter at once — there is no per-item delete.
//
// Observed quirk: even on a successful removal the API sometimes surfaces
// a generic envelope-level SOAP fault (the captured
// testdata/response_failed_internal_server_error.xml carries a PHP
// "sizeof(): Argument #1 ($value) must be of type Countable|array, null
// given" string in faultstring) while the filter is in fact gone on the
// server. We do NOT special-case that fault — the Caller surfaces it
// verbatim so the audit log and the exit code reflect what KAS reported.
// Callers can verify the actual outcome via mail accounts get <login>
// (the configured chain is reported in the mail_spamfilter field).
func (cl *Client) Delete(ctx context.Context, mailLogin string) error {
	if mailLogin == "" {
		return errors.New("mailfilter: delete_mailstandardfilter requires a non-empty mail login")
	}
	_, err := kaswrite.Call(ctx, cl.c, "mailfilter", deleteAction, DeleteParams(mailLogin))
	return err
}

// DeleteParams builds the delete_mailstandardfilter KAS request parameter
// map (single source of truth, see AddParams).
func DeleteParams(mailLogin string) map[string]any {
	return map[string]any{FieldMailLogin: mailLogin}
}
