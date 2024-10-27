package data

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/jsmit257/userservice/shared/v1"
)

var (
	_basic = shared.BasicAuth{
		UUID:         "basic",
		Name:         "basic",
		Pass:         "snakeoilpinch",
		Salt:         "pinch",
		LoginSuccess: &rightaboutnow,
		LoginFailure: nil,
		FailureCount: 0,
		MTime:        rightaboutnow,
		CTime:        rightaboutnow,
	}
	basicFields = row{"uuid", "name", "pass", "salt", "success", "failure", "count", "mtime", "ctime"}
	basicValues = values{
		_basic.UUID,
		_basic.Name,
		_basic.Pass,
		_basic.Salt,
		_basic.LoginSuccess,
		_basic.LoginFailure,
		_basic.FailureCount,
		_basic.MTime,
		_basic.CTime,
	}
)

func Test_GetAuthByAttrs(t *testing.T) {
	t.Parallel()

	l := testLogger(t, log.Fields{"app": "basicauth_test.go", "test": "Test_GetAuthByAttrs"})

	tcs := map[string]struct {
		db     getMockDB
		id     *shared.UUID
		name   *string
		result *shared.BasicAuth
		err    error
	}{
		"happy_path": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields).
						AddRow(basicValues...))
				return db
			},
			// result: &_basic,
		},
		"no_rows_returned": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields))
				return db
			},
			err: sql.ErrNoRows,
		},
		"query_error": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
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

			_, err := (&Conn{
				tc.db(sqlmock.New()),
				nil,
				mockSqls(),
				l,
			}).GetAuthByAttrs(context.Background(), tc.id, tc.name, shared.CID("Test_GetAuthByAttrs-"+name))

			require.Equal(t, tc.err, err)
		})
	}
}

func Test_ResetPassword(t *testing.T) {
	t.Parallel()

	l := testLogger(t, log.Fields{"app": "basicauth_test.go", "test": "Test_ResetPassword"})

	tcs := map[string]struct {
		db       getMockDB
		old, new shared.BasicAuth
		err      error
	}{
		"happy_path": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields).
						AddRow(basicValues...))
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))

				return db
			},
			old: shared.BasicAuth{Pass: "snakeoil"},
			new: shared.BasicAuth{Pass: "whiskey"},
		},
		"login_fails": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
		},
		"same_password": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields).
						AddRow(basicValues...))
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))

				return db
			},
			old: shared.BasicAuth{Pass: "snakeoil"},
			new: shared.BasicAuth{Pass: "snakeoil"},
			err: shared.PasswordsMatch,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := (&Conn{
				tc.db(sqlmock.New()),
				nil,
				mockSqls(),
				l,
			}).ResetPassword(context.Background(), &tc.old, &tc.new, shared.CID("Test_ResetPassword-"+name))

			require.Equal(t, tc.err, err)
		})
	}
}

func Test_Login(t *testing.T) {
	t.Parallel()

	l := testLogger(t, log.Fields{"app": "basicauth_test.go", "test": "Test_Login"})

	tcs := map[string]struct {
		db    getMockDB
		login shared.BasicAuth
		err   error
	}{
		"happy_path": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields).
						AddRow(basicValues...))
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))

				return db
			},
			login: shared.BasicAuth{Pass: "snakeoil"},
		},
		"no_rows_found": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields))
				return db
			},
			err: sql.ErrNoRows,
		},
		"select_fails": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
		},
		"failure_count": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields).
						AddRow(basicValues.replace(repl{6, maxfailure + 1})...))
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))

				return db
			},
			err: shared.MaxFailedLoginError,
		},
		"bad_password": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields).
						AddRow(basicValues...))
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))

				return db
			},
			err: shared.BadUserOrPassError,
		},
		"bad_password_exec_fails": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields).
						AddRow(basicValues...))
				mock.ExpectExec("").WillReturnError(fmt.Errorf("some error"))

				return db
			},
			err: fmt.Errorf("some error"),
		},
		"update_fails": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields).
						AddRow(basicValues...))
				mock.ExpectExec("").WillReturnError(fmt.Errorf("some error"))

				return db
			},
			login: shared.BasicAuth{Pass: "snakeoil"},
			err:   fmt.Errorf("some error"),
		},
		"update_no_rows": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields).
						AddRow(basicValues...))
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 0))

				return db
			},
			login: shared.BasicAuth{Pass: "snakeoil"},
			err:   shared.UserNotUpdatedError,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := (&Conn{
				tc.db(sqlmock.New()),
				nil,
				mockSqls(),
				l,
			}).Login(context.Background(), &tc.login, shared.CID("Test_Login-"+name))

			require.Equal(t, tc.err, err)
		})
	}
}

func Test_updateBasicAuth(t *testing.T) {
	t.Parallel()

	l := testLogger(t, log.Fields{"app": "basicauth_test.go", "test": "Test_updateBasicAuth"})

	tcs := map[string]struct {
		db    getMockDB
		login shared.BasicAuth
		err   error
	}{
		"happy_path": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
		},
		"no_rows_updated": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			err: shared.UserNotUpdatedError,
		},
		"query_error": {db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
			mock.ExpectExec("").WillReturnError(fmt.Errorf("some error"))
			return db
		},
			err: fmt.Errorf("some error"),
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := (&Conn{
				tc.db(sqlmock.New()),
				nil,
				mockSqls(),
				l,
			}).updateBasicAuth(context.Background(), &tc.login, shared.CID("Test_updateBasicAuth-"+name))

			require.Equal(t, tc.err, err)
		})
	}
}
