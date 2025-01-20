package data

import (
	"context"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jsmit257/userservice/shared/v1"
)

func (db *Conn) GetAllUsers(ctx context.Context) ([]shared.User, error) {
	done, log := db.logging("GetAllUsers", nil, ctx.Value(shared.CTXKey("cid")).(shared.CID))

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
			break
		}
		result = append(result, row)
	}

	return result, done(err, log)
}

func (db *Conn) GetUser(ctx context.Context, id shared.UUID) (*shared.User, error) {
	done, log := db.logging("GetUser", id, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	result := &shared.User{}
	err := db.
		QueryRowContext(ctx, db.sqls["user"]["select"], id).
		Scan(
			&result.UUID,
			&result.Name,
			&result.Email,
			&result.Cell,
			&result.MTime,
			&result.CTime,
			&result.DTime)

	if err != nil {
		return nil, done(err, log)
	} else if result.Contact, err = db.getContact(ctx, id); err != nil {
		result = nil
	}

	return result, done(err, log)
}

func (db *Conn) AddUser(ctx context.Context, u *shared.User) (shared.UUID, error) {
	done, log := db.logging("AddUser", u, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	now := time.Now().UTC()
	u.UUID = db.uuidgen()
	u.MTime = now
	u.CTime = now

	var rows int64
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
				return db.AddUser(ctx, u) // FIXME: handle infinite recursion (unlikely as it is)
			} else if strings.Contains(v.Message, "users.name") {
				err = shared.UserExistsError
			}
		}
	} else if rows, err = result.RowsAffected(); err == nil && rows != 1 {
		err = shared.UserNotAddedError
	}
	return u.UUID, done(err, log)
}

func (db *Conn) UpdateUser(ctx context.Context, u *shared.User) error {
	done, log := db.logging("UpdateUser", u, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	u.MTime = time.Now().UTC()
	result, err := db.ExecContext(ctx, db.sqls["user"]["update"],
		u.Name,
		u.MTime,
		u.UUID)

	if err == nil {
		var rows int64
		if rows, err = result.RowsAffected(); err == nil && rows != 1 {
			return shared.UserNotUpdatedError
		}
	}

	return done(err, log)
}

func (db *Conn) DeleteUser(ctx context.Context, id shared.UUID) error {
	done, log := db.logging("DeleteUser", id, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	result, err := db.ExecContext(ctx, db.sqls["user"]["delete"], time.Now().UTC(), id)
	if err == nil {
		var rows int64
		if rows, err = result.RowsAffected(); err == nil && rows != 1 {
			return shared.UserNotDeletedError
		}
	}

	return done(err, log)
}

func (db *Conn) CreateContact(ctx context.Context, u *shared.User, c shared.Contact) (*shared.Contact, error) {
	var err error
	done, log := db.logging("CreateContact", u, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	u.Contact, err = db.addContact(ctx, u.UUID, c)

	return u.Contact, done(err, log)
}
