package account

import "strconv"

// AccountList is a slice wrapper that satisfies cli.Tabular so
// kasapi-cli accounts list --output=table works without a per-command
// renderer. The same Account values are emitted by --output=json|yaml
// directly.
type AccountList []Account

// TableHeaders returns the column names for the account list view.
func (AccountList) TableHeaders() []string {
	return []string{"LOGIN", "COMMENT", "MAIL", "MAX_DOMAIN", "MAX_WEBSPACE", "USED_MB", "2FA", "IN_PROGRESS"}
}

// TableRows returns one row per account, in the order returned by KAS.
func (l AccountList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, a := range l {
		rows = append(rows, []string{
			a.Login,
			a.AccountComment,
			a.AccountContactMail,
			strconv.Itoa(a.MaxDomain),
			strconv.Itoa(a.MaxWebspace),
			strconv.FormatFloat(a.UsedAccountSpace/1024, 'f', 1, 64),
			a.Account2FA,
			a.InProgress,
		})
	}
	return rows
}

// TableHeaders for AccountResources.
func (AccountResources) TableHeaders() []string {
	return []string{"RESOURCE", "MAX", "USED", "FREE", "RESERVED", "EXCEEDED"}
}

// TableRows emits the resource quotas in a stable, declared order.
func (r AccountResources) TableRows() [][]string {
	entries := []struct {
		name string
		q    ResourceQuota
	}{
		{"max_account", r.MaxAccount},
		{"max_domain", r.MaxDomain},
		{"max_subdomain", r.MaxSubdomain},
		{"max_webspace", r.MaxWebspace},
		{"max_database", r.MaxDatabase},
		{"max_mail_account", r.MaxMailAccount},
		{"max_mail_forward", r.MaxMailForward},
		{"max_mailinglist", r.MaxMailingList},
		{"max_ftpuser", r.MaxFTPUser},
		{"max_sambauser", r.MaxSambaUser},
		{"max_cronjobs", r.MaxCronjobs},
		{"max_wbk", r.MaxWBK},
	}
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, []string{
			e.name,
			quotaInt(e.q.Max),
			strconv.Itoa(e.q.Used),
			quotaInt(e.q.Free),
			strconv.Itoa(e.q.Reserved),
			strconv.FormatBool(e.q.Exceeded),
		})
	}
	return rows
}

// TableHeaders for AccountSettings produces a key/value layout, since
// the settings record is wide and tall-format is easier to read in a
// terminal than a 30-column row.
func (AccountSettings) TableHeaders() []string {
	return []string{"FIELD", "VALUE"}
}

// TableRows emits one row per scalar field. SSH fingerprints and the
// user_prefs sub-trees are summarised; consumers that need them should
// use --output=json or --output=yaml.
func (s AccountSettings) TableRows() [][]string {
	rows := [][]string{
		{"account_login", s.Login},
		{"account_comment", s.AccountComment},
		{"account_contact_mail", s.AccountContactMail},
		{"is_superuser", s.IsSuperuser},
		{"server", s.Server},
		{"logging", s.Logging},
		{"logage", strconv.Itoa(s.Logage)},
		{"statistic", s.Statistic},
		{"dns_settings", s.DNSSettings},
		{"inst_htaccess", s.InstHtaccess},
		{"inst_fpse", s.InstFPSE},
		{"inst_software", s.InstSoftware},
		{"ssh_access", s.SSHAccess},
		{"ssh_php_version", s.SSHPHPVersion},
		{"account_2fa", s.Account2FA},
		{"in_progress", s.InProgress},
		{"user_prefs.per_page", strconv.Itoa(s.UserPrefs.PerPage)},
	}
	return rows
}

// quotaInt formats -1 (KAS sentinel for "unlimited") as the symbol "∞"
// so table users do not have to know the convention.
func quotaInt(n int) string {
	if n < 0 {
		return "∞"
	}
	return strconv.Itoa(n)
}
