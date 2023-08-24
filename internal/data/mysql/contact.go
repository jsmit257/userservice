package mysql

import (
	"context"
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

func (db *Conn) GetContact(ctx context.Context, userID string, cid string) (*sharedv1.Contact, error) {
	var billto, shipto string
	result := &sharedv1.Contact{}

	contactRow := db.QueryRowContext(ctx, selectContact, userID)

	err := contactRow.Scan(&result.ID, &result.FirstName, &result.LastName, &billto, &shipto)
	if err != nil {
		return nil, err
	} else if result.User, err = db.GetUser(ctx, userID, cid); err != nil {
		return nil, err
	}

	if billto != "" {
		if result.BillTo, err = db.GetAddress(ctx, billto); err != nil {
			return nil, err
		}
	}

	if shipto != "" {
		if result.ShipTo, err = db.GetAddress(ctx, shipto); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (db *Conn) AddContact(ctx context.Context, c *sharedv1.Contact, cid string) (string, error) {
	if c.User == nil {
		return "", fmt.Errorf("contact requires a user")
	} else if c.User.ID == "" {
		return "", fmt.Errorf("contact requires a valid user")
	}
	id, now := db.generateUUID(), time.Now().UTC()
	result, err := db.ExecContext(ctx, insertContact, id, c.User.ID, c.LastName, c.FirstName, now, now)
	if err != nil {
		return "", err
	} else if rows, err := result.RowsAffected(); err != nil {
		return "", err
	} else if rows != 1 {
		return "", fmt.Errorf("contact was not inserted: '%s'", id)
	}
	return id.String(), nil
}

func (db *Conn) UpdateContact(ctx context.Context, c *sharedv1.Contact, cid string) error {
	// disregard values for Contact.{User,BillTo,ShipTo}; those fields and their attendant objects are managed elsewhere
	if result, err := db.ExecContext(ctx, updateContact, c.LastName, c.FirstName, time.Now().UTC(), c.ID); err != nil {
		return err
	} else if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows != 1 {
		return fmt.Errorf("contact was not updated: '%s'", c.ID)
	}
	return nil
}

func (db *Conn) DeleteContact(ctx context.Context, id string, cid string) error {
	result, err := db.ExecContext(ctx, deleteContact, id)
	if err != nil {
		return err
	} else if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows != 1 {
		return fmt.Errorf("contact could not be deleted")
	}
	return nil
}
