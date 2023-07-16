package mysql

import (
	"context"
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
	selectBasicAuth = "select id, password, salt from users where name = ?"
	selectUser      = "select name, mtime, dtime from users where id = ?"
	updatePassword  = "update users set password = ?, salt = ?, mtime = ? where id = ?"
	updateUser      = "update users set name = ?, mtime = ? where id = ?"
)

var (
	UserExistsError   = fmt.Errorf("user already exists")
	UserNotAddedError = fmt.Errorf("user was not added")
)

func (db *Conn) BasicAuth(ctx context.Context, login *sharedv1.BasicAuth) (*sharedv1.User, error) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"method": "BasicAuth"})
	var id, pass, salt string
	if err := db.QueryRowContext(ctx, selectBasicAuth, login.Name).Scan(&id, &pass, &salt); err != nil {
		m.WithLabelValues(err.Error()).Inc()
		return nil, fmt.Errorf("bad username or password")
	}
	if hash(login.Pass, salt) != pass {
		m.WithLabelValues("bad_password").Inc()
		return nil, fmt.Errorf("bad username or password")
	}
	// for all intents and purposes, BasicAuth is successful (i.e "err":"none" in the metric) here,
	// GetUser may still fail with a separate metric; the context is the key to correlating the
	// nested calls, but i haven't figured that part out yet
	m.WithLabelValues("none").Inc()
	return db.GetUser(ctx, id)
}

func (db *Conn) GetUser(ctx context.Context, id string) (*sharedv1.User, error) {
	result := &sharedv1.User{ID: id}

	return result, db.
		QueryRowContext(ctx, selectUser, id).
		Scan(&result.Name, &result.MTime, &result.DTime)
}

func (db *Conn) AddUser(ctx context.Context, u *sharedv1.User) (string, error) {
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
			if strings.Contains(v.Error(), "users.id") {
				return db.AddUser(ctx, u) // FIXME: handle infinite recursion (unlikely as it is)
			} else if strings.Contains(v.Error(), "users.name") {
				return u.Name, UserExistsError
			}
		default:
			return u.Name, err
		}
	} else if rows, err := result.RowsAffected(); err != nil {
		return u.Name, err
	} else if rows != 1 {
		return u.Name, UserNotAddedError
	}
	return u.Name, nil
}

func (db *Conn) UpdateUser(ctx context.Context, u *sharedv1.User) error {
	curr, err := db.GetUser(ctx, u.ID)
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

func (db *Conn) DeleteUser(ctx context.Context, id string) error {
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

func hash(pass, salt string) string {
	return pass + salt
}

func generateSalt() string {
	return "salt"
}

func (db *Conn) CreateContact(ctx context.Context, id string, c *sharedv1.Contact) (string, error) {
	var err error
	if c.User, err = db.GetUser(ctx, id); err != nil {
		return "", err
	}
	return db.AddContact(ctx, c)
}
