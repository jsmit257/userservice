package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	sharedv1 "github.com/jsmit257/userservice/shared/v1"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"github.com/stretchr/testify/require"
)

var userMTime = time.Now()

func TestBasicAuth(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		mockDB getMockDB
		login  *sharedv1.BasicAuth
		user   *sharedv1.User
		err    error
	}{
		"happy_path": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectBasicAuth).
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "password", "salt"}).
						AddRow("1", hash("pass", "salt"), "salt"))
				mock.ExpectQuery(selectUser).
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "name", "mtime"}).
						AddRow("1", "foo", userMTime))
				return db
			},
			login: &sharedv1.BasicAuth{
				Name: "foobar",
				Pass: "pass",
			},
			user: &sharedv1.User{
				ID:    "1",
				Name:  "foo",
				MTime: userMTime,
			},
		},
		"selectRow_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectBasicAuth).WithArgs("foobar").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			login: &sharedv1.BasicAuth{
				Name: "foobar",
				Pass: "pass",
			},
			err: fmt.Errorf("bad username or password"),
		},
		"authn_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectBasicAuth).
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "password", "salt"}).
						AddRow("1", "bogus hash", "salt"))
				return db
			},
			login: &sharedv1.BasicAuth{
				Name: "foobar",
				Pass: "pass",
			},
			err: fmt.Errorf("bad username or password"),
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			user, err := (&Conn{tc.mockDB(), nil}).BasicAuth(context.Background(), tc.login)
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.user, user)
		})
	}
}

func TestGetUser(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		mockDB getMockDB
		user   *sharedv1.User
		err    error
	}{
		"happy_path": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectUser).
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "name", "mtime"}).
						AddRow("1", "foo", userMTime))
				return db
			},
			user: &sharedv1.User{
				ID:    "1",
				Name:  "foo",
				MTime: userMTime,
			},
		},
		"selectRow_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectUser).WillReturnError(fmt.Errorf("some error"))
				return db
			},
			user: &sharedv1.User{},
			err:  fmt.Errorf("some error"),
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			user, err := (&Conn{tc.mockDB(), nil}).GetUser(context.Background(), "1")
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.user, user)
		})
	}
}

func TestAddUser(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		mockDB getMockDB
		user   *sharedv1.User
		result string
		err    error
	}{
		"happy_path": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			user:   &sharedv1.User{Name: "username"},
			result: "username",
		},
		"exec_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(".*").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			user:   &sharedv1.User{Name: "username"},
			result: "username",
			err:    fmt.Errorf("some error"),
		},
		"no_insert": { // how would this happen w/o an error?
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			user:   &sharedv1.User{Name: "username"},
			result: "username",
			err:    UserNotAddedError,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := (&Conn{tc.mockDB(), mockUUIDGen}).AddUser(context.Background(), tc.user)
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.result, result)
		})
	}
}

func TestUpdateUser(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		mockDB getMockDB
		user   *sharedv1.User
		err    error
	}{
		"happy_path": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectUser).
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "name", "mtime"}).
						AddRow("1", "old username", userMTime))
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			user: &sharedv1.User{ID: "1", Name: "new username", MTime: userMTime},
		},
		"get_user_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectUser).
					WillReturnError(fmt.Errorf("get user fails"))
				return db
			},
			user: &sharedv1.User{ID: "1", Name: "get user fails", MTime: userMTime},
			err:  fmt.Errorf("failed to fetch user: '%s' %w", "1", fmt.Errorf("get user fails")),
		},
		"nothing_to_update": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectUser).
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "name", "mtime"}).
						AddRow("1", "nothing to update", userMTime))
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			user: &sharedv1.User{ID: "1", Name: "nothing to update", MTime: userMTime},
		},
		"exec_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectUser).
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "name", "mtime"}).
						AddRow("1", "old exec fails", userMTime))
				mock.ExpectExec(".*").WillReturnError(fmt.Errorf("exec fails"))
				return db
			},
			user: &sharedv1.User{ID: "1", Name: "exec fails", MTime: userMTime},
			err:  fmt.Errorf("couldn't update user: '1', %w", fmt.Errorf("exec fails")),
		},
		"update_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectUser).
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "name", "mtime"}).
						AddRow("1", "old update fails", userMTime))
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 12))
				return db
			},
			user: &sharedv1.User{ID: "1", Name: "update fails", MTime: userMTime},
			err:  fmt.Errorf("user was not updated: '1'"),
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.err, (&Conn{tc.mockDB(), nil}).UpdateUser(context.Background(), tc.user))
		})
	}
}

func TestDeleteUser(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		mockDB getMockDB
		err    error
	}{
		"happy_path": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(deleteUser).WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
		},
		"exec_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(deleteUser).WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
		},
		"user_not_found": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(deleteUser).WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			err: fmt.Errorf("user could not be deleted"),
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.err, (&Conn{tc.mockDB(), nil}).DeleteUser(context.Background(), "1"))
		})
	}
}

func TestCreateContact(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		mockDB  getMockDB
		contact *sharedv1.Contact
		result  string
		err     error
	}{
		"happy_path": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectUser).
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "name", "mtime"}).
						AddRow("1", "foo", userMTime))
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			contact: &sharedv1.Contact{},
			result:  mockUUIDGen().String(),
		},
		"get_user_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectUser).
					WillReturnError(fmt.Errorf("some error"))
				return db
			},
			contact: &sharedv1.Contact{},
			err:     fmt.Errorf(("some error")),
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := (&Conn{tc.mockDB(), mockUUIDGen}).CreateContact(context.Background(), "1", tc.contact)
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.result, result)
		})
	}
}
