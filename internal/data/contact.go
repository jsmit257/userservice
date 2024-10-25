package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sharedv1 "github.com/jsmit257/userservice/shared/v1"
)

const (
	deleteContact = "delete from contacts where id = ?"
	insertContact = "insert into contacts(id, user_id, lastname, firstname, mtime, ctime) values(?, ?, ?, ?, ?, ?)"
	selectContact = "select id, firstname, lastname, billto_id, sendto_id from contacts where user_id = ?"
	updateContact = "update contacts set lastname = ?, firstname = ?, mtime = ? where id = ?"
)

func (db *Conn) GetContact(ctx context.Context, id sharedv1.UUID, cid sharedv1.CID) (*sharedv1.Contact, error) {
	var billto, shipto *sharedv1.UUID

	result := &sharedv1.Contact{}

	contactRow := db.QueryRowContext(ctx, selectContact, id)

	err := contactRow.Scan(
		&result.UUID,
		&result.FirstName,
		&result.LastName,
		&billto,
		&shipto)
	if err != nil {
		return nil, err
	}

	if billto != nil {
		if result.BillTo, err = db.GetAddress(ctx, *billto, cid); err != nil {
			return nil, err
		}
	}

	if shipto != nil {
		if result.ShipTo, err = db.GetAddress(ctx, *shipto, cid); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (db *Conn) AddContact(ctx context.Context, userid sharedv1.UUID, c *sharedv1.Contact, cid sharedv1.CID) (sharedv1.UUID, error) {
	if userid == "" {
		return "", fmt.Errorf("contacts require a valid user")
	}

	id := db.generateUUID()
	now := time.Now().UTC()

	result, err := db.ExecContext(ctx, insertContact,
		id,
		c.LastName,
		c.FirstName,
		userid,
		now,
		now)
	if err != nil {
		return "", err
	} else if rows, err := result.RowsAffected(); err != nil {
		return "", err
	} else if rows != 1 {
		return "", sql.ErrNoRows
	}
	return id, nil
}

func (db *Conn) UpdateContact(ctx context.Context, c *sharedv1.Contact, cid sharedv1.CID) error {
	if result, err := db.ExecContext(ctx, updateContact, c.LastName, c.FirstName, time.Now().UTC(), c.UUID); err != nil {
		return err
	} else if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows != 1 {
		return fmt.Errorf("contact was not updated: '%s'", c.UUID)
	}
	return nil
}

func (db *Conn) DeleteContact(ctx context.Context, id sharedv1.UUID, cid sharedv1.CID) error {
	result, err := db.ExecContext(ctx, deleteContact, id)
	if err != nil {
		return err
	} else if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows != 1 {
		return sql.ErrNoRows
	}
	return nil
}
