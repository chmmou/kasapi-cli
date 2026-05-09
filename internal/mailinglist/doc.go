// Package mailinglist holds the domain types and use cases for the KAS
// mailinglist endpoints (get_mailinglists in list and singular form,
// add_mailinglist, update_mailinglist, delete_mailinglist). The
// singular form is get_mailinglists with a mailinglist_name filter;
// the response is still an array, which Client.Get unwraps to a single
// MailingList. See issues #9 and #13.
package mailinglist
