package account

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// DecodeAccounts maps the ReturnInfo of a get_accounts response into
// the typed slice. ReturnInfo must be a Map[]; each entry is itself a
// Map carrying the documented fields.
func DecodeAccounts(returnInfo soap.Value) ([]Account, error) {
	if returnInfo.Kind != soap.KindArray {
		return nil, fmt.Errorf("account: expected ReturnInfo array, got kind %d", returnInfo.Kind)
	}
	out := make([]Account, 0, len(returnInfo.Array))
	for i, item := range returnInfo.Array {
		if item.Kind != soap.KindMap {
			return nil, fmt.Errorf("account: ReturnInfo[%d] is not a Map", i)
		}
		out = append(out, decodeAccount(item))
	}
	return out, nil
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
		Login:                       getString(m, "account_login"),
		Password:                    getString(m, "account_password"),
		MaxAccount:                  getInt(m, "max_account"),
		MaxDomain:                   getInt(m, "max_domain"),
		MaxSubdomain:                getInt(m, "max_subdomain"),
		MaxWebspace:                 getInt(m, "max_webspace"),
		MaxMailAccount:              getInt(m, "max_mail_account"),
		MaxMailForward:              getInt(m, "max_mail_forward"),
		MaxMailList:                 getInt(m, "max_mail_list"),
		MaxDatabases:                getInt(m, "max_databases"),
		MaxFTPUser:                  getInt(m, "max_ftpuser"),
		MaxSambaUser:                getInt(m, "max_sambauser"),
		MaxCronjobs:                 getInt(m, "max_cronjobs"),
		MaxWBK:                      getInt(m, "max_wbk"),
		InstHtaccess:                getString(m, "inst_htaccess"),
		InstFPSE:                    getString(m, "inst_fpse"),
		InstSoftware:                getString(m, "inst_software"),
		KASAccessForbid:             getString(m, "kas_access_forbidden"),
		Logging:                     getString(m, "logging"),
		Statistic:                   getString(m, "statistic"),
		Logage:                      getInt(m, "logage"),
		ShowPassword:                getString(m, "show_password"),
		DNSSettings:                 getString(m, "dns_settings"),
		ShowDirectLinks:             getInt(m, "show_direct_links"),
		SSHAccess:                   getString(m, "ssh_access"),
		UsedAccountSpace:            getFloat(m, "used_account_space"),
		Account2FA:                  getString(m, "account_2fa"),
		Account2FAInherited:         getString(m, "account_2fa_inherited"),
		ShowDirectLinksWBK:          getString(m, "show_direct_links_wbk"),
		ShowDirectLinksSambausers:   getString(m, "show_direct_links_sambausers"),
		ShowDirectLinksAccounts:     getString(m, "show_direct_links_accounts"),
		ShowDirectLinksMailaccounts: getString(m, "show_direct_links_mailaccounts"),
		ShowDirectLinksFTPUser:      getString(m, "show_direct_links_ftpuser"),
		ShowDirectLinksDatabases:    getString(m, "show_direct_links_databases"),
		AccountComment:              getString(m, "account_comment"),
		AccountContactMail:          getString(m, "account_contact_mail"),
		InProgress:                  getString(m, "in_progress"),
	}
}

func decodeSettings(m soap.Value) AccountSettings {
	return AccountSettings{
		Login:              getString(m, "account_login"),
		AccountComment:     getString(m, "account_comment"),
		AccountContactMail: getString(m, "account_contact_mail"),
		IsSuperuser:        getString(m, "is_superuser"),
		AccountPassword:    getString(m, "account_password"),
		ShowPassword:       getString(m, "show_password"),
		Logging:            getString(m, "logging"),
		Logage:             getInt(m, "logage"),
		Statistic:          getString(m, "statistic"),
		DNSSettings:        getString(m, "dns_settings"),
		InstHtaccess:       getString(m, "inst_htaccess"),
		InstFPSE:           getString(m, "inst_fpse"),
		InstSoftware:       getString(m, "inst_software"),
		SSHAccess:          getString(m, "ssh_access"),
		SSHKeys:            getString(m, "ssh_keys"),
		SSHFingerprints:    decodeFingerprints(m),
		SSHPHPVersion:      getString(m, "ssh_php_version"),
		Server:             getString(m, "server"),
		InProgress:         getString(m, "in_progress"),
		ShowDirectLinks: DirectLinkPrefs{
			WBK:          getString(m, "show_direct_links_wbk"),
			Sambausers:   getString(m, "show_direct_links_sambausers"),
			Accounts:     getString(m, "show_direct_links_accounts"),
			Mailaccounts: getString(m, "show_direct_links_mailaccounts"),
			FTPUser:      getString(m, "show_direct_links_ftpuser"),
			Databases:    getString(m, "show_direct_links_databases"),
		},
		Account2FA:        getString(m, "account_2fa"),
		Account2FAInherit: getString(m, "account_2fa_inherited"),
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
			MD5:    getString(kv.Value, "MD5:"),
			SHA256: getString(kv.Value, "SHA256:"),
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
		PerPage: getInt(up, "per_page"),
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
		Max:      getInt(v, "max"),
		Exceeded: getBool(v, "exceeded"),
		Reserved: getInt(v, "reserved"),
		Created:  getInt(v, "created"),
		Used:     getInt(v, "used"),
		Free:     getInt(v, "free"),
	}
}

func getString(m soap.Value, key string) string {
	v, ok := m.Get(key)
	if !ok {
		return ""
	}
	return v.AsString()
}

func getInt(m soap.Value, key string) int {
	v, ok := m.Get(key)
	if !ok {
		return 0
	}
	switch v.Kind {
	case soap.KindInt:
		return int(v.Int)
	case soap.KindFloat:
		return int(v.Float)
	case soap.KindString:
		s := strings.TrimSpace(v.String)
		if s == "" {
			return 0
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}

func getFloat(m soap.Value, key string) float64 {
	v, ok := m.Get(key)
	if !ok {
		return 0
	}
	return v.AsFloat()
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
