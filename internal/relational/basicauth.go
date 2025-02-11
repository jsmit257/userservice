package data

import (
	"context"
	"time"

	"github.com/jsmit257/userservice/shared/v1"
)

var (
	maxfailure uint8 = 3 // config value
)

func (db *Conn) GetAuthByAttrs(ctx context.Context, id *shared.UUID, name *string) (*shared.BasicAuth, error) {
	done, log := db.logging("GetAuthByAttrs", id, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	result := &shared.BasicAuth{}
	err := db.QueryRowContext(ctx, db.sqls["basic-auth"]["select"], id, name).Scan(
		&result.UUID,
		&result.Name,
		&result.Pass,
		&result.Salt,
		&result.LoginSuccess,
		&result.LoginFailure,
		&result.FailureCount,
		&result.MTime,
		&result.CTime)

	return result, done(err, log)
}

func (db *Conn) ChangePassword(ctx context.Context, uid shared.UUID, old, new shared.Password) error {
	done, log := db.logging("ChangePassword", uid, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	auth, err := db.Login(ctx, &shared.BasicAuth{UUID: uid, Pass: old})
	if err != nil {
		return done(err, log)
	} else if err = validate(auth, new); err != nil {
		return done(err, log)
	}

	now := time.Now().UTC()

	auth.Salt = generateSalt()
	auth.Pass = hash(new, auth.Salt)
	auth.LoginSuccess = &now
	auth.FailureCount = 0

	err = db.updateBasicAuth(ctx, auth)

	return done(err, log)
}

func (db *Conn) SoftLogin(ctx context.Context, login *shared.BasicAuth) (*shared.BasicAuth, error) {
	done, log := db.logging("SoftLogin", login.UUID, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	result, err := db.GetAuthByAttrs(ctx, &login.UUID, nil)
	if err != nil {
		return result, done(err, log)
	} else if result.FailureCount > maxfailure {
		err = shared.MaxFailedLoginError
	} else if hash(login.Pass, result.Salt) != result.Pass {
		err = shared.BadUserOrPassError
	}

	return result, done(err, log)
}

func (db *Conn) Login(ctx context.Context, login *shared.BasicAuth) (*shared.BasicAuth, error) {
	done, log := db.logging("Login", login.UUID, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	now := time.Now().UTC()
	result, err := db.SoftLogin(ctx, login)
	if err == shared.BadUserOrPassError {
		if err = db.updateBasicAuth(
			ctx,
			&shared.BasicAuth{
				UUID:         result.UUID,
				Pass:         result.Pass,
				Salt:         result.Salt,
				LoginSuccess: result.LoginSuccess,
				LoginFailure: &now,
				FailureCount: result.FailureCount + 1,
			},
		); err == nil {
			err = shared.BadUserOrPassError
		}
	} else if err == nil {
		if err = db.updateBasicAuth(
			ctx,
			&shared.BasicAuth{
				UUID:         result.UUID,
				Pass:         result.Pass,
				Salt:         result.Salt,
				LoginSuccess: &now,
				LoginFailure: result.LoginFailure,
				FailureCount: 0,
			},
		); err == nil {
			result.LoginSuccess = &now
		}
	}

	return result, done(err, log)
}

func (db *Conn) ResetPassword(ctx context.Context, id *shared.UUID) error {
	done, log := db.logging("ResetPassword", id, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	auth, err := db.GetAuthByAttrs(ctx, id, nil)
	if err != nil {
		return done(err, log)
	}

	now := time.Now().UTC()

	auth.Salt = generateSalt()
	auth.Pass = hash(shared.Password(Obfuscate(string(auth.UUID))), auth.Salt)
	auth.LoginSuccess = &now
	auth.FailureCount = 0

	return done(db.updateBasicAuth(ctx, auth), log)
}

func (db *Conn) updateBasicAuth(ctx context.Context, login *shared.BasicAuth) error {
	done, log := db.logging("Login", nil, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	result, err := db.ExecContext(ctx, db.sqls["basic-auth"]["update"],
		login.Pass,
		login.Salt,
		login.LoginSuccess,
		login.LoginFailure,
		login.FailureCount,
		login.UUID)

	if err == nil {
		var rows int64
		if rows, err = result.RowsAffected(); err == nil && rows != 1 {
			err = shared.UserNotUpdatedError
		}
	}

	return done(err, log)
}

func validate(old *shared.BasicAuth, new shared.Password) error {
	if !new.Valid() { // all the complexity rules
		return shared.BadUserOrPassError
	} else if string(new) == old.Name { // can't use username as password
		return shared.PasswordsMatch
	} else if hash(new, old.Salt) == old.Pass { // can't re-use passwords
		return shared.PasswordsMatch
	}
	return nil
}
