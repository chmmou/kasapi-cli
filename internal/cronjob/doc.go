// Package cronjob holds the domain types and use cases for the KAS cronjob
// endpoints (get_cronjobs, add_cronjob, update_cronjob, delete_cronjob).
// A single cronjob is a get_cronjobs call with the cronjob_id filter;
// there is no singular get action. See issues #11 and #13.
package cronjob
