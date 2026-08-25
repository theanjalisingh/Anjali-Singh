// Package auth implements login and logout using the existing identity schema.
//
// Login calls identity.sp_get_login_details, verifies password hashes in Go,
// and issues a JWT. It does not create identity tables or duplicate login procedures.
package auth
