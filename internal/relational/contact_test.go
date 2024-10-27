package data

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	sharedv1 "github.com/jsmit257/userservice/shared/v1"
	log "github.com/sirupsen/logrus"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/stretchr/testify/require"
)

var (
	_con = sharedv1.Contact{
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
		result *sharedv1.Contact
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
			result: func(c sharedv1.Contact) *sharedv1.Contact {
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
			result: func(c sharedv1.Contact) *sharedv1.Contact {
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
			result: func(c sharedv1.Contact) *sharedv1.Contact {
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
			err: fmt.Errorf("some error"),
		},
		"selectshipto_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(conFields).
						AddRow(conValues...))
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(addrFields).
						AddRow(addrValues[0]...))
				mock.ExpectQuery("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
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
			}).getContact(context.Background(), "1", sharedv1.CID("TestGetContact-"+name))
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
		userid  sharedv1.UUID
		contact sharedv1.Contact
		result  *sharedv1.Contact
		err     error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			userid:  "1",
			contact: sharedv1.Contact{},
			result: &sharedv1.Contact{
				MTime: rightaboutnow,
				CTime: rightaboutnow,
			},
		},
		"nil_user_id": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				return db
			},
			contact: sharedv1.Contact{},
			err:     fmt.Errorf("contacts require a valid user"),
		},
		"exec_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			userid:  "1",
			contact: sharedv1.Contact{},
			err:     fmt.Errorf("some error"),
		},
		"no_update": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			userid:  "1",
			contact: sharedv1.Contact{},
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
			}).addContact(context.Background(), tc.userid, tc.contact, sharedv1.CID("TestAddContact-"+name))
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
		userid  sharedv1.UUID
		contact *sharedv1.Contact
		err     error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			contact: &sharedv1.Contact{},
		},
		"exec_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			contact: &sharedv1.Contact{},
			err:     fmt.Errorf("some error"),
		},
		"update_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			userid:  "update fails",
			contact: &sharedv1.Contact{},
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
			}).UpdateContact(context.Background(), tc.userid, tc.contact, sharedv1.CID("TestUpdateContact-"+name)))
		})
	}
}
