package data

import (
	"context"
	"fmt"
	"time"

	sharedv1 "github.com/jsmit257/userservice/shared/v1"
)

func (db *Conn) GetAllAddresses(ctx context.Context, cid sharedv1.CID) ([]sharedv1.Address, error) {
	done, log := db.logging("GetAllAddresses", nil, cid)

	rows, err := db.QueryContext(ctx, db.sqls["address"]["select-all"])
	if err != nil {
		return nil, done(err, log)
	}

	result := []sharedv1.Address{}
	for rows.Next() {
		row := sharedv1.Address{}
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

func (db *Conn) GetAddress(ctx context.Context, id sharedv1.UUID, cid sharedv1.CID) (*sharedv1.Address, error) {
	done, log := db.logging("GetAddress", id, cid)

	result := &sharedv1.Address{}
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

func (db *Conn) AddAddress(ctx context.Context, addr *sharedv1.Address, cid sharedv1.CID) (sharedv1.UUID, error) {
	done, log := db.logging("AddAddress", addr, cid)

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

func (db *Conn) UpdateAddress(ctx context.Context, addr *sharedv1.Address, cid sharedv1.CID) error {
	done, log := db.logging("UpdateAddress", addr, cid)

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
			err = sharedv1.AddressNotUpdatedError
		}
	}

	return done(err, log)
}
