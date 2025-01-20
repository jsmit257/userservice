package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jsmit257/userservice/shared/v1"
)

func (db *Conn) getContact(ctx context.Context, id shared.UUID) (*shared.Contact, error) {
	done, log := db.logging("getContact", id, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	var billto, shipto *shared.UUID
	result := &shared.Contact{}
	contactRow := db.QueryRowContext(ctx, db.sqls["contact"]["select"], id)
	err := contactRow.Scan(
		&result.FirstName,
		&result.LastName,
		&billto,
		&shipto,
		&result.MTime,
		&result.CTime)

	if err == nil {
		if billto != nil {
			result.BillTo, err = db.GetAddress(ctx, *billto)
		}

		if err == nil && shipto != nil {
			result.ShipTo, err = db.GetAddress(ctx, *shipto)
		}
	} else {
		result = nil
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
		}
	}

	return result, done(err, log)
}

func (db *Conn) addContact(ctx context.Context, id shared.UUID, c shared.Contact) (*shared.Contact, error) {
	var err error
	done, log := db.logging("addContact", id, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	if id == "" {
		return nil, fmt.Errorf("contacts require a valid user")
	}

	c.CTime = time.Now().UTC()
	c.MTime = c.CTime

	billto := (*shared.UUID)(nil)
	if c.BillTo != nil {
		billto = &c.BillTo.UUID
	}
	shipto := (*shared.UUID)(nil)
	if c.ShipTo != nil {
		shipto = &c.ShipTo.UUID
	}

	var rows int64
	result, err := db.ExecContext(ctx, db.sqls["contact"]["insert"],
		c.FirstName,
		c.LastName,
		&billto,
		&shipto,
		c.MTime,
		c.CTime,
		id)
	if err == nil {
		if rows, err = result.RowsAffected(); err == nil && rows != 1 {
			err = fmt.Errorf("contact was not inserted: '%s'", id)
		}
	}

	return &c, done(err, log)
}

func (db *Conn) UpdateContact(ctx context.Context, id shared.UUID, c *shared.Contact) error {
	var err error
	done, log := db.logging("UpdateContact", id, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	billto := (*shared.UUID)(nil)
	if c.BillTo != nil {
		billto = &c.BillTo.UUID
	}
	shipto := (*shared.UUID)(nil)
	if c.ShipTo != nil {
		shipto = &c.ShipTo.UUID
	}

	var rows int64
	result, err := db.ExecContext(ctx, db.sqls["contact"]["update"],
		c.FirstName,
		c.LastName,
		billto,
		shipto,
		time.Now().UTC(),
		id,
	)
	if err == nil {
		if rows, err = result.RowsAffected(); err == nil && rows != 1 {
			err = fmt.Errorf("contact was not updated: '%s'", id)
		}
	}

	return done(err, log)
}
