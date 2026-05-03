package account

// Account is one row from get_accounts. The KAS API returns every field
// as xsd:string, including the numeric quotas and the Y/N flags; this
// package parses the obvious numerics into ints/floats but keeps Y/N
// flags as strings since some fields (account_2fa) accept "inherited"
// alongside Y and N.
type Account struct {
	Login            string  `json:"account_login" yaml:"account_login"`
	Password         string  `json:"account_password" yaml:"account_password"`
	MaxAccount       int     `json:"max_account" yaml:"max_account"`
	MaxDomain        int     `json:"max_domain" yaml:"max_domain"`
	MaxSubdomain     int     `json:"max_subdomain" yaml:"max_subdomain"`
	MaxWebspace      int     `json:"max_webspace" yaml:"max_webspace"`
	MaxMailAccount   int     `json:"max_mail_account" yaml:"max_mail_account"`
	MaxMailForward   int     `json:"max_mail_forward" yaml:"max_mail_forward"`
	MaxMailList      int     `json:"max_mail_list" yaml:"max_mail_list"`
	MaxDatabases     int     `json:"max_databases" yaml:"max_databases"`
	MaxFTPUser       int     `json:"max_ftpuser" yaml:"max_ftpuser"`
	MaxSambaUser     int     `json:"max_sambauser" yaml:"max_sambauser"`
	MaxCronjobs      int     `json:"max_cronjobs" yaml:"max_cronjobs"`
	MaxWBK           int     `json:"max_wbk" yaml:"max_wbk"`
	InstHtaccess     string  `json:"inst_htaccess" yaml:"inst_htaccess"`
	InstFPSE         string  `json:"inst_fpse" yaml:"inst_fpse"`
	InstSoftware     string  `json:"inst_software" yaml:"inst_software"`
	KASAccessForbid  string  `json:"kas_access_forbidden" yaml:"kas_access_forbidden"`
	Logging          string  `json:"logging" yaml:"logging"`
	Statistic        string  `json:"statistic" yaml:"statistic"`
	Logage           int     `json:"logage" yaml:"logage"`
	ShowPassword     string  `json:"show_password" yaml:"show_password"`
	DNSSettings      string  `json:"dns_settings" yaml:"dns_settings"`
	ShowDirectLinks  int     `json:"show_direct_links" yaml:"show_direct_links"`
	SSHAccess        string  `json:"ssh_access" yaml:"ssh_access"`
	UsedAccountSpace float64 `json:"used_account_space" yaml:"used_account_space"`
	Account2FA       string  `json:"account_2fa" yaml:"account_2fa"`
	// Account2FAInherited is only present on superuser/main-login responses.
	Account2FAInherited         string `json:"account_2fa_inherited,omitempty" yaml:"account_2fa_inherited,omitempty"`
	ShowDirectLinksWBK          string `json:"show_direct_links_wbk" yaml:"show_direct_links_wbk"`
	ShowDirectLinksSambausers   string `json:"show_direct_links_sambausers" yaml:"show_direct_links_sambausers"`
	ShowDirectLinksAccounts     string `json:"show_direct_links_accounts" yaml:"show_direct_links_accounts"`
	ShowDirectLinksMailaccounts string `json:"show_direct_links_mailaccounts" yaml:"show_direct_links_mailaccounts"`
	ShowDirectLinksFTPUser      string `json:"show_direct_links_ftpuser" yaml:"show_direct_links_ftpuser"`
	ShowDirectLinksDatabases    string `json:"show_direct_links_databases" yaml:"show_direct_links_databases"`
	AccountComment              string `json:"account_comment" yaml:"account_comment"`
	AccountContactMail          string `json:"account_contact_mail" yaml:"account_contact_mail"`
	InProgress                  string `json:"in_progress" yaml:"in_progress"`
}

