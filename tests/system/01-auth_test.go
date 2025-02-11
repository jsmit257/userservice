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
		resp     shared.UUID
		sc       int
		headers  http.Header
	}{
		"happy_path": {
			username: users[readonly].Name,
			sc:       http.StatusOK,
			resp:     users[readonly].UUID,
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

			var got = &shared.BasicAuth{}
			if resp.StatusCode == http.StatusOK {
				err = json.Unmarshal(body, &got)
				require.Nil(t, err)
			}
			require.Equal(t, tc.resp, got.UUID)
			// require.Equal(t, tc.headers, resp.Header)
		})
	}
}

func Test_LoginPost(t *testing.T) {
	// t.Parallel()

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
			resp: shared.BadUserOrPassError.Error(),
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
			// t.Parallel()

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

			require.Equal(t, tc.sc, resp.StatusCode, "cid: %s, error: %s", resp.Header.Get("Cid"), string(body))

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
				want.CTime = got.CTime
				require.Equal(t, want, got, string(tc.resp))
			} else {
				require.Equal(t, tc.resp, string(body))
			}
		})
	}
}

func Test_LoginPatch(t *testing.T) {
	// t.Parallel()

	pwd, bad, changed := shared.Password(""),
		shared.Password("bad"),
		shared.Password("snakeoil")

	tcs := map[string]struct {
		uid      shared.UUID
		old, new *shared.Password
		resp     string
		sc       int
	}{
		"happy_path": {
			uid: users[passchange].UUID,
			old: &pwd,
			new: &changed,
			sc:  http.StatusNoContent,
		},
		"bad_pass": {
			uid:  users[passchange].UUID,
			old:  &bad,
			new:  &changed,
			sc:   http.StatusBadRequest,
			resp: shared.BadUserOrPassError.Error(),
		},
		"get_fails": {
			uid:  "users.readonly.UUID",
			old:  &pwd,
			new:  &changed,
			sc:   http.StatusBadRequest,
			resp: sql.ErrNoRows.Error(),
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			// t.Parallel()

			data, _ := json.Marshal(map[string]interface{}{
				"new": tc.new,
				"old": tc.old,
			})

			req, err := http.NewRequest(
				http.MethodPatch,
				fmt.Sprintf("http://%s:%d/auth/%s",
					cfg.ServerHost,
					cfg.ServerPort,
					tc.uid),
				bytes.NewReader(data))
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.sc, resp.StatusCode, "cid: %s, error: %s", resp.Header.Get("Cid"), string(body))
			require.Equal(t, tc.resp, string(body))
		})
	}
}

func Test_LoginDelete(t *testing.T) {
	t.Parallel()

	bademail, badcell := shared.Email("bad email"), shared.Cell("bad cell")

	tcs := map[string]struct {
		login *shared.User
		redir string
		sc    int
	}{
		"happy_path": {
			login: users[logindelete],
			redir: "localhost",
			sc:    http.StatusNoContent,
		},
		"missing_redirect": {
			login: users[logindelete],
			sc:    http.StatusBadRequest,
		},
		"bad_user": {
			login: &shared.User{
				UUID:  "missing",
				Email: users[logindelete].Email,
			},
			redir: "localhost",
			sc:    http.StatusInternalServerError,
		},
		"bad_email": {
			login: &shared.User{
				UUID:  users[logindelete].UUID,
				Email: &bademail,
			},
			redir: "localhost",
			sc:    http.StatusBadRequest,
		},
		"bad_cell": {
			login: &shared.User{
				UUID: users[logindelete].UUID,
				Cell: &badcell,
			},
			redir: "localhost",
			sc:    http.StatusBadRequest,
		},
		"undeliverable": {
			login: &shared.User{
				UUID:  users[logindelete].UUID,
				Email: nil,
				Cell:  nil,
			},
			redir: "localhost",
			sc:    http.StatusBadRequest,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodDelete,
				fmt.Sprintf("http://%s:%d/auth",
					cfg.ServerHost,
					cfg.ServerPort),
				userToDelete(tc.login, tc.redir))
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)

			require.Equal(t, tc.sc, resp.StatusCode, "cid: %s", resp.Header.Get("Cid"))
		})
	}
}

func userToDelete(u *shared.User, redirect string) io.Reader {
	result, _ := json.Marshal(u)
	temp := map[string]interface{}{}
	_ = json.Unmarshal(result, &temp)
	temp["redirect"] = redirect
	result, _ = json.Marshal(temp)
	return bytes.NewReader(result)
}

func authToReader(a *shared.BasicAuth) io.Reader {
	body, _ := json.Marshal(a)
	return bytes.NewReader(body)
}
