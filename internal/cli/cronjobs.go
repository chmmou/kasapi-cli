package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/cronjob"
)

// NewCronjobsCmd returns the "kasapi-cli cronjobs" subcommand tree:
// list/get (get_cronjobs, list and singular) plus the add / update /
// delete write endpoints (add_cronjob / update_cronjob /
// delete_cronjob). update and delete are gated by the #109
// confirmation prompt; add is reversible and not prompted.
func NewCronjobsCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cronjobs",
		Short: "Inspect and manage cronjobs (get/add/update/delete_cronjob)",
	}
	cmd.AddCommand(
		newCronjobsListCmd(opts),
		newCronjobsGetCmd(opts),
		newCronjobsAddCmd(opts),
		newCronjobsUpdateCmd(opts),
		newCronjobsDeleteCmd(opts),
	)
	return cmd
}

func newCronjobsListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all cronjobs (get_cronjobs, no filter)",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_cronjobs", func(c *api.Client, ctx context.Context) (cronjob.CronjobList, error) {
			return cronjob.NewClient(c).List(ctx)
		}),
	}
}

func newCronjobsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <cronjob-id>",
		Short: "Show details for a single cronjob (get_cronjobs with cronjob_id)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_cronjobs", func(c *api.Client, ctx context.Context, arg string) (cronjob.Cronjob, error) {
			return cronjob.NewClient(c).Get(ctx, arg)
		}),
	}
}

// cronjobWriteFlags binds the shared add_cronjob / update_cronjob
// request fields to a command. The same flag set serves both: add
// reads every value (defaults included), update sends only the flags
// the user explicitly changed (see cronjobChangedFields).
type cronjobWriteFlags struct {
	protocol      string
	url           string
	comment       string
	minute        string
	hour          string
	dayOfMonth    string
	month         string
	dayOfWeek     string
	httpUser      string
	httpPassword  string
	mailAddress   string
	mailCondition string
	mailSubject   string
	active        bool
}

func (f *cronjobWriteFlags) bind(cmd *cobra.Command) {
	fl := cmd.Flags()
	fl.StringVar(&f.protocol, "protocol", "https", "request protocol (http|https)")
	fl.StringVar(&f.url, "url", "", "URL to call (http_url; required for add)")
	fl.StringVar(&f.comment, "comment", "", "cronjob comment / label (required for add)")
	fl.StringVar(&f.minute, "minute", "", "schedule minute field (required for add)")
	fl.StringVar(&f.hour, "hour", "", "schedule hour field (required for add)")
	fl.StringVar(&f.dayOfMonth, "day-of-month", "*", "schedule day-of-month field")
	fl.StringVar(&f.month, "month", "*", "schedule month field")
	fl.StringVar(&f.dayOfWeek, "day-of-week", "*", "schedule day-of-week field (0-7, Sun=0|7)")
	fl.StringVar(&f.httpUser, "http-user", "", "HTTP basic-auth user for the call")
	fl.StringVar(&f.httpPassword, "http-password", "", "HTTP basic-auth password for the call")
	fl.StringVar(&f.mailAddress, "mail-address", "", "notification mail address (mail_adress)")
	fl.StringVar(&f.mailCondition, "mail-condition", "", "when to send the notification mail")
	fl.StringVar(&f.mailSubject, "mail-subject", "default", "notification mail subject (default|comment)")
	fl.BoolVar(&f.active, "active", true, "whether the cronjob is active; pass --active=false to disable")
}

func (f *cronjobWriteFlags) spec() cronjob.Spec {
	return cronjob.Spec{
		Protocol:      f.protocol,
		HTTPURL:       f.url,
		Comment:       f.comment,
		Minute:        f.minute,
		Hour:          f.hour,
		DayOfMonth:    f.dayOfMonth,
		Month:         f.month,
		DayOfWeek:     f.dayOfWeek,
		HTTPUser:      f.httpUser,
		HTTPPassword:  f.httpPassword,
		MailAdress:    f.mailAddress,
		MailCondition: f.mailCondition,
		MailSubject:   f.mailSubject,
		IsActive:      boolToYN(f.active),
	}
}

func boolToYN(b bool) string {
	if b {
		return "Y"
	}
	return "N"
}

