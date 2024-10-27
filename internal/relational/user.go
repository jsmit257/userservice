package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jsmit257/userservice/shared/v1"
)

func (db *Conn) GetAllUsers(ctx context.Context, cid shared.CID) ([]shared.User, error) {

	rows, err := db.QueryContext(ctx, db.sqls["user"]["select-all"])
	if err != nil {
		return nil, err
	}

	result := []shared.User{}
	for rows.Next() {
		row := shared.User{}
		if err = rows.Scan(
			&row.UUID,
			&row.Name,
			&row.MTime,
			&row.CTime,
			&row.DTime,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, nil
}

func (db *Conn) GetUser(ctx context.Context, id shared.UUID, cid shared.CID) (*shared.User, error) {
	result := &shared.User{}

	err := db.
		QueryRowContext(ctx, db.sqls["user"]["select"], id).
		Scan(
			&result.UUID,
			&result.Name,
			&result.MTime,
			&result.CTime,
			&result.DTime)

	if err != nil {
		return nil, err
	}

	if result.Contact, err = db.getContact(ctx, id, cid); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return result, nil
}

func (db *Conn) AddUser(ctx context.Context, u *shared.User, cid shared.CID) (shared.UUID, error) {
	// salt := generateSalt()
	now := time.Now().UTC()
	u.UUID = db.generateUUID()
	u.MTime = now
	u.CTime = now

	result, err := db.ExecContext(ctx, db.sqls["user"]["insert"],
		u.UUID,
		u.Name,
		"", //hash("password", salt),
		"", //salt,
		now,
		now)
	if err != nil {
		switch v := err.(type) {
		case *mysql.MySQLError:
			if strings.Contains(v.Message, "users.PRIMARY") {
				return db.AddUser(ctx, u, cid) // FIXME: handle infinite recursion (unlikely as it is)
			} else if strings.Contains(v.Message, "users.name") {
				return "", shared.UserExistsError
			}
			return "", v
		default:
			return "", err
		}
	} else if rows, err := result.RowsAffected(); err != nil {
		return "", err
	} else if rows != 1 {
		return "", shared.UserNotAddedError
	}
	return u.UUID, nil
}

func (db *Conn) UpdateUser(ctx context.Context, u *shared.User, cid shared.CID) error {
	u.MTime = time.Now().UTC()

	if result, err := db.ExecContext(ctx, db.sqls["user"]["update"],
		u.Name,
		u.MTime,
		u.UUID,
	); err != nil {
		return fmt.Errorf("couldn't update user: %w", err)
	} else if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows != 1 {
		return shared.UserNotUpdatedError
	}

	return nil
}

func (db *Conn) DeleteUser(ctx context.Context, id shared.UUID, cid shared.CID) error {
	result, err := db.ExecContext(ctx, db.sqls["user"]["delete"], time.Now().UTC(), id)
	if err != nil {
		return err
	} else if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows != 1 {
		return shared.UserNotDeletedError
	}
	return nil
}

func (db *Conn) CreateContact(ctx context.Context, u *shared.User, c shared.Contact, cid shared.CID) (*shared.Contact, error) {
	var err error

	if u.Contact, err = db.addContact(ctx, u.UUID, c, cid); err != nil {
		return nil, err
	}

	return u.Contact, nil
}
