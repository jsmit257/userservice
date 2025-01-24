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

func (db *Conn) ChangePassword(ctx context.Context, old, new *shared.BasicAuth) error {
	done, log := db.logging("ChangePassword", old.UUID, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	auth, err := db.Login(ctx, old)
	if err != nil {
		return done(err, log)
	} else if err = validate(auth.Name, new.Pass, auth.Pass, auth.Salt); err != nil {
		return done(err, log)
	}

	now := time.Now().UTC()

	auth.Salt = generateSalt()
	auth.Pass = hash(new.Pass, auth.Salt)
	auth.LoginSuccess = &now
	auth.FailureCount = 0

	err = db.updateBasicAuth(ctx, auth)

	return done(err, log)
}

func (db *Conn) Login(ctx context.Context, login *shared.BasicAuth) (*shared.BasicAuth, error) {
	done, log := db.logging("Login", login.UUID, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	now := time.Now().UTC()
	result, err := db.GetAuthByAttrs(ctx, &login.UUID, nil)
	if err != nil {
		return result, done(err, log)
	} else if result.FailureCount > maxfailure {
		err = shared.MaxFailedLoginError
	} else if hash(login.Pass, result.Salt) != result.Pass {
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
			err = shared.PasswordsMatch
		}
	} else if err = db.updateBasicAuth(
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

	return result, done(err, log)
}

func (db *Conn) ResetPassword(ctx context.Context, id *shared.UUID) error {
	done, log := db.logging("ResetPassword", id, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	auth, err := db.GetAuthByAttrs(ctx, id, nil)
	if err != nil {
		return done(err, log)
	}

	now := time.Now().UTC()

	auth.Pass = ""
	auth.Salt = ""
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

func validate(username, newpass, oldpass, oldsalt string) error {
	if len(newpass) < 8 {
		return shared.BadUserOrPassError
	} else if newpass == username {
		return shared.PasswordsMatch
	} else if hash(newpass, oldsalt) == oldpass {
		return shared.PasswordsMatch
	}
	return nil
}
