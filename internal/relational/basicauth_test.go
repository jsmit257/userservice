package data

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/prometheus/client_golang/prometheus"
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
	basicFields = row{
		"uuid",
		"name",
		"pass",
		"salt",
		"success",
		"failure",
		"count",
		"mtime",
		"ctime",
	}
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
				testmetrics.MustCurryWith(prometheus.Labels{"db": "test db"}),
			}).GetAuthByAttrs(mockContext(shared.CID("Test_GetAuthByAttrs-"+name)), tc.id, tc.name)

			require.Equal(t, tc.err, err)
		})
	}
}

func Test_ChangePassword(t *testing.T) {
	t.Parallel()

	l := testLogger(t, log.Fields{"app": "basicauth_test.go", "test": "Test_ResetPassword"})

	tcs := map[string]struct {
		db       getMockDB
		uid      shared.UUID
		old, new shared.Password
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
			old: "snakeoil",
			new: "whiskeytango",
		},
		"too_short": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields).
						AddRow(basicValues...))
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))

				return db
			},
			old: "snakeoil",
			new: "shorty",
			err: shared.BadUserOrPassError,
		},
		"vanity": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields).
						AddRow(basicValues.replace(repl{1, "anaconda"})...))
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))

				return db
			},
			old: "snakeoil",
			new: "anaconda",
			err: shared.PasswordsMatch,
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
			old: "snakeoil",
			new: "snakeoil",
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
				testmetrics.MustCurryWith(prometheus.Labels{"db": "test db"}),
			}).ChangePassword(mockContext(shared.CID("Test_ResetPassword-"+name)), tc.uid, tc.old, tc.new)

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
				testmetrics.MustCurryWith(prometheus.Labels{"db": "test db"}),
			}).Login(mockContext(shared.CID("Test_Login-"+name)), &tc.login)

			require.Equal(t, tc.err, err)
		})
	}
}

func Test_ResetPassword(t *testing.T) {
	t.Parallel()

	l := testLogger(t, log.Fields{"app": "basicauth_test.go", "test": "Test_ResetPassword"})

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
			login: shared.BasicAuth{UUID: "1", Pass: "snakeoil"},
		},
		"no_basicauth_found": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").
					WillReturnRows(sqlmock.
						NewRows(basicFields))

				return db
			},
			err: sql.ErrNoRows,
		},
		"select_basicauth_fails": {
			db: func(db *sql.DB, mock sqlmock.Sqlmock, err error) *sql.DB {
				mock.ExpectQuery("").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
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

			err := (&Conn{
				tc.db(sqlmock.New()),
				nil,
				mockSqls(),
				l,
				testmetrics.MustCurryWith(prometheus.Labels{"db": "test db"}),
			}).ResetPassword(mockContext(shared.CID("Test_ResetPassword-"+name)), &tc.login.UUID)

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
				testmetrics.MustCurryWith(prometheus.Labels{"db": "test db"}),
			}).updateBasicAuth(mockContext(shared.CID("Test_updateBasicAuth-"+name)), &tc.login)

			require.Equal(t, tc.err, err)
		})
	}
}
