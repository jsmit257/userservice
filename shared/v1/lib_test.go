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
		sc       int
		response *http.Cookie
		status   int
	}{
		"happy_path": {
			host: "Test_CheckValid",
			port: 1,
			cookie: &http.Cookie{
				Name:     "us-authn",
				Value:    "Test_CheckValid",
				Expires:  now,
				MaxAge:   900,
				HttpOnly: true,
			},
			sc: http.StatusFound,
			response: &http.Cookie{
				Name:       "us-authn",
				Value:      "Test_CheckValid",
				Expires:    now.Truncate(time.Second),
				RawExpires: now.In(time.FixedZone("GMT", 0)).Format(time.RFC1123),
				MaxAge:     900,
				HttpOnly:   true,
			},
			status: http.StatusFound,
		},
		"forbidden": {
			host:   "Test_CheckValid",
			port:   1,
			cookie: &http.Cookie{},
			sc:     http.StatusForbidden,
			status: http.StatusForbidden,
		},
		"tx_empty_cookie": {
			host:   "Test_CheckValid",
			port:   1,
			cookie: &http.Cookie{},
			sc:     http.StatusNoContent,
			status: http.StatusForbidden,
		},
		// "unparseable": {
		// 	// the service won't add a nameless cookie (http.AddCookie),
		// 	// so this is untestable
		// 	host:   "Test_CheckValid",
		// 	port:   1,
		// 	cookie: &http.Cookie{Value: "Test_CheckValid"},
		// 	sc:     http.StatusFound,
		// 	err:    fmt.Errorf("http: blank cookie"),
		// },
		"nil_cookie": { // unlikely to ever happen
			host:   "Test_CheckValid",
			port:   1,
			status: http.StatusForbidden,
		},
	}

	for name, tc := range tcs {
		// name, tc := name, tc // do NOT parallelize

		t.Run(name, func(t *testing.T) {
			// tc.cookie.Raw = tc.cookie.String()
			// srv := httpmock.NewMockTransport()
			httpmock.RegisterResponder(http.MethodGet,
				fmt.Sprintf("http://%s:%d/valid", tc.host, tc.port),
				func(r *http.Request) (*http.Response, error) {
					resp := httpmock.NewBytesResponse(tc.sc, nil)
					resp.Header.Set("Set-Cookie", tc.cookie.String())
					return resp, nil
				})
			httpmock.Activate()
			defer httpmock.Deactivate()

			response, sc := CheckValid(tc.host, tc.port, tc.cookie)
			require.Equal(t, tc.status, sc)
			if tc.response != nil {
				require.Equal(t, tc.response.Value, response.Value)
				require.Equal(t, tc.response.Expires, response.Expires)
				require.Equal(t, tc.response.MaxAge, response.MaxAge)
				require.Equal(t, tc.response.HttpOnly, response.HttpOnly)
			} else {
				require.Nil(t, response)
			}
		})
	}
}

func Test_CheckOTP(t *testing.T) {
	// t.Parallel()

	tcs := map[string]struct {
		host   string
		port   uint16
		uid    UUID
		pad    string
		err    error
		sc     int
		cookie http.Cookie
		status int
	}{
		"happy_path": {
			host: "Test_CheckOTP",
			port: 1,
			pad:  "1",
			uid:  "uid",
			sc:   http.StatusOK,
			cookie: http.Cookie{
				Name:       "us-authn",
				Value:      "Test_CheckOTP",
				Expires:    now.Truncate(time.Second),
				RawExpires: now.In(time.FixedZone("GMT", 0)).Format(time.RFC1123),
				MaxAge:     900,
				HttpOnly:   true,
			},
			status: http.StatusOK,
		},
		"nil_cookie": {
			status: http.StatusBadRequest,
		},
		"forbidden": {
			sc:     http.StatusForbidden,
			status: http.StatusForbidden,
		},
		"request_fails": {
			host:   "Test_CheckOTP",
			port:   1,
			pad:    "1",
			err:    fmt.Errorf("some error"),
			status: http.StatusInternalServerError,
		},
		"empty_body": {
			host:   "Test_CheckOTP",
			port:   1,
			pad:    "1",
			sc:     http.StatusOK,
			status: http.StatusForbidden,
		},
	}

	for name, tc := range tcs {
		// name, tc := name, tc // do NOT parallelize

		t.Run(name, func(t *testing.T) {
			// tc.cookie.Raw = tc.cookie.String()
			// srv := httpmock.NewMockTransport()
			httpmock.RegisterResponder(http.MethodPost,
				fmt.Sprintf("http://%s:%d/validateotp/%s", tc.host, tc.port, tc.cookie.Value),
				func(r *http.Request) (*http.Response, error) {
					resp := httpmock.NewStringResponse(tc.sc, string(tc.uid))
					return resp, tc.err
				})
			httpmock.Activate()
			defer httpmock.Deactivate()

			c := &tc.cookie
			if name == "nil_cookie" {
				c = nil
			}

			response, sc := CheckOTP(tc.host, tc.port, c, tc.pad)
			require.Equal(t, tc.status, sc)
			require.Equal(t, tc.uid, response)
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
	require.Nil(t, (&User{}).PasswordResetEmail("host", "token"))
	email := Email("email")
	require.NotNil(t, (&User{Email: &email}).PasswordResetEmail("host", "token"))
}

func Test_PasswordResetSMS(t *testing.T) {
	require.Nil(t, (&User{}).PasswordResetSMS("token"))
	sms := Cell("cell")
	require.NotNil(t, (&User{Cell: &sms}).PasswordResetSMS("token"))
}

func Test_PasswordValid(t *testing.T) {
	require.False(t, Password("").Valid())
	require.True(t, Password("01234567").Valid())
}

func Test_EmailValid(t *testing.T) {
	var email *Email
	require.False(t, email.Valid())
	temp := Email("")
	email = &temp
	require.False(t, email.Valid())
	*email = "email"
	require.True(t, email.Valid())
}

func Test_CellValid(t *testing.T) {
	var cell *Cell
	require.False(t, cell.Valid())
	temp := Cell("")
	cell = &temp
	require.False(t, cell.Valid())
	*cell = "cell"
	require.True(t, cell.Valid())
}

func Test_Redact(t *testing.T) {
	b := BasicAuth{
		Pass: "not empty",
		Salt: "not empty",
	}.Redact()
	require.Empty(t, b.Pass)
	require.Empty(t, b.Salt)
}
