package data

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/jsmit257/userservice/shared/v1"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/stretchr/testify/require"
)

var (
	_con = shared.Contact{
		FirstName: "firstname",
		LastName:  "lastname",
		MTime:     rightaboutnow,
		CTime:     rightaboutnow,
	}
	conFields = row{
		"firstname",
		"lastname",
		"billto_id",
		"shipto_id",
		"mtime",
		"ctime",
	}
	conValues = values{
		_con.FirstName,
		_con.LastName,
		"nil",
		"nil",
		_con.MTime,
		_con.CTime,
	}
)

func TestGetContact(t *testing.T) {
	t.Parallel()

	l := testLogger(t, log.Fields{"app": "contact_test.go", "test": "TestGetContact"})

	tcs := map[string]struct {
		mockDB getMockDB
		result *shared.Contact
		err    error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(conFields).
						AddRow(conValues...))
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(addrFields).
						AddRow(addrValues[0]...))
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(addrFields).
						AddRow(addrValues[0]...))
				return db
			},
			result: func(c shared.Contact) *shared.Contact {
				c.BillTo = &_addr
				c.ShipTo = &_addr

				return &c
			}(_con),
		},
		"no_billto": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(conFields).
						AddRow(conValues.replace(repl{2, nil})...))
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(addrFields).
						AddRow(addrValues[0]...))
				return db
			},
			result: func(c shared.Contact) *shared.Contact {
				c.ShipTo = &_addr

				return &c
			}(_con),
		},
		"no_shipto": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(conFields).
						AddRow(conValues.replace(repl{3, nil})...))
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(addrFields).
						AddRow(addrValues[0]...))
				return db
			},
			result: func(c shared.Contact) *shared.Contact {
				c.BillTo = &_addr

				return &c
			}(_con),
		},
		"selectbillto_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(conFields).
						AddRow(conValues...))
				mock.ExpectQuery("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err:    fmt.Errorf("some error"),
			result: &_con,
		},
		"selectshipto_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(conFields).
						AddRow(conValues.replace(repl{2, nil})...))
				mock.ExpectQuery("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err:    fmt.Errorf("some error"),
			result: &_con,
		},
		"contact_fails": {
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
			contact, err := (&Conn{
				tc.mockDB(sqlmock.New()),
				nil,
				mockSqls(),
				l,
				testmetrics.MustCurryWith(prometheus.Labels{"db": "test db"}),
			}).getContact(mockContext(shared.CID("TestGetContact-"+name)), "1")
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.result, contact)
		})
	}
}

func TestAddContact(t *testing.T) {
	t.Parallel()

	l := testLogger(t, log.Fields{"app": "contact_test.go", "test": "TestAddContact"})

	tcs := map[string]struct {
		mockDB  getMockDB
		userid  shared.UUID
		contact shared.Contact
		result  *shared.Contact
		err     error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			userid:  "1",
			contact: shared.Contact{},
			result: &shared.Contact{
				MTime: rightaboutnow,
				CTime: rightaboutnow,
			},
		},
		"nil_user_id": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				return db
			},
			contact: shared.Contact{},
			err:     fmt.Errorf("contacts require a valid user"),
		},
		"exec_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			userid:  "1",
			contact: shared.Contact{},
			err:     fmt.Errorf("some error"),
		},
		"no_update": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			userid:  "1",
			contact: shared.Contact{},
			err:     fmt.Errorf("contact was not inserted: '1'"),
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := (&Conn{
				tc.mockDB(sqlmock.New()),
				mockUUIDGen,
				mockSqls(),
				l,
				testmetrics.MustCurryWith(prometheus.Labels{"db": "test db"}),
			}).addContact(mockContext(shared.CID("TestAddContact-"+name)), tc.userid, tc.contact)
			require.Equal(t, tc.err, err)
			// require.Equal(t, tc.result, result) // there's no way to match mtime/ctime
		})
	}
}

func TestUpdateContact(t *testing.T) {
	t.Parallel()

	l := testLogger(t, log.Fields{"app": "contact_test.go", "test": "TestUpdateContact"})

	tcs := map[string]struct {
		mockDB  getMockDB
		userid  shared.UUID
		contact *shared.Contact
		err     error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			contact: &shared.Contact{},
		},
		"exec_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			contact: &shared.Contact{},
			err:     fmt.Errorf("some error"),
		},
		"update_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			userid:  "update fails",
			contact: &shared.Contact{},
			err:     fmt.Errorf("contact was not updated: 'update fails'"),
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.err, (&Conn{
				tc.mockDB(sqlmock.New()),
				nil,
				mockSqls(),
				l,
				testmetrics.MustCurryWith(prometheus.Labels{"db": "test db"}),
			}).UpdateContact(mockContext(shared.CID("TestUpdateContact-"+name)), tc.userid, tc.contact))
		})
	}
}
