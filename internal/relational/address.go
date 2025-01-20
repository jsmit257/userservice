package data

import (
	"context"
	"fmt"
	"time"

	"github.com/jsmit257/userservice/shared/v1"
)

func (db *Conn) GetAllAddresses(ctx context.Context) ([]shared.Address, error) {
	done, log := db.logging("GetAllAddresses", nil, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	rows, err := db.QueryContext(ctx, db.sqls["address"]["select-all"])
	if err != nil {
		return nil, done(err, log)
	}

	result := []shared.Address{}
	for rows.Next() {
		row := shared.Address{}
		if err = rows.Scan(
			&row.UUID,
			&row.Street1,
			&row.Street2,
			&row.City,
			&row.State,
			&row.Country,
			&row.Zip,
			&row.MTime,
			&row.CTime,
		); err != nil {
			break
		}
		result = append(result, row)
	}

	return result, done(err, log)
}

func (db *Conn) GetAddress(ctx context.Context, id shared.UUID) (*shared.Address, error) {
	done, log := db.logging("GetAddress", id, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	result := &shared.Address{}
	err := db.
		QueryRowContext(ctx, db.sqls["address"]["select"], id).
		Scan(
			&result.UUID,
			&result.Street1,
			&result.Street2,
			&result.City,
			&result.State,
			&result.Country,
			&result.Zip,
			&result.MTime,
			&result.CTime)

	if err != nil {
		result = nil
	}

	return result, done(err, log)
}

func (db *Conn) AddAddress(ctx context.Context, addr *shared.Address) (shared.UUID, error) {
	done, log := db.logging("AddAddress", addr, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	now := time.Now().UTC()
	addr.UUID, addr.MTime, addr.CTime =
		db.uuidgen(),
		now,
		now

	result, err := db.ExecContext(ctx, db.sqls["address"]["insert"],
		addr.UUID,
		addr.Street1,
		addr.Street2,
		addr.City,
		addr.State,
		addr.Country,
		addr.Zip,
		addr.MTime,
		addr.CTime)

	if err == nil {
		var rows int64
		if rows, err = result.RowsAffected(); err == nil && rows != 1 {
			err = fmt.Errorf("address was not added")
		}
	}

	return addr.UUID, done(err, log)
}

func (db *Conn) UpdateAddress(ctx context.Context, addr *shared.Address) error {
	done, log := db.logging("UpdateAddress", addr, ctx.Value(shared.CTXKey("cid")).(shared.CID))

	now := time.Now().UTC()

	result, err := db.ExecContext(ctx, db.sqls["address"]["update"],
		addr.Street1,
		addr.Street2,
		addr.City,
		addr.State,
		addr.Country,
		addr.Zip,
		now,
		addr.UUID)

	var rows int64
	if err == nil {
		if rows, err = result.RowsAffected(); err == nil && rows != 1 {
			err = shared.AddressNotUpdatedError
		}
	}

	return done(err, log)
}
