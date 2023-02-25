package shared

import (
	sharedv1 "github.com/jsmit257/userservice/shared/v1"
)

// you don't have to alias types that don't change between versions, in
// which case the client has to import "v1", but if you do inherit unchanged
// types from the previous version, then the client only needs to import the
// *current* version - six of one, half dozen the other
type (
	BasicAuth sharedv1.BasicAuth
	User      sharedv1.User
	Address   sharedv1.Address
	Contact   sharedv1.Contact
)
