package shared

import (
	"context"
)

type (
	Addresser interface {
		GetAllAddresses(context.Context, CID) ([]Address, error)
		GetAddress(context.Context, UUID, CID) (*Address, error)
		AddAddress(context.Context, *Address, CID) (UUID, error)
		UpdateAddress(context.Context, *Address, CID) error
	}

	BasicAuther interface{}

	Contacter interface {
		UpdateContact(context.Context, UUID, *Contact, CID) error
	}

	Userer interface {
		GetAllUsers(context.Context, CID) ([]User, error)
		GetUser(context.Context, UUID, CID) (*User, error)
		AddUser(context.Context, *User, CID) (UUID, error)
		UpdateUser(context.Context, *User, CID) error
		DeleteUser(context.Context, UUID, CID) error
		CreateContact(context.Context, *User, Contact, CID) (*Contact, error)
	}
)
