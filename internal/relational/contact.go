package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jsmit257/userservice/shared/v1"
)

func (db *Conn) getContact(ctx context.Context, id shared.UUID, cid shared.CID) (*shared.Contact, error) {
	done, log := db.logging("getContact", id, cid)

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
			result.BillTo, err = db.GetAddress(ctx, *billto, cid)
		}

		if err == nil && shipto != nil {
			result.ShipTo, err = db.GetAddress(ctx, *shipto, cid)
		}
	} else {
		result = nil
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
		}
	}

	return result, done(err, log)
}

func (db *Conn) addContact(ctx context.Context, id shared.UUID, c shared.Contact, cid shared.CID) (*shared.Contact, error) {
	var err error
	done, log := db.logging("addContact", id, cid)

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

func (db *Conn) UpdateContact(ctx context.Context, id shared.UUID, c *shared.Contact, cid shared.CID) error {
	var err error
	done, log := db.logging("UpdateContact", id, cid)

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
