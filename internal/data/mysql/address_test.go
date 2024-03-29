package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	sharedv1 "github.com/jsmit257/userservice/shared/v1"
	log "github.com/sirupsen/logrus"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"github.com/stretchr/testify/require"
)

func TestGetAddress(t *testing.T) {
	t.Parallel()
	l := log.WithFields(log.Fields{"app": "address_test.go", "test": "TestGetAddress"})
	tcs := map[string]struct {
		mockDB getMockDB
		addr   *sharedv1.Address
		err    error
	}{
		"happy_path": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectAddress).
					WithArgs("1").
					WillReturnRows(sqlmock.
						NewRows([]string{"id"}).
						AddRow("1"))
				return db
			},
			addr: &sharedv1.Address{
				ID: "1",
			},
		},
		"selectRow_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectAddress).
					WithArgs("1").
					WillReturnError(fmt.Errorf("some error"))
				return db
			},
			addr: &sharedv1.Address{},
			err:  fmt.Errorf("some error"),
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cid := sharedv1.CID("TestGetAddress-" + name)
			addr, err := (&Conn{tc.mockDB(), nil, l}).
				GetAddress(context.Background(), "1", cid)
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.addr, addr)
		})
	}
}

func TestAddAddress(t *testing.T) {
	t.Parallel()
	l := log.WithFields(log.Fields{"app": "address_test.go", "test": "TestAddAddress"})
	tcs := map[string]struct {
		mockDB getMockDB
		addr   *sharedv1.Address
		uuid   string
		err    error
	}{
		"happy_path": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			addr: &sharedv1.Address{},
			uuid: mockUUIDGen().String(),
		},
		"exec_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(".*").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			addr: &sharedv1.Address{},
			err:  fmt.Errorf("some error"),
		},
		"no_insert": { // how would this happen w/o an error?
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			addr: &sharedv1.Address{},
			err:  fmt.Errorf("address was not added"),
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cid := sharedv1.CID("TestGetAddress-" + name)
			uuid, err := (&Conn{tc.mockDB(), mockUUIDGen, l}).
				AddAddress(context.Background(), tc.addr, cid)
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.uuid, uuid)
		})
	}
}

func TestUpdateAddress(t *testing.T) {
	t.Parallel()
	l := log.WithFields(log.Fields{"app": "address_test.go", "test": "TestUpdateAddress"})
	tcs := map[string]struct {
		mockDB getMockDB
		addr   *sharedv1.Address
		err    error
	}{
		"happy_path": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectAddress).
					WillReturnRows(sqlmock.
						NewRows([]string{"id"}).
						AddRow("1"))
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			addr: &sharedv1.Address{},
		},
		"get_address_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectAddress).
					WillReturnError(fmt.Errorf("some error"))
				return db
			},
			addr: &sharedv1.Address{},
			err:  fmt.Errorf("some error"),
		},
		"exec_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectAddress).
					WillReturnRows(sqlmock.
						NewRows([]string{"id"}).
						AddRow("1"))
				mock.ExpectExec(".*").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			addr: &sharedv1.Address{},
			err:  fmt.Errorf("some error"),
		},
		"nothing_to_update": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectAddress).
					WillReturnRows(sqlmock.
						NewRows([]string{"id"}).
						AddRow("1"))
				return db
			},
			addr: &sharedv1.Address{ID: "1"},
		},
		"no_update": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectAddress).
					WillReturnRows(sqlmock.
						NewRows([]string{"id"}).
						AddRow("1"))
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			addr: &sharedv1.Address{ID: "not added"},
			err:  fmt.Errorf("address was not updated: '%s'", "not added"),
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cid := sharedv1.CID("TestGetAddress-" + name)
			require.Equal(t, tc.err, (&Conn{tc.mockDB(), nil, l}).
				UpdateAddress(context.Background(), tc.addr, cid))
		})
	}
}

func TestDeleteAddress(t *testing.T) {
	t.Parallel()
	l := log.WithFields(log.Fields{"app": "address_test.go", "test": "TestDeleteAddress"})
	tcs := map[string]struct {
		mockDB getMockDB
		err    error
	}{
		"happy_path": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(deleteAddress).WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
		},
		"exec_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(deleteAddress).WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
		},
		"user_not_found": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(deleteAddress).WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			err: fmt.Errorf("address could not be deleted"),
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cid := sharedv1.CID("TestDeleteAddress-" + name)
			require.Equal(
				t,
				tc.err,
				(&Conn{tc.mockDB(), nil, l}).
					DeleteAddress(context.Background(), "", cid))
		})
	}
}