func newCronjobsAddCmd(opts *RootOptions) *cobra.Command {
	f := &cronjobWriteFlags{}
	cmd := &cobra.Command{
		Use:   "add --url <url> --comment <text> --minute <m> --hour <h> [flags]",
		Short: "Create a cronjob (add_cronjob)",
		Args:  cobra.NoArgs,
		RunE: runWriteE(opts, func([]string) (writeSpec, error) {
			if f.url == "" {
				return writeSpec{}, fmt.Errorf("--url is required")
			}
			if f.comment == "" {
				return writeSpec{}, fmt.Errorf("--comment is required")
			}
			if f.minute == "" || f.hour == "" {
				return writeSpec{}, fmt.Errorf("--minute and --hour are required")
			}
			s := f.spec()
			return writeSpec{
				action:      "add_cronjob",
				destructive: false,
				confirm:     ConfirmAction{Verb: "create", Resource: "cronjob", ID: f.comment},
				params:      cronjob.AddParams(s),
				dispatch: func(c *api.Client, ctx context.Context) (string, error) {
					id, derr := cronjob.NewClient(c).Add(ctx, s)
					if derr != nil {
						return "", derr
					}
					return "created cronjob " + id, nil
				},
			}, nil
		}),
	}
	f.bind(cmd)
	return cmd
}

// cronjobChangedFields collects only the write flags the user
// explicitly set into the update_cronjob field map (keyed on the
// cronjob.Field* constants). Each field is a wholesale replacement and
// an empty value is a meaningful set, so presence is keyed on cobra
// Changed, not on the value being non-empty — the same pattern the
// mailing-list update uses.
func cronjobChangedFields(cmd *cobra.Command, f *cronjobWriteFlags) map[string]string {
	fields := map[string]string{}
	// flag name -> {KAS request key, current flag value}.
	for flag, kv := range map[string][2]string{
		"protocol":       {cronjob.FieldProtocol, f.protocol},
		"url":            {cronjob.FieldHTTPURL, f.url},
		"comment":        {cronjob.FieldComment, f.comment},
		"minute":         {cronjob.FieldMinute, f.minute},
		"hour":           {cronjob.FieldHour, f.hour},
		"day-of-month":   {cronjob.FieldDayOfMonth, f.dayOfMonth},
		"month":          {cronjob.FieldMonth, f.month},
		"day-of-week":    {cronjob.FieldDayOfWeek, f.dayOfWeek},
		"http-user":      {cronjob.FieldHTTPUser, f.httpUser},
		"http-password":  {cronjob.FieldHTTPPassword, f.httpPassword},
		"mail-address":   {cronjob.FieldMailAdress, f.mailAddress},
		"mail-condition": {cronjob.FieldMailCondition, f.mailCondition},
		"mail-subject":   {cronjob.FieldMailSubject, f.mailSubject},
	} {
		if cmd.Flags().Changed(flag) {
			fields[kv[0]] = kv[1]
		}
	}
	if cmd.Flags().Changed("active") {
		fields[cronjob.FieldIsActive] = boolToYN(f.active)
	}
	return fields
}

func newCronjobsUpdateCmd(opts *RootOptions) *cobra.Command {
	f := &cronjobWriteFlags{}
	cmd := &cobra.Command{
		Use:   "update <cronjob-id> [schedule/mail flags]",
		Short: "Replace mutable fields of a cronjob (update_cronjob)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.RunE = runWriteE(opts, func(args []string) (writeSpec, error) {
		id := args[0]
		fields := cronjobChangedFields(cmd, f)
		if len(fields) == 0 {
			return writeSpec{}, fmt.Errorf("at least one field flag (e.g. --comment/--minute/--active) is required")
		}
		return writeSpec{
			action:      "update_cronjob",
			destructive: true,
			confirm:     ConfirmAction{Verb: "replace the settings of", Resource: "cronjob", ID: id},
			params:      cronjob.UpdateParams(id, fields),
			dispatch: func(c *api.Client, ctx context.Context) (string, error) {
				if derr := cronjob.NewClient(c).Update(ctx, id, fields); derr != nil {
					return "", derr
				}
				return "updated cronjob " + id, nil
			},
		}, nil
	})
	f.bind(cmd)
	return cmd
}

func newCronjobsDeleteCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <cronjob-id>",
		Short: "Delete a cronjob (delete_cronjob)",
		Args:  cobra.ExactArgs(1),
		RunE: runWriteE(opts, func(args []string) (writeSpec, error) {
			id := args[0]
			return writeSpec{
				action:      "delete_cronjob",
				destructive: true,
				confirm:     ConfirmAction{Verb: "delete", Resource: "cronjob", ID: id},
				params:      cronjob.DeleteParams(id),
				dispatch: func(c *api.Client, ctx context.Context) (string, error) {
					if derr := cronjob.NewClient(c).Delete(ctx, id); derr != nil {
						return "", derr
					}
					return "deleted cronjob " + id, nil
				},
			}, nil
		}),
	}
}
