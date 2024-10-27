package shared

import "fmt"

var (
	UserExistsError     = fmt.Errorf("user already exists")
	UserNotAddedError   = fmt.Errorf("user was not added")
	UserNotUpdatedError = fmt.Errorf("user was not updated")
	UserNotDeletedError = fmt.Errorf("user was not deleted")

	AddressNotAddedError   = fmt.Errorf("address was not added")
	AddressNotUpdatedError = fmt.Errorf("address was not updated")
)
