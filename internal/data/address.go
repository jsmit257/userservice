package data

import (
	"context"
	"fmt"
	"time"

	sharedv1 "github.com/jsmit257/userservice/shared/v1"
)

const (
	deleteAddress = "delete from addresses where id = ?"
	insertAddress = "insert into addresses(id, street1, street2, city, state, country, zip, mtime, ctime) values(?, ?, ?, ?, ?, ?, ?, ?, ?)"
	selectAddress = "select id from addresses where id = ?"
	updateAddress = "update addresses set street1 = ?, street2 = ?, city = ?, state = ?, country = ?, zip = ?, mtime = ? where id = ?"
)

func (db *Conn) GetAddress(ctx context.Context, id sharedv1.UUID, cid sharedv1.CID) (*sharedv1.Address, error) {
	result := &sharedv1.Address{}

	return result, db.
		QueryRowContext(ctx, selectAddress, id).
		Scan(&result.UUID)
}

func (db *Conn) AddAddress(ctx context.Context, addr *sharedv1.Address, cid sharedv1.CID) (sharedv1.UUID, error) {
	now := time.Now().UTC()
	addr.UUID, addr.MTime, addr.CTime =
		db.generateUUID(),
		now,
		now

	result, err := db.ExecContext(ctx, insertAddress,
		addr.UUID,
		addr.Street1,
		addr.Street2,
		addr.City,
		addr.State,
		addr.Country,
		addr.Zip,
		addr.MTime,
		addr.CTime)
	if err != nil {
		return "", err
	} else if rows, err := result.RowsAffected(); err != nil {
		return "", err
	} else if rows != 1 {
		return "", fmt.Errorf("address was not added")
	}
	return addr.UUID, nil
}

func (db *Conn) UpdateAddress(ctx context.Context, addr *sharedv1.Address, cid sharedv1.CID) error {
	oldaddr, err := db.GetAddress(ctx, addr.UUID, cid)
	if err != nil {
		return err
	}
	// assume these aren't set in the input, even if they are, we'll ignore
	// them and override as needed
	oldaddr.MTime = addr.MTime
	oldaddr.CTime = addr.CTime
	if *addr == *oldaddr {
		return nil
	}
	// get address by id
	// if nothing except ctime/mtime is different, then bail with nil
	now := time.Now().UTC()
	result, err := db.ExecContext(ctx, updateAddress,
		addr.Street1,
		addr.Street2,
		addr.City,
		addr.State,
		addr.Country,
		addr.Zip,
		now,
		addr.UUID)
	if err != nil {
		return err
	} else if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows != 1 {
		return fmt.Errorf("address was not updated: '%s'", addr.UUID)
	}
	return nil
}

func (db *Conn) DeleteAddress(ctx context.Context, id sharedv1.UUID, cid sharedv1.CID) error {
	result, err := db.ExecContext(ctx, deleteAddress, id)
	if err != nil {
		return err
	} else if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows != 1 {
		return fmt.Errorf("address could not be deleted")
	}
	return nil
}
