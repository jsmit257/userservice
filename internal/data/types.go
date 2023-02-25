package data

import (
	"context"

	sharedv1 "github.com/jsmit257/userservice/shared/v1"
)

type (
	User interface {
		// BasicAuth returns an authenticated user on success
		BasicAuth(ctx context.Context, login *sharedv1.BasicAuth) (*sharedv1.User, error)
		// GetUser fetches a user by ID, or returns an error
		GetUser(ctx context.Context, id string) (*sharedv1.User, error)
		// AddUser creates a new user, the happy return path is a *username*, not ID
		// that can be used to call ResetPassword to complete an onboarding process;
		// tokenized authn and such are the business of the api layer
		AddUser(ctx context.Context, u *sharedv1.User) (string, error)
		// UpdateUser updates anything except password and salt; that's handled elsewhere
		UpdateUser(ctx context.Context, u *sharedv1.User) error
		// DeleteUser shouldn't ever be called
		DeleteUser(ctx context.Context, id string) error
		// CreateContact adds a contact to the current user
		CreateContact(ctx context.Context, id string, c *sharedv1.Contact) (string, error)
	}

	Address interface {
		// GetAddress fetches an Address by ID
		GetAddress(ctx context.Context, id string) (*sharedv1.Address, error)
		// AddAddress inserts a row into Addresses and returns the primary key, or an error
		AddAddress(ctx context.Context, addr *sharedv1.Address) (string, error)
		// UpdateAddress shouldn't really be called either; if a client wants to change
		// a bill-to or ship-to or whatever, then create a new address, and assign it to
		// their primary *-to - buildings don't move, users do (see Contact)
		UpdateAddress(ctx context.Context, addr *sharedv1.Address) error
		// DeleteAddress shouldn't ever be called
		DeleteAddress(ctx context.Context, id string) error
	}

	Contact interface {
		// GetContact returns Contact details with non-nill user object and optional billto
		// and shipto properties
		GetContact(ctx context.Context, userID string) (*sharedv1.Contact, error)
		// Addcontact creates a contact record associated with a user.id, but other referential
		// fields ignored; manage billing/shipping/user or whatever elsewhere; this should
		// probably only be called from User.CreateContact(string, *sharedv1.Contact), not from
		// the API
		AddContact(ctx context.Context, c *sharedv1.Contact) (string, error)
		// UpdateContact updates the non-referential fields in a contact; if e.g. a bill_to
		// object were present it would be ignored; that functionality is handled elsewhere
		UpdateContact(ctx context.Context, c *sharedv1.Contact) error
		// DeleteContact shouldn't ever be called
		DeleteContact(ctx context.Context, id string) error
		// SetDefaultBillto
		// SetDefaultShipto
		// GetBillTos
		// GetShipTos
	}
)
