package shared

import (
	sharedv1 "github.com/jsmit257/userservice/shared/v1"
)

// you don't have to alias types that don't change between versions, in
// which case the client has to import "v1", but if you do inherit unchanged
// types from the previous version, then the client only needs to import the
// *current* version - six of one, half dozen the other

// from types.go
type (
	Cell     sharedv1.Cell
	CID      sharedv1.CID
	CTXKey   sharedv1.CTXKey
	Email    sharedv1.Email
	Password sharedv1.Password
	UUID     sharedv1.UUID

	Address   sharedv1.Address
	BasicAuth sharedv1.BasicAuth
	Contact   sharedv1.Contact
	User      sharedv1.User
)
