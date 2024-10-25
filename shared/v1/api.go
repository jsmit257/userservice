package shared

import (
	"context"
)

type (
	Addresser interface {
		GetAddress(context.Context, UUID, CID) (*Address, error)
		AddAddress(context.Context, *Address, CID) (UUID, error)
		UpdateAddress(context.Context, *Address, CID) error
		DeleteAddress(context.Context, UUID, CID) error
	}

	Contacter interface {
		GetContact(context.Context, UUID, CID) (*Contact, error)
		AddContact(context.Context, UUID, *Contact, CID) (UUID, error)
		UpdateContact(context.Context, *Contact, CID) error
		DeleteContact(context.Context, UUID, CID) error
	}

	Userer interface {
		BasicAuth(context.Context, *BasicAuth, CID) (*User, error)
		GetUser(context.Context, UUID, CID) (*User, error)
		AddUser(context.Context, *User, CID) (UUID, error)
		UpdateUser(context.Context, *User, CID) error
		DeleteUser(context.Context, UUID, CID) error
		CreateContact(context.Context, *User, *Contact, CID) (*Contact, error)
	}
)
