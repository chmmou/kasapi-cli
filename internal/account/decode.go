package account

import (
	"fmt"
	"strings"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// DecodeAccounts maps the ReturnInfo of a get_accounts response into
// the typed AccountList. ReturnInfo must be a Map[]; each entry is
// itself a Map carrying the documented fields.
func DecodeAccounts(returnInfo soap.Value) (AccountList, error) {
	out, err := soap.DecodeArray(returnInfo, "account", decodeAccount)
	if err != nil {
		return nil, err
	}
	return AccountList(out), nil
}

// DecodeAccountSettings maps the ReturnInfo of a get_accountsettings
// response into AccountSettings. The KAS payload nests the settings
// under ReturnInfo.settings.
func DecodeAccountSettings(returnInfo soap.Value) (AccountSettings, error) {
	settings, ok := returnInfo.Get("settings")
	if !ok {
		return AccountSettings{}, fmt.Errorf("account: ReturnInfo has no settings key")
	}
	if settings.Kind != soap.KindMap {
		return AccountSettings{}, fmt.Errorf("account: settings is not a Map (kind=%d)", settings.Kind)
	}
	return decodeSettings(settings), nil
}

// DecodeAccountResources maps the ReturnInfo of get_accountresources
// into the typed AccountResources struct. Unknown top-level keys are
// silently dropped.
func DecodeAccountResources(returnInfo soap.Value) (AccountResources, error) {
	if returnInfo.Kind != soap.KindMap {
		return AccountResources{}, fmt.Errorf("account: expected ReturnInfo Map, got kind %d", returnInfo.Kind)
	}
	var out AccountResources
	for _, kv := range returnInfo.Map {
		q := decodeQuota(kv.Value)
		switch kv.Key {
		case "max_subdomain":
			out.MaxSubdomain = q
		case "max_domain":
			out.MaxDomain = q
		case "max_wbk":
			out.MaxWBK = q
		case "max_ftpuser":
			out.MaxFTPUser = q
		case "max_sambauser":
			out.MaxSambaUser = q
		case "max_account":
			out.MaxAccount = q
		case "max_webspace":
			out.MaxWebspace = q
		case "max_database":
			out.MaxDatabase = q
		case "max_mail_account":
			out.MaxMailAccount = q
		case "max_mail_forward":
			out.MaxMailForward = q
		case "max_mailinglist":
			out.MaxMailingList = q
		case "max_cronjobs":
			out.MaxCronjobs = q
		}
	}
	return out, nil
}

func decodeAccount(m soap.Value) Account {
	return Account{
		Login:                       m.MapString("account_login"),
		Password:                    m.MapString("account_password"),
		MaxAccount:                  m.MapInt("max_account"),
		MaxDomain:                   m.MapInt("max_domain"),
		MaxSubdomain:                m.MapInt("max_subdomain"),
		MaxWebspace:                 m.MapInt("max_webspace"),
		MaxMailAccount:              m.MapInt("max_mail_account"),
		MaxMailForward:              m.MapInt("max_mail_forward"),
		MaxMailList:                 m.MapInt("max_mail_list"),
		MaxDatabases:                m.MapInt("max_databases"),
		MaxFTPUser:                  m.MapInt("max_ftpuser"),
		MaxSambaUser:                m.MapInt("max_sambauser"),
		MaxCronjobs:                 m.MapInt("max_cronjobs"),
		MaxWBK:                      m.MapInt("max_wbk"),
		InstHtaccess:                m.MapString("inst_htaccess"),
		InstFPSE:                    m.MapString("inst_fpse"),
		InstSoftware:                m.MapString("inst_software"),
		KASAccessForbid:             m.MapString("kas_access_forbidden"),
		Logging:                     m.MapString("logging"),
		Statistic:                   m.MapString("statistic"),
		Logage:                      m.MapInt("logage"),
		ShowPassword:                m.MapString("show_password"),
		DNSSettings:                 m.MapString("dns_settings"),
		ShowDirectLinks:             m.MapInt("show_direct_links"),
		SSHAccess:                   m.MapString("ssh_access"),
		UsedAccountSpace:            m.MapFloat("used_account_space"),
		Account2FA:                  m.MapString("account_2fa"),
		Account2FAInherited:         m.MapString("account_2fa_inherited"),
		ShowDirectLinksWBK:          m.MapString("show_direct_links_wbk"),
		ShowDirectLinksSambausers:   m.MapString("show_direct_links_sambausers"),
		ShowDirectLinksAccounts:     m.MapString("show_direct_links_accounts"),
		ShowDirectLinksMailaccounts: m.MapString("show_direct_links_mailaccounts"),
		ShowDirectLinksFTPUser:      m.MapString("show_direct_links_ftpuser"),
		ShowDirectLinksDatabases:    m.MapString("show_direct_links_databases"),
		AccountComment:              m.MapString("account_comment"),
		AccountContactMail:          m.MapString("account_contact_mail"),
		InProgress:                  m.MapString("in_progress"),
	}
}

func decodeSettings(m soap.Value) AccountSettings {
	return AccountSettings{
		Login:              m.MapString("account_login"),
		AccountComment:     m.MapString("account_comment"),
		AccountContactMail: m.MapString("account_contact_mail"),
		IsSuperuser:        m.MapString("is_superuser"),
		AccountPassword:    m.MapString("account_password"),
		ShowPassword:       m.MapString("show_password"),
		Logging:            m.MapString("logging"),
		Logage:             m.MapInt("logage"),
		Statistic:          m.MapString("statistic"),
		DNSSettings:        m.MapString("dns_settings"),
		InstHtaccess:       m.MapString("inst_htaccess"),
		InstFPSE:           m.MapString("inst_fpse"),
		InstSoftware:       m.MapString("inst_software"),
		SSHAccess:          m.MapString("ssh_access"),
		SSHKeys:            m.MapString("ssh_keys"),
		SSHFingerprints:    decodeFingerprints(m),
		SSHPHPVersion:      m.MapString("ssh_php_version"),
		Server:             m.MapString("server"),
		InProgress:         m.MapString("in_progress"),
		ShowDirectLinks: DirectLinkPrefs{
			WBK:          m.MapString("show_direct_links_wbk"),
			Sambausers:   m.MapString("show_direct_links_sambausers"),
			Accounts:     m.MapString("show_direct_links_accounts"),
			Mailaccounts: m.MapString("show_direct_links_mailaccounts"),
			FTPUser:      m.MapString("show_direct_links_ftpuser"),
			Databases:    m.MapString("show_direct_links_databases"),
		},
		Account2FA:        m.MapString("account_2fa"),
		Account2FAInherit: m.MapString("account_2fa_inherited"),
		UserPrefs:         decodeUserPrefs(m),
	}
}

func decodeFingerprints(settings soap.Value) map[string]Fingerprint {
	fp, ok := settings.Get("ssh_fingerprints")
	if !ok || fp.Kind != soap.KindMap {
		return nil
	}
	out := make(map[string]Fingerprint, len(fp.Map))
	for _, kv := range fp.Map {
		out[kv.Key] = Fingerprint{
			MD5:    kv.Value.MapString("MD5:"),
			SHA256: kv.Value.MapString("SHA256:"),
		}
	}
	return out
}

func decodeUserPrefs(settings soap.Value) UserPrefs {
	up, ok := settings.Get("user_prefs")
	if !ok || up.Kind != soap.KindMap {
		return UserPrefs{}
	}
	out := UserPrefs{
		PerPage: up.MapInt("per_page"),
	}
	if exp, ok := up.Get("expandableTables"); ok && exp.Kind == soap.KindMap {
		out.ExpandableTables = mapOfStringSlices(exp)
	}
	if pin, ok := up.Get("pinnedTables"); ok && pin.Kind == soap.KindMap {
		out.PinnedTables = mapOfStringSlices(pin)
	}
	return out
}

func mapOfStringSlices(m soap.Value) map[string][]string {
	out := make(map[string][]string, len(m.Map))
	for _, kv := range m.Map {
		switch kv.Value.Kind {
		case soap.KindArray:
			vals := make([]string, 0, len(kv.Value.Array))
			for _, item := range kv.Value.Array {
				vals = append(vals, item.AsString())
			}
			out[kv.Key] = vals
		case soap.KindString:
			out[kv.Key] = []string{kv.Value.String}
		}
	}
	return out
}

func decodeQuota(v soap.Value) ResourceQuota {
	if v.Kind != soap.KindMap {
		return ResourceQuota{}
	}
	return ResourceQuota{
		Max:      v.MapInt("max"),
		Exceeded: getBool(v, "exceeded"),
		Reserved: v.MapInt("reserved"),
		Created:  v.MapInt("created"),
		Used:     v.MapInt("used"),
		Free:     v.MapInt("free"),
	}
}

func getBool(m soap.Value, key string) bool {
	v, ok := m.Get(key)
	if !ok {
		return false
	}
	if v.Kind == soap.KindBool {
		return v.Bool
	}
	switch strings.ToLower(strings.TrimSpace(v.AsString())) {
	case "true", "1", "y", "yes":
		return true
	}
	return false
}
