package data

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/jsmit257/userservice/shared/v1"
)

var maxfailure uint8 = 3 // config value

func (db *Conn) GetAuthByAttrs(ctx context.Context, id *shared.UUID, name *string, cid shared.CID) (*shared.BasicAuth, error) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "GetAuthByAttrs"})
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
	if err != nil {
		return nil, trackError(cid, db.logger, m, err, err.Error())
	}

	return result, nil
}

func (db *Conn) ResetPassword(ctx context.Context, old, new *shared.BasicAuth, cid shared.CID) error {
	_ = mtrcs.MustCurryWith(prometheus.Labels{"function": "ResetPassword"})

	auth, err := db.Login(ctx, old, cid)
	if err != nil {
		return err
	} else if hash(new.Pass, auth.Salt) == auth.Pass { // do other validation
		return shared.PasswordsMatch
	}
	db.logger.WithField("got", hash(new.Pass, auth.Salt)).WithField("want", auth.Pass).Errorf("wtf?!")
	salt := generateSalt()
	now := time.Now().UTC()

	err = db.updateBasicAuth(
		ctx,
		&shared.BasicAuth{
			Pass:         hash(new.Pass, salt),
			Salt:         salt,
			LoginSuccess: &now,
			LoginFailure: auth.LoginFailure,
			FailureCount: 0,
			UUID:         old.UUID,
		},
		cid)

	return err
}

func (db *Conn) Login(ctx context.Context, login *shared.BasicAuth, cid shared.CID) (*shared.BasicAuth, error) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "Login"})
	now := time.Now().UTC()

	result, err := db.GetAuthByAttrs(ctx, &login.UUID, nil, cid)
	if err != nil {
		return nil, trackError(cid, db.logger, m, err, err.Error()) // FIXME: fmt.Errorf("internal server error")
	} else if result.FailureCount > maxfailure {
		return nil, trackError(cid, db.logger, m, shared.MaxFailedLoginError, "password_lockout")
	} else if hash(login.Pass, result.Salt) != result.Pass {
		if err := db.updateBasicAuth(
			ctx,
			&shared.BasicAuth{
				UUID:         login.UUID,
				Pass:         result.Pass,
				Salt:         result.Salt,
				LoginSuccess: result.LoginSuccess,
				LoginFailure: &now,
				FailureCount: result.FailureCount + 1,
			},
			cid,
		); err != nil {
			return nil, trackError(cid, db.logger, m, err, err.Error())
		}
		return nil, trackError(cid, db.logger, m, shared.BadUserOrPassError, "bad_password")
	} else if err := db.updateBasicAuth(
		ctx,
		&shared.BasicAuth{
			UUID:         login.UUID,
			Pass:         result.Pass,
			Salt:         result.Salt,
			LoginSuccess: &now,
			LoginFailure: result.LoginFailure,
			FailureCount: result.FailureCount,
		},
		cid,
	); err != nil {
		return nil, err
	} else {
		result.LoginSuccess = &now
		m.WithLabelValues("none").Inc()
	}

	return result, err
}

func (db *Conn) updateBasicAuth(ctx context.Context, login *shared.BasicAuth, cid shared.CID) error {
	result, err := db.ExecContext(ctx, db.sqls["basic-auth"]["update"],
		login.Pass,
		login.Salt,
		login.LoginSuccess,
		login.LoginFailure,
		login.FailureCount,
		login.UUID)
	if err != nil {
		return err
	} else if updateCount, err := result.RowsAffected(); err != nil {
		return err
	} else if updateCount != 1 {
		return shared.UserNotUpdatedError
	}

	return nil
}
