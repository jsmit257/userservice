package shared

import "fmt"

var (
	UserExistsError     = fmt.Errorf("user already exists")
	UserNotAddedError   = fmt.Errorf("user was not added")
	UserNotUpdatedError = fmt.Errorf("user was not updated")
	UserNotDeletedError = fmt.Errorf("user was not deleted")

	AddressNotAddedError   = fmt.Errorf("address was not added")
	AddressNotUpdatedError = fmt.Errorf("address was not updated")

	BadUserOrPassError  CustomError = fmt.Errorf("bad username or password")
	MaxFailedLoginError CustomError = fmt.Errorf("too many failed login attempts")
	MissingAuthToken    CustomError = fmt.Errorf("missing auth token")

	RedisTokenFail = fmt.Errorf("failed redis login token")

	MissingParams = fmt.Errorf("parameter missing from URL")

	PasswordsMatch = fmt.Errorf("passwords match")

	Undeliverable = fmt.Errorf("there is no way to send this user reset data")
)
