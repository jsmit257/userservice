package data

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/jsmit257/userservice/shared/v1"
)

var (
	_user = shared.User{
		UUID:  "uuid",
		Name:  "username",
		MTime: rightaboutnow,
		CTime: rightaboutnow,
	}
	userFields = row{"uuid", "name", "mtime", "ctime", "dtime"}
	userValues = values{
		_user.UUID,
		_user.Name,
		_user.MTime,
		_user.CTime,
		_user.DTime,
	}
)

func TestGetAllUsers(t *testing.T) {
	t.Parallel()

	l := testLogger(t, log.Fields{"app": "address_test.go", "test": "TestGetAddress"})

	tcs := map[string]struct {
		mockDB getMockDB
		result []shared.User
		err    error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(userFields).
						AddRow(userValues...).
						AddRow(userValues...))
				return db
			},
			result: []shared.User{_user, _user},
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
			result, err := (&Conn{
				tc.mockDB(sqlmock.New()),
				nil,
				mockSqls(),
				l,
			}).GetAllUsers(context.Background(), cid)
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.result, result)
		})
	}
}

func TestGetUser(t *testing.T) {
	t.Parallel()

	l := testLogger(t, log.Fields{"app": "user_test.go", "test": "TestGetUser"})

	tcs := map[string]struct {
		mockDB getMockDB
		user   *shared.User
		err    error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(userFields).
						AddRow(userValues...))
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(conFields).
						AddRow(conValues.nil(2, 3)...))
				return db
			},
			user: func(u shared.User) *shared.User {
				u.Contact = &shared.Contact{
					FirstName: _con.FirstName,
					LastName:  _con.LastName,
					MTime:     _con.MTime,
					CTime:     _con.CTime,
				}

				return &u
			}(_user),
		},
		"contact_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(userFields).
						AddRow(userValues...))
				mock.ExpectQuery("").
					WillReturnError(fmt.Errorf("some error"))
				return db
			},
			// user: &_user,
			err: fmt.Errorf("some error"),
		},
		"happy_path_no_contact": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(userFields).
						AddRow(userValues...))
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(conFields))
				return db
			},
			user: &_user,
		},
		"query_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			// user: &shared.User{},
			err: fmt.Errorf("some error"),
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// if name != "happy_path" {
			// 	return
			// }
			user, err := (&Conn{
				tc.mockDB(sqlmock.New()),
				nil,
				mockSqls(),
				l,
			}).GetUser(context.Background(), "1", shared.CID("TestGetUser-"+name))
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.user, user)
		})
	}
}

func TestAddUser(t *testing.T) {
	t.Parallel()

	l := testLogger(t, log.Fields{"app": "user_test.go", "test": "TestAddUser"})

	tcs := map[string]struct {
		mockDB getMockDB
		user   *shared.User
		result shared.UUID
		err    error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			user:   &shared.User{Name: "username"},
			result: mockUUIDGen(),
		},
		"exec_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec(".*").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			user:   &shared.User{Name: "username"},
			result: mockUUIDGen(),
			err:    fmt.Errorf("some error"),
		},
		"no_insert": { // how would this happen w/o an error?
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			user:   &shared.User{Name: "username"},
			result: mockUUIDGen(),
			err:    shared.UserNotAddedError,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := (&Conn{
				tc.mockDB(sqlmock.New()),
				mockUUIDGen,
				mockSqls(),
				l,
			}).AddUser(context.Background(), tc.user, shared.CID("TestAddUser-"+name))
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.result, result)
		})
	}
}

func TestUpdateUser(t *testing.T) {
	t.Parallel()
	l := testLogger(t, log.Fields{"app": "user_test.go", "test": "TestUpdateUser"})
	tcs := map[string]struct {
		mockDB getMockDB
		user   *shared.User
		err    error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			user: &shared.User{UUID: "1", Name: "new username", MTime: rightaboutnow},
		},
		"exec_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec(".*").WillReturnError(fmt.Errorf("exec fails"))
				return db
			},
			user: &shared.User{UUID: "1", Name: "exec fails", MTime: rightaboutnow},
			err:  fmt.Errorf("couldn't update user: %w", fmt.Errorf("exec fails")),
		},
		"query_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			user: &shared.User{UUID: "1", Name: "update fails", MTime: rightaboutnow},
			err:  fmt.Errorf("user was not updated"),
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
			}).UpdateUser(context.Background(), tc.user, shared.CID("TestUpdateUser-"+name)))
		})
	}
}

func TestDeleteUser(t *testing.T) {
	t.Parallel()
	l := testLogger(t, log.Fields{"app": "user_test.go", "test": "TestDeleteUser"})
	tcs := map[string]struct {
		mockDB getMockDB
		err    error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
		},
		"exec_fails": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
		},
		// "get_rows_affected_fails": {},
		"user_not_found": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			err: shared.UserNotDeletedError,
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
			}).DeleteUser(context.Background(), "1", shared.CID("TestDeleteUser-"+name)))
		})
	}
}

func TestCreateContact(t *testing.T) {
	t.Parallel()

	l := testLogger(t, log.Fields{"app": "user_test.go", "test": "TestCreateContact"})

	tcs := map[string]struct {
		mockDB  getMockDB
		user    shared.User
		contact shared.Contact
		result  *shared.Contact
		err     error
	}{
		"happy_path": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			user:    shared.User{UUID: "1"},
			contact: shared.Contact{},
			result:  &shared.Contact{},
		},
		"missing_userid": {
			mockDB: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			user: shared.User{},
			err:  fmt.Errorf(("contacts require a valid user")),
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := (&Conn{
				tc.mockDB(sqlmock.New()),
				mockUUIDGen,
				mockSqls(),
				l,
			}).CreateContact(context.Background(), &tc.user, tc.contact, shared.CID("TestCreateContact-"+name))

			if result != nil {
				result.MTime = time.Time{}
				result.CTime = time.Time{}
			}

			require.Equal(t, tc.err, err)
			require.Equal(t, tc.result, result)
		})
	}
}
