package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	sharedv1 "github.com/jsmit257/userservice/shared/v1"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/stretchr/testify/require"
)

func TestGetContact(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		mockDB  getMockDB
		contact *sharedv1.Contact
		err     error
	}{
		"happy_path": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectContact).
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "firstname", "lastname", "billto_id", "sendto_id "}).
						AddRow("1", "foo", "bar", "bill_to", "ship_to"))
				mock.ExpectQuery(selectUser).
					WillReturnRows(sqlmock.
						NewRows([]string{"name", "mtime", "dtime", "login_success"}).
						AddRow("foo", userMTime, nil, nil))
				mock.ExpectQuery(selectAddress).
					WithArgs("bill_to").
					WillReturnRows(sqlmock.
						NewRows([]string{"id"}).
						AddRow("1"))
				mock.ExpectQuery(selectAddress).
					WithArgs("ship_to").
					WillReturnRows(sqlmock.
						NewRows([]string{"id"}).
						AddRow("2"))
				return db
			},
			contact: &sharedv1.Contact{
				ID:        "1",
				FirstName: "foo",
				LastName:  "bar",
				User: &sharedv1.User{
					ID:    "1",
					Name:  "foo",
					MTime: userMTime,
				},
				BillTo: &sharedv1.Address{ID: "1"},
				ShipTo: &sharedv1.Address{ID: "2"},
			},
		},
		"no_billto": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectContact).
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "firstname", "lastname", "billto_id", "sendto_id "}).
						AddRow("1", "foo", "bar", "", "ship_to"))
				mock.ExpectQuery(selectUser).
					WillReturnRows(sqlmock.
						NewRows([]string{"name", "mtime", "dtime", "login_success"}).
						AddRow("foo", userMTime, nil, nil))
				mock.ExpectQuery(selectAddress).
					WithArgs("ship_to").
					WillReturnRows(sqlmock.
						NewRows([]string{"id"}).
						AddRow("2"))
				return db
			},
			contact: &sharedv1.Contact{
				ID:        "1",
				FirstName: "foo",
				LastName:  "bar",
				User: &sharedv1.User{
					ID:    "1",
					Name:  "foo",
					MTime: userMTime,
				},
				ShipTo: &sharedv1.Address{ID: "2"},
			},
		},
		"no_shipto": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectContact).
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "firstname", "lastname", "billto_id", "sendto_id "}).
						AddRow("1", "foo", "bar", "", ""))
				mock.ExpectQuery(selectUser).
					WillReturnRows(sqlmock.
						NewRows([]string{"name", "mtime", "dtime", "login_success"}).
						AddRow("foo", userMTime, nil, nil))
				return db
			},
			contact: &sharedv1.Contact{
				ID:        "1",
				FirstName: "foo",
				LastName:  "bar",
				User: &sharedv1.User{
					ID:    "1",
					Name:  "foo",
					MTime: userMTime,
				},
			},
		},
		"selectRow_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectContact).WithArgs("1").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
		},
		"selectUser_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectContact).
					WithArgs("1").
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "firstname", "lastname", "billto_id", "sendto_id "}).
						AddRow("1", "foo", "bar", "bill_to", "ship_to"))
				mock.ExpectQuery(selectUser).WithArgs("1").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
		},
		"selectbillto_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectContact).
					WithArgs("1").
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "firstname", "lastname", "billto_id", "sendto_id "}).
						AddRow("1", "foo", "bar", "bill_to", "ship_to"))
				mock.ExpectQuery(selectUser).
					WithArgs("1").
					WillReturnRows(sqlmock.
						NewRows([]string{"name", "mtime", "dtime", "login_success"}).
						AddRow("foo", userMTime, nil, nil))
				mock.ExpectQuery(selectAddress).WithArgs("bill_to").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
		},
		"selectshipto_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectQuery(selectContact).
					WithArgs("1").
					WillReturnRows(sqlmock.
						NewRows([]string{"id", "firstname", "lastname", "billto_id", "sendto_id "}).
						AddRow("1", "foo", "bar", "bill_to", "ship_to"))
				mock.ExpectQuery(selectUser).
					WithArgs("1").
					WillReturnRows(sqlmock.
						NewRows([]string{"name", "mtime", "dtime", "login_success"}).
						AddRow("foo", userMTime, nil, nil))
				mock.ExpectQuery(selectAddress).
					WithArgs("bill_to").
					WillReturnRows(sqlmock.
						NewRows([]string{"id"}).
						AddRow("1"))
				mock.ExpectQuery(selectAddress).WithArgs("ship_to").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			contact, err := (&Conn{tc.mockDB(), nil}).GetContact(context.Background(), "1", sharedv1.CID("TestGetContact-"+name))
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.contact, contact)
		})
	}
}

func TestAddContact(t *testing.T) {
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
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			contact: &sharedv1.Contact{User: &sharedv1.User{
				ID:    "1",
				Name:  "old username",
				MTime: userMTime,
			}},
			result: mockUUIDGen().String(),
		},
		"nil_user": {
			mockDB: func() *sql.DB {
				db, _, _ := sqlmock.New()
				return db
			},
			contact: &sharedv1.Contact{},
			err:     fmt.Errorf("contact requires a user"),
		},
		"nil_user_id": {
			mockDB: func() *sql.DB {
				db, _, _ := sqlmock.New()
				return db
			},
			contact: &sharedv1.Contact{User: &sharedv1.User{}},
			err:     fmt.Errorf("contact requires a valid user"),
		},
		"exec_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(".*").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			contact: &sharedv1.Contact{User: &sharedv1.User{ID: "1", Name: "old username", MTime: userMTime}},
			err:     fmt.Errorf("some error"),
		},
		"no_update": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			contact: &sharedv1.Contact{User: &sharedv1.User{ID: "1", Name: "old username", MTime: userMTime}},
			err:     fmt.Errorf("contact was not inserted: '%s'", mockUUIDGen()),
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := (&Conn{tc.mockDB(), mockUUIDGen}).AddContact(context.Background(), tc.contact, sharedv1.CID("TestAddContact-"+name))
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.result, result)
		})
	}
}

func TestUpdateContact(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		mockDB  getMockDB
		contact *sharedv1.Contact
		err     error
	}{
		"happy_path": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
			contact: &sharedv1.Contact{},
		},
		"exec_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(".*").WillReturnError(fmt.Errorf("some error"))
				return db
			},
			contact: &sharedv1.Contact{},
			err:     fmt.Errorf("some error"),
		},
		"update_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			contact: &sharedv1.Contact{ID: "update fails"},
			err:     fmt.Errorf("contact was not updated: 'update fails'"),
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.err, (&Conn{tc.mockDB(), nil}).UpdateContact(context.Background(), tc.contact, sharedv1.CID("TestUpdateContact-"+name)))
		})
	}
}
func TestDeleteContact(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		mockDB getMockDB
		err    error
	}{
		"happy_path": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(deleteContact).WillReturnResult(sqlmock.NewResult(0, 1))
				return db
			},
		},
		"exec_fails": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(deleteContact).WillReturnError(fmt.Errorf("some error"))
				return db
			},
			err: fmt.Errorf("some error"),
		},
		"user_not_found": {
			mockDB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectExec(deleteContact).WillReturnResult(sqlmock.NewResult(0, 0))
				return db
			},
			err: fmt.Errorf("contact could not be deleted"),
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(
				t,
				tc.err,
				(&Conn{tc.mockDB(), nil}).DeleteContact(context.Background(), "1", sharedv1.CID("TestDeleteContact-"+name)))
		})
	}
}