// AccountSettings is the payload of get_accountsettings, which returns
// the settings for the authenticated account only. Most fields overlap
// with Account but settings carries SSH keys and user_prefs that are
// not part of the get_accounts list view.
type AccountSettings struct {
	Login              string                 `json:"account_login" yaml:"account_login"`
	AccountComment     string                 `json:"account_comment" yaml:"account_comment"`
	AccountContactMail string                 `json:"account_contact_mail" yaml:"account_contact_mail"`
	IsSuperuser        string                 `json:"is_superuser" yaml:"is_superuser"`
	AccountPassword    string                 `json:"account_password" yaml:"account_password"`
	ShowPassword       string                 `json:"show_password" yaml:"show_password"`
	Logging            string                 `json:"logging" yaml:"logging"`
	Logage             int                    `json:"logage" yaml:"logage"`
	Statistic          string                 `json:"statistic" yaml:"statistic"`
	DNSSettings        string                 `json:"dns_settings" yaml:"dns_settings"`
	InstHtaccess       string                 `json:"inst_htaccess" yaml:"inst_htaccess"`
	InstFPSE           string                 `json:"inst_fpse" yaml:"inst_fpse"`
	InstSoftware       string                 `json:"inst_software" yaml:"inst_software"`
	SSHAccess          string                 `json:"ssh_access" yaml:"ssh_access"`
	SSHKeys            string                 `json:"ssh_keys" yaml:"ssh_keys"`
	SSHFingerprints    map[string]Fingerprint `json:"ssh_fingerprints" yaml:"ssh_fingerprints"`
	SSHPHPVersion      string                 `json:"ssh_php_version" yaml:"ssh_php_version"`
	Server             string                 `json:"server" yaml:"server"`
	InProgress         string                 `json:"in_progress" yaml:"in_progress"`
	ShowDirectLinks    DirectLinkPrefs        `json:"show_direct_links" yaml:"show_direct_links"`
	Account2FA         string                 `json:"account_2fa" yaml:"account_2fa"`
	Account2FAInherit  string                 `json:"account_2fa_inherited,omitempty" yaml:"account_2fa_inherited,omitempty"`
	UserPrefs          UserPrefs              `json:"user_prefs" yaml:"user_prefs"`
}

// Fingerprint holds the per-algorithm SSH host-key fingerprints
// returned under settings.ssh_fingerprints[<algo>].
type Fingerprint struct {
	MD5    string `json:"md5" yaml:"md5"`
	SHA256 string `json:"sha256" yaml:"sha256"`
}

// DirectLinkPrefs groups the show_direct_links_* flags from settings.
type DirectLinkPrefs struct {
	WBK          string `json:"wbk" yaml:"wbk"`
	Sambausers   string `json:"sambausers" yaml:"sambausers"`
	Accounts     string `json:"accounts" yaml:"accounts"`
	Mailaccounts string `json:"mailaccounts" yaml:"mailaccounts"`
	FTPUser      string `json:"ftpuser" yaml:"ftpuser"`
	Databases    string `json:"databases" yaml:"databases"`
}

// UserPrefs is the user_prefs Map under settings. Only PerPage is
// well-typed; the table-related sub-keys are surfaced as raw maps so
// downstream consumers that need them can introspect without a new
// schema bump for every UI flag KAS adds.
type UserPrefs struct {
	PerPage          int                 `json:"per_page" yaml:"per_page"`
	ExpandableTables map[string][]string `json:"expandable_tables,omitempty" yaml:"expandable_tables,omitempty"`
	PinnedTables     map[string][]string `json:"pinned_tables,omitempty" yaml:"pinned_tables,omitempty"`
}

// ResourceQuota is one entry of the get_accountresources response. A
// Max of -1 means "unlimited" in KAS terminology; Free is also -1 in
// that case.
type ResourceQuota struct {
	Max      int  `json:"max" yaml:"max"`
	Exceeded bool `json:"exceeded" yaml:"exceeded"`
	Reserved int  `json:"reserved" yaml:"reserved"`
	Created  int  `json:"created" yaml:"created"`
	Used     int  `json:"used" yaml:"used"`
	Free     int  `json:"free" yaml:"free"`
}

// AccountResources is the payload of get_accountresources: a fixed set
// of named quotas. Unknown quota keys are dropped at the mapping layer
// so adding a new one to the API is a non-breaking change.
type AccountResources struct {
	MaxSubdomain   ResourceQuota `json:"max_subdomain" yaml:"max_subdomain"`
	MaxDomain      ResourceQuota `json:"max_domain" yaml:"max_domain"`
	MaxWBK         ResourceQuota `json:"max_wbk" yaml:"max_wbk"`
	MaxFTPUser     ResourceQuota `json:"max_ftpuser" yaml:"max_ftpuser"`
	MaxSambaUser   ResourceQuota `json:"max_sambauser" yaml:"max_sambauser"`
	MaxAccount     ResourceQuota `json:"max_account" yaml:"max_account"`
	MaxWebspace    ResourceQuota `json:"max_webspace" yaml:"max_webspace"`
	MaxDatabase    ResourceQuota `json:"max_database" yaml:"max_database"`
	MaxMailAccount ResourceQuota `json:"max_mail_account" yaml:"max_mail_account"`
	MaxMailForward ResourceQuota `json:"max_mail_forward" yaml:"max_mail_forward"`
	MaxMailingList ResourceQuota `json:"max_mailinglist" yaml:"max_mailinglist"`
	MaxCronjobs    ResourceQuota `json:"max_cronjobs" yaml:"max_cronjobs"`
}
