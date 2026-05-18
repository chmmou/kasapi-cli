package mailforward

import (
	"context"
	"errors"
	"strconv"

	"github.com/chmmou/kasapi-cli/internal/kaswrite"
)

// ErrUnexpectedReturnString is the shared canonical post-call-contract
// sentinel, re-exported so errors.Is(err, mailforward.ErrUnexpectedReturnString)
// keeps working and the slice stays self-describing. See
// kaswrite.ErrUnexpectedReturnString for the full contract.
var ErrUnexpectedReturnString = kaswrite.ErrUnexpectedReturnString

const (
	addAction    = "add_mailforward"
	updateAction = "update_mailforward"
	deleteAction = "delete_mailforward"
)

// Add creates a mail forward from localPart@domainPart to the given
// targets (add_mailforward). It returns the created forward address as
// echoed by the server in ReturnInfo (e.g. "info@example.de").
//
// localPart, domainPart and at least one target are required; an empty
// value is rejected before any SOAP call so the CLI can surface a fast
// validation error.
func (cl *Client) Add(ctx context.Context, localPart, domainPart string, targets []string) (string, error) {
	if localPart == "" || domainPart == "" {
		return "", errors.New("mailforward: add_mailforward requires a non-empty local and domain part")
	}
	if len(targets) == 0 {
		return "", errors.New("mailforward: add_mailforward requires at least one target")
	}
	resp, err := kaswrite.Call(ctx, cl.c, "mailforward", addAction, AddParams(localPart, domainPart, targets))
	if err != nil {
		return "", err
	}
	return resp.Body.ReturnInfo.AsString(), nil
}

// AddParams builds the add_mailforward KAS request parameter map. It is
// the single source of truth for the request shape so the CLI dry-run
// preview / audit record and the dispatched call cannot diverge.
func AddParams(localPart, domainPart string, targets []string) map[string]any {
	params := map[string]any{
		"local_part":  localPart,
		"domain_part": domainPart,
	}
	addTargets(params, targets)
	return params
}

// Update replaces the full target list of an existing mail forward
// (update_mailforward). The KAS API does not append: the supplied
// targets become the new complete set, so at least one is required.
func (cl *Client) Update(ctx context.Context, address string, targets []string) error {
	if address == "" {
		return errors.New("mailforward: update_mailforward requires a non-empty mail forward address")
	}
	if len(targets) == 0 {
		return errors.New("mailforward: update_mailforward requires at least one target")
	}
	_, err := kaswrite.Call(ctx, cl.c, "mailforward", updateAction, UpdateParams(address, targets))
	return err
}

// UpdateParams builds the update_mailforward KAS request parameter map
// (single source of truth, see AddParams).
func UpdateParams(address string, targets []string) map[string]any {
	params := map[string]any{"mail_forward": address}
	addTargets(params, targets)
	return params
}

// Delete removes a mail forward (delete_mailforward). A SOAP fault
// (e.g. mail_forward_not_found_in_kas, in_progress) is surfaced
// verbatim by the Caller so the caller can classify it via the api
// error helpers.
func (cl *Client) Delete(ctx context.Context, address string) error {
	if address == "" {
		return errors.New("mailforward: delete_mailforward requires a non-empty mail forward address")
	}
	_, err := kaswrite.Call(ctx, cl.c, "mailforward", deleteAction, DeleteParams(address))
	return err
}

// DeleteParams builds the delete_mailforward KAS request parameter map
// (single source of truth, see AddParams).
func DeleteParams(address string) map[string]any {
	return map[string]any{"mail_forward": address}
}

// addTargets writes the targets slice into params as the KAS-numbered
// target_0, target_1, … keys.
func addTargets(params map[string]any, targets []string) {
	for i, t := range targets {
		params["target_"+strconv.Itoa(i)] = t
	}
}
