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
	t.Parallel()

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
	t.Parallel()

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
