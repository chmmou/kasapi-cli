// Package mailfilter holds the domain types and use cases for the KAS
// mail-standard-filter endpoints: get_mailstandardfilter (read; the
// catalogue of available filters) plus add_mailstandardfilter and
// delete_mailstandardfilter (write; set / clear the configured filter
// chain on a mail account, wired via the kaswrite seam). See issues #9,
// #13 and #116.
package mailfilter
