package data

import (
	"context"
	"fmt"
	"time"

	sharedv1 "github.com/jsmit257/userservice/shared/v1"
	"github.com/prometheus/client_golang/prometheus"
)

type userVec struct{ *prometheus.CounterVec }

func (u userVec) labels(l prometheus.Labels) userVec {
	u.CounterVec = u.CounterVec.MustCurryWith(l)
	return u
}

func (u userVec) done(err error) {
	if err == nil {
		u.WithLabelValues("none").Inc()
	} else {
		u.WithLabelValues(err.Error()).Inc()
	}
}
func (db *Conn) GetAllAddresses(ctx context.Context, cid sharedv1.CID) ([]sharedv1.Address, error) {

	rows, err := db.QueryContext(ctx, db.sqls["address"]["select-all"])
	if err != nil {
		return nil, err
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
			return nil, err
		}
		result = append(result, row)
	}

	return result, nil
}

func (db *Conn) GetAddress(ctx context.Context, id sharedv1.UUID, cid sharedv1.CID) (*sharedv1.Address, error) {
	result := &sharedv1.Address{}

	m := userVec{mtrcs}.labels(prometheus.Labels{"function": "GetAddress"})

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

	m.done(err)

	return result, err
}

func (db *Conn) AddAddress(ctx context.Context, addr *sharedv1.Address, cid sharedv1.CID) (sharedv1.UUID, error) {
	now := time.Now().UTC()
	addr.UUID, addr.MTime, addr.CTime =
		db.generateUUID(),
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
	if err != nil {
		return err
	} else if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows != 1 {
		return sharedv1.AddressNotUpdatedError
	}
	return nil
}
