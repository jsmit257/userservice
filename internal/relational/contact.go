package data

import (
	"context"
	"fmt"
	"time"

	"github.com/jsmit257/userservice/shared/v1"
)

func (db *Conn) getContact(ctx context.Context, id shared.UUID, cid shared.CID) (*shared.Contact, error) {
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

func (db *Conn) addContact(ctx context.Context, userid shared.UUID, c shared.Contact, _ shared.CID) (*shared.Contact, error) {
	if userid == "" {
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

	result, err := db.ExecContext(ctx, db.sqls["contact"]["insert"],
		c.FirstName,
		c.LastName,
		&billto,
		&shipto,
		c.MTime,
		c.CTime,
		userid)
	if err != nil {
		return nil, err
	} else if rows, err := result.RowsAffected(); err != nil {
		return nil, err
	} else if rows != 1 {
		return nil, fmt.Errorf("contact was not inserted: '%s'", userid)
	}

	return &c, nil
}

func (db *Conn) UpdateContact(ctx context.Context, userid shared.UUID, c *shared.Contact, cid shared.CID) error {
	billto := (*shared.UUID)(nil)
	if c.BillTo != nil {
		billto = &c.BillTo.UUID
	}
	shipto := (*shared.UUID)(nil)
	if c.ShipTo != nil {
		shipto = &c.ShipTo.UUID
	}

	if result, err := db.ExecContext(ctx, db.sqls["contact"]["update"],
		c.FirstName,
		c.LastName,
		billto,
		shipto,
		time.Now().UTC(),
		userid,
	); err != nil {
		return err
	} else if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows != 1 {
		return fmt.Errorf("contact was not updated: '%s'", userid)
	}

	return nil
}
