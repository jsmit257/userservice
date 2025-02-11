package shared

import sharedv1 "github.com/jsmit257/userservice/shared/v1"

var (
	UserExistsError     = sharedv1.UserExistsError
	UserNotAddedError   = sharedv1.UserNotAddedError
	UserNotUpdatedError = sharedv1.UserNotUpdatedError
	UserNotDeletedError = sharedv1.UserNotDeletedError

	AddressNotAddedError   = sharedv1.AddressNotAddedError
	AddressNotUpdatedError = sharedv1.AddressNotUpdatedError

	BadUserOrPassError  = sharedv1.BadUserOrPassError
	MaxFailedLoginError = sharedv1.MaxFailedLoginError
	MissingAuthToken    = sharedv1.MissingAuthToken

	RedisTokenFail = sharedv1.RedisTokenFail

	MissingParams = sharedv1.MissingParams

	PasswordsMatch = sharedv1.PasswordsMatch

	Undeliverable = sharedv1.Undeliverable
)
