package seed

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jsmit257/userservice/shared/v1"
)

func Test_AuthGet(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		username string
		resp     *shared.User
		sc       int
		headers  http.Header
	}{
		"happy_path": {
			username: users[readonly].Name,
			sc:       http.StatusFound,
			resp:     users[readonly],
		},
		"get_fails": {
			username: "users.readonly.UUID",
			sc:       http.StatusBadRequest,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://%s:%d/auth/%s",
					cfg.ServerHost,
					cfg.ServerPort,
					tc.username),
				nil)
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.sc, resp.StatusCode)

			var got *shared.User
			if resp.StatusCode == http.StatusFound {
				err = json.Unmarshal(body, &got)
				require.Nil(t, err)
			}
			require.Equal(t, tc.resp, got)
			// require.Equal(t, tc.headers, resp.Header)
		})
	}
}

func Test_LoginPost(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		login *shared.BasicAuth
		resp  string
		sc    int
	}{
		"happy_path": {
			login: &shared.BasicAuth{UUID: users[logintest].UUID},
			sc:    http.StatusOK,
			resp: func(u shared.User) string {
				resp, _ := json.Marshal(u)
				return string(resp)
			}(*users[logintest]),
		},
		"bad_pass": {
			login: &shared.BasicAuth{
				UUID: users[pwdtest].UUID,
				Pass: "wrong!",
			},
			sc:   http.StatusBadRequest,
			resp: shared.PasswordsMatch.Error(),
		},
		"get_fails": {
			login: &shared.BasicAuth{UUID: "users.readonly.UUID"},
			sc:    http.StatusBadRequest,
			resp:  sql.ErrNoRows.Error(),
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("http://%s:%d/auth",
					cfg.ServerHost,
					cfg.ServerPort),
				authToReader(tc.login))
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.sc, resp.StatusCode, string(body))

			if resp.StatusCode == http.StatusOK {
				j := len(resp.Cookies())
				for i, c := range resp.Cookies() {
					if c.Name == "us-authn" {
						require.NotEmpty(t, c.Value)
						break
					} else if i == j {
						require.Fail(t, "auth token not found", c)
					}
				}
				var want, got *shared.User
				require.Nil(t, json.Unmarshal([]byte(tc.resp), &want), tc.resp)
				require.Nil(t, json.Unmarshal(body, &got), string(body))
				want.MTime = got.MTime
				require.Equal(t, want, got)
			} else {
				require.Equal(t, tc.resp, string(body))
			}
		})
	}
}

func Test_LoginPatch(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		old, new *shared.BasicAuth
		resp     string
		sc       int
	}{
		"happy_path": {
			old: &shared.BasicAuth{UUID: users[passchange].UUID},
			new: &shared.BasicAuth{
				UUID: users[passchange].UUID,
				Pass: "changed!",
			},
			sc: http.StatusNoContent,
		},
		"bad_pass": {
			old: &shared.BasicAuth{
				UUID: users[pwdfail].UUID,
				Pass: "wrong!",
			},
			sc:   http.StatusBadRequest,
			resp: shared.PasswordsMatch.Error(),
		},
		"get_fails": {
			old:  &shared.BasicAuth{UUID: "users.readonly.UUID"},
			sc:   http.StatusBadRequest,
			resp: sql.ErrNoRows.Error(),
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data, _ := json.Marshal(map[string]interface{}{
				"new": tc.new,
				"old": tc.old,
			})

			req, err := http.NewRequest(
				http.MethodPatch,
				fmt.Sprintf("http://%s:%d/auth/%s",
					cfg.ServerHost,
					cfg.ServerPort,
					tc.old.UUID),
				bytes.NewReader(data))
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.sc, resp.StatusCode, string(body))
			require.Equal(t, tc.resp, string(body))
		})
	}
}

func authToReader(a *shared.BasicAuth) io.Reader {
	body, _ := json.Marshal(a)
	return bytes.NewReader(body)
}
