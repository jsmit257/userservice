package shared

import "fmt"

var (
	UserExistsError     = fmt.Errorf("user already exists")
	UserNotAddedError   = fmt.Errorf("user was not added")
	UserNotUpdatedError = fmt.Errorf("user was not updated")
	UserNotDeletedError = fmt.Errorf("user was not deleted")

	AddressNotAddedError   = fmt.Errorf("address was not added")
	AddressNotUpdatedError = fmt.Errorf("address was not updated")

	BadUserOrPassError  = fmt.Errorf("bad username or password")
	MaxFailedLoginError = fmt.Errorf("too many failed login attempts")
	MissingAuthToken    = fmt.Errorf("missing auth token")

	PasswordsMatch = fmt.Errorf("passwords match")
)
