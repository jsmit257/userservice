package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	sharedv1 "github.com/jsmit257/userservice/shared/v1"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	deleteUser      = "delete from user where id = ?"
	insertUser      = "insert into users(id, name, password, salt, mtime, ctime) values(?, ?, ?, ?, ?, ?)"
	selectBasicAuth = "select id, password, salt, login_success, login_failure, failure_count from users where name = ?"
	updateBasicAuth = "update user set login_success = ?, login_failure = ?, failure_count = ? where id = ?"
	selectUser      = "select name, mtime, dtime, login_success from users where id = ?"
	updatePassword  = "update users set password = ?, salt = ?, mtime = ? where id = ?"
	updateUser      = "update users set name = ?, mtime = ? where id = ?"

	maxFailure = 3
)

var (
	UserExistsError   = fmt.Errorf("user already exists")
	UserNotAddedError = fmt.Errorf("user was not added")
)

func (db *Conn) BasicAuth(ctx context.Context, login *sharedv1.BasicAuth, cid string) (*sharedv1.User, error) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"method": "BasicAuth"})

	var id, pass, salt string
	var loginSuccess, loginFailure *time.Time
	var failureCount uint8

	err := db.QueryRowContext(ctx, selectBasicAuth, login.Name).Scan(
		&id,
		&pass,
		&salt,
		&loginSuccess,
		&loginFailure,
		&failureCount)
	if err != nil {
		return nil, trackError(m, err, err.Error()) // FIXME: fmt.Errorf("internal server error")
	} else if id == "" {
		return nil, trackError(m, fmt.Errorf("bad username or password"), "bad_username")
	} else if failureCount > maxFailure {
		return nil, trackError(m, fmt.Errorf("too many failed login attempts"), "password_lockout")
	}

	now := time.Now().UTC()
	if hash(login.Pass, salt) != pass {
		if err := db.updateBasicAuth(ctx, id, loginSuccess, &now, failureCount+1); err != nil {
			return nil, trackError(m, err, err.Error()) // fmt.Errorf("internal server error"), err.Error())
		}
		return nil, trackError(m, fmt.Errorf("bad username or password"), "bad_password")
	}

	loginFailure = nil
	if err := db.updateBasicAuth(ctx, id, &now, nil, 0); err != nil {
		return nil, err
	}

	user, err := db.GetUser(ctx, id, cid)
	if err == nil {
		// for all intents and purposes, BasicAuth is successful (i.e "err":"none" in the metric) here,
		// GetUser may still fail with a separate metric; the context is the key to correlating the
		// nested calls, but i haven't figured that part out yet
		m.WithLabelValues("none").Inc()
	}

	// when authing, the returned user has the login_success from the previous successful logged-in session,
	// not this one, for anyone who actually pays attention
	user.LoginSuccess = loginSuccess
	return user, err
}

func (db *Conn) GetUser(ctx context.Context, id string, cid string) (*sharedv1.User, error) {
	result := &sharedv1.User{ID: id}

	return result, db.
		QueryRowContext(ctx, selectUser, id).
		Scan(&result.Name, &result.MTime, &result.DTime, &result.LoginSuccess)
}

func (db *Conn) AddUser(ctx context.Context, u *sharedv1.User, cid string) (string, error) {
	salt, now := generateSalt(), time.Now().UTC()
	result, err := db.ExecContext(ctx, insertUser,
		db.generateUUID(),
		u.Name,
		hash("password", salt),
		salt,
		now,
		now)
	if err != nil {
		switch v := err.(type) {
		case *mysql.MySQLError:
			// FIXME: is there no better way to determine key violation vs. some other error?
			if strings.Contains(v.Error(), "users.id") {
				return db.AddUser(ctx, u, cid) // FIXME: handle infinite recursion (unlikely as it is)
			} else if strings.Contains(v.Error(), "users.name") {
				return u.Name, UserExistsError
			}
		default:
			return u.Name, err // FIXME: should log the specifics and return something generic
		}
	} else if rows, err := result.RowsAffected(); err != nil {
		return u.Name, err // FIXME: should log the specifics and return something generic
	} else if rows != 1 {
		return u.Name, UserNotAddedError
	}
	return u.Name, nil
}

func (db *Conn) UpdateUser(ctx context.Context, u *sharedv1.User, cid string) error {
	curr, err := db.GetUser(ctx, u.ID, cid)
	if err != nil {
		return fmt.Errorf("failed to fetch user: '%s' %w", u.ID, err)
	}
	u.MTime = curr.MTime
	u.CTime = curr.CTime
	if *u == *curr {
		return nil
	}
	u.MTime = time.Now().UTC()
	if result, err := db.ExecContext(ctx, updateUser, u.Name, u.MTime, u.ID); err != nil {
		return fmt.Errorf("couldn't update user: '%s', %w", u.ID, err)
	} else if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows != 1 {
		return fmt.Errorf("user was not updated: '%s'", u.ID)
	}

	return nil
}

func (db *Conn) DeleteUser(ctx context.Context, id string, cid string) error {
	result, err := db.ExecContext(ctx, deleteUser, id)
	if err != nil {
		return err
	} else if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows != 1 {
		return fmt.Errorf("user could not be deleted")
	}
	return nil
}

func (db *Conn) CreateContact(ctx context.Context, id string, c *sharedv1.Contact, cid string) (string, error) {
	var err error
	if c.User, err = db.GetUser(ctx, id, cid); err != nil {
		return "", err
	}
	return db.AddContact(ctx, c, cid)
}

func (db *Conn) updateBasicAuth(ctx context.Context, id string, loginSuccess, loginFailure *time.Time, failureCount uint8) error {
	var err error
	var result sql.Result
	var updateCount int64

	result, err = db.ExecContext(ctx, updateBasicAuth, loginSuccess, loginFailure, failureCount, id)
	if err == nil {
		updateCount, err = result.RowsAffected()
		if err == nil && updateCount != 1 {
			return fmt.Errorf("basicAuth not updated")
		}
	}
	return err
}

func hash(pass, salt string) string {
	return pass + salt
}

func generateSalt() string {
	return "salt"
}

func trackError(m *prometheus.CounterVec, err error, lvs ...string) error {
	m.WithLabelValues(lvs...).Inc()
	// choose a logger and use it
	return err
}
