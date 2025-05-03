package shared

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

var now = time.Now().UTC()

func Test_CheckValid(t *testing.T) {
	// t.Parallel()

	tcs := map[string]struct {
		host     string
		port     uint16
		cookie   *http.Cookie
		location string
		validsc  int
		err      error
		checksc  int
	}{
		"new_request_fails": {
			host:    "\t",
			checksc: http.StatusInternalServerError,
		},
		"happy_path": {
			host: "Test_CheckValid",
			port: 1313,
			cookie: &http.Cookie{
				Name:     "us-authn",
				Value:    "Test_CheckValid",
				Expires:  now,
				MaxAge:   900,
				HttpOnly: true,
			},
			location: "/location",
			validsc:  http.StatusMovedPermanently,
			checksc:  http.StatusMovedPermanently,
		},
		"unparseable": {
			host:     "Test_CheckValid",
			port:     1313,
			location: "/location",
			validsc:  http.StatusMovedPermanently,
			checksc:  http.StatusInternalServerError,
		},
		"get_request_fails": {
			host:    "Test_CheckValid",
			port:    1313,
			err:     fmt.Errorf("some error"),
			checksc: http.StatusInternalServerError,
		},
		"nil_cookie": {
			host:    "Test_CheckValid",
			port:    1,
			checksc: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		// name, tc := name, tc // do NOT parallelize

		t.Run(name, func(t *testing.T) {
			httpmock.RegisterResponder(http.MethodGet,
				fmt.Sprintf("http://%s:%d/valid", tc.host, tc.port),
				func(r *http.Request) (*http.Response, error) {
					if tc.err != nil {
						return nil, tc.err
					}
					resp := httpmock.NewBytesResponse(tc.validsc, nil)
					resp.Header.Set("Location", tc.location)
					if tc.cookie != nil {
						resp.Header.Set("Set-Cookie", tc.cookie.String())
					} else if name == "unparseable" {
						resp.Header.Set("Set-Cookie", name)
					}
					return resp, nil
				})
			httpmock.Activate()
			defer httpmock.Deactivate()

			cookie, _, sc := CheckValid(tc.host, tc.port, tc.cookie)
			require.Equal(t, tc.checksc, sc)
			if tc.cookie != nil {
				require.Equal(t, tc.cookie.Value, cookie.Value)
				require.Equal(t, tc.cookie.Expires.Truncate(time.Second), cookie.Expires)
				require.Equal(t, tc.cookie.MaxAge, cookie.MaxAge)
				require.Equal(t, tc.cookie.HttpOnly, cookie.HttpOnly)
			} else {
				require.Nil(t, cookie)
			}
		})
	}
}

func Test_Undeliverable(t *testing.T) {
	t.Parallel()

	e := Email("point to me")
	c := Cell("point to me")

	tcs := map[string]struct {
		email  *Email
		cell   *Cell
		result bool
	}{
		"has_email":     {email: &e},
		"has_cell":      {cell: &c},
		"undeliverable": {result: true},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.result, (&User{
				Email: tc.email,
				Cell:  tc.cell,
			}).Undeliverable())
		})
	}
}

func Test_PasswordResetEmail(t *testing.T) {
	t.Parallel()

	require.Nil(t, (&User{}).PasswordResetEmail("host", "token"))
	email := Email("email")
	require.NotNil(t, (&User{Email: &email}).PasswordResetEmail("host", "token"))
}

func Test_PasswordResetSMS(t *testing.T) {
	t.Parallel()

	require.Nil(t, (&User{}).PasswordResetSMS("host", "token"))
	sms := Cell("cell")
	require.NotNil(t, (&User{Cell: &sms}).PasswordResetSMS("host", "token"))
}

func Test_PasswordValid(t *testing.T) {
	t.Parallel()

	require.False(t, Password("").Valid())
	require.True(t, Password("01234567").Valid())
}

func Test_EmailValid(t *testing.T) {
	t.Parallel()

	var email *Email
	require.False(t, email.Valid())
	temp := Email("")
	email = &temp
	require.False(t, email.Valid())
	*email = "email"
	require.True(t, email.Valid())
}

func Test_CellValid(t *testing.T) {
	t.Parallel()

	var cell *Cell
	require.False(t, cell.Valid())
	temp := Cell("")
	cell = &temp
	require.False(t, cell.Valid())
	*cell = "cell"
	require.True(t, cell.Valid())
}

func Test_Redact(t *testing.T) {
	t.Parallel()

	b := BasicAuth{
		Pass: "not empty",
		Salt: "not empty",
	}.Redact()
	require.Empty(t, b.Pass)
	require.Empty(t, b.Salt)
}
