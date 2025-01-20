package data

import (
	"database/sql"
	"fmt"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/jsmit257/userservice/shared/v1"
)

var (
	_addr = shared.Address{
		UUID:    "uuid",
		Street1: "street1",
		Street2: "street2",
		City:    "city",
		State:   "state",
		Country: "country",
		Zip:     "zip",
		MTime:   rightaboutnow,
		CTime:   rightaboutnow,
	}
	addrFields = row{
		"uuid",
		"street1",
		"street2",
		"city",
		"state",
		"country",
		"zip",
		"mtime",
		"ctime",
	}
	addrValues = []values{{
		_addr.UUID,
		_addr.Street1,
		_addr.Street2,
		_addr.City,
		_addr.State,
		_addr.Country,
		_addr.Zip,
		_addr.MTime,
		_addr.CTime,
	}}
)

func TestGetAllAddresses(t *testing.T) {
	t.Parallel()

	l := testLogger(t, logrus.Fields{"app": "address_test.go", "test": "TestGetAllAddresses"})

	tcs := map[string]struct {
		mockDB getMockDB
		addr   []shared.Address
		err    error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(addrFields).
						AddRow(addrValues[0]...).
						AddRow(addrValues[0]...))
				return db
			},
			addr: []shared.Address{_addr, _addr},
		},
		"db_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cid := shared.CID("TestGetAddress-" + name)
			addr, err := (&Conn{
				tc.mockDB(sqlmock.New()),
				nil,
				mockSqls(),
				&senderMock{},
				l,
				testmetrics,
			}).GetAllAddresses(mockContext(cid))
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.addr, addr)
		})
	}
}

func TestGetAddress(t *testing.T) {
	t.Parallel()

	l := testLogger(t, logrus.Fields{"app": "address_test.go", "test": "TestGetAddress"})

	tcs := map[string]struct {
		mockDB getMockDB
		result *shared.Address
		err    error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(addrFields).
						AddRow(addrValues[0]...))
				return db
			},
			result: &_addr,
		},
		"query_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cid := shared.CID("TestGetAddress-" + name)
			addr, err := (&Conn{
				tc.mockDB(sqlmock.New()),
				nil,
				mockSqls(),
				&senderMock{},
				l,
				testmetrics,
			}).GetAddress(mockContext(cid), "1")
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.result, addr)
		})
	}
}

func TestAddAddress(t *testing.T) {
	t.Parallel()

	l := testLogger(t, logrus.Fields{"app": "address_test.go", "test": "TestAddAddress"})

	tcs := map[string]struct {
		mockDB getMockDB
		addr   *shared.Address
		result shared.UUID
		err    error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			addr:   &shared.Address{},
			result: mockUUIDGen(),
		},
		"exec_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec(".*").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			addr:   &shared.Address{},
			err:    fmt.Errorf("some error"),
			result: mockUUIDGen(),
		},
		"no_insert": { // how would this happen w/o an error?
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			addr:   &shared.Address{},
			err:    fmt.Errorf("address was not added"),
			result: mockUUIDGen(),
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cid := shared.CID("TestGetAddress-" + name)
			uuid, err := (&Conn{
				tc.mockDB(sqlmock.New()),
				mockUUIDGen,
				mockSqls(),
				&senderMock{},
				l,
				testmetrics,
			}).AddAddress(mockContext(cid), tc.addr)
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.result, uuid)
		})
	}
}

func TestUpdateAddress(t *testing.T) {
	t.Parallel()

	l := testLogger(t, logrus.Fields{"app": "address_test.go", "test": "TestUpdateAddress"})

	tcs := map[string]struct {
		mockDB getMockDB
		addr   *shared.Address
		err    error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			addr: &shared.Address{},
		},
		"no_update": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			addr: &shared.Address{UUID: "not added"},
			err:  shared.AddressNotUpdatedError,
		},
		"exec_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec(".*").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			addr: &shared.Address{},
			err:  fmt.Errorf("some error"),
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cid := shared.CID("TestGetAddress-" + name)
			require.Equal(t, tc.err, (&Conn{
				tc.mockDB(sqlmock.New()),
				nil,
				mockSqls(),
				&senderMock{},
				l,
				testmetrics,
			}).UpdateAddress(mockContext(cid), tc.addr))
		})
	}
}
