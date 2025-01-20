package shared

import (
	"context"
)

type (
	Addresser interface {
		GetAllAddresses(context.Context) ([]Address, error)
		GetAddress(context.Context, UUID) (*Address, error)
		AddAddress(context.Context, *Address) (UUID, error)
		UpdateAddress(context.Context, *Address) error
	}

	Auther interface {
		GetAuthByAttrs(context.Context, *UUID, *string) (*BasicAuth, error)
		ChangePassword(context.Context, *BasicAuth, *BasicAuth) error
		Login(context.Context, *BasicAuth) (*BasicAuth, error)
		ResetPassword(context.Context, *UUID) error
	}

	BasicAuther interface{}

	Contacter interface {
		UpdateContact(context.Context, UUID, *Contact) error
	}

	Userer interface {
		GetAllUsers(context.Context) ([]User, error)
		GetUser(context.Context, UUID) (*User, error)
		AddUser(context.Context, *User) (UUID, error)
		UpdateUser(context.Context, *User) error
		DeleteUser(context.Context, UUID) error
		CreateContact(context.Context, *User, Contact) (*Contact, error)
	}
)

func (a BasicAuth) Redact() BasicAuth {
	a.Pass = ""
	a.Salt = ""

	return a
}
