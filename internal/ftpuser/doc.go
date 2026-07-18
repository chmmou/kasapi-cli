// Package ftpuser holds the domain types and use cases for the KAS FTP-user
// endpoints (get_ftpusers, add_ftpuser, update_ftpuser, delete_ftpuser).
// A single user is a get_ftpusers call with the ftp_login filter; there
// is no singular get action. See issues #11 and #13.
//
// Note: KAS docs spell the create action add_ftpusers (plural); the
// captured request fixture (#119, verified against the live API)
// confirms the real action is the singular add_ftpuser.
package ftpuser
