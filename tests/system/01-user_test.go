package seed

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/jsmit257/userservice/shared/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_UserInit(t *testing.T) {
	for i, u := range users {
		i, u := i, u

		t.Run(u.Name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://%s:%d/user/%s",
					cfg.ServerHost,
					cfg.ServerPort,
					u.UUID),
				nil)
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			require.Equal(t, http.StatusOK, resp.StatusCode)

			require.Nil(t, json.Unmarshal(body, users[i]))
		})
	}
}

func Test_UsersGet(t *testing.T) {
	// t.Parallel() // this can't run before patch

	tcs := map[string]struct {
		resp string
		sc   int
	}{"happy_path": {sc: http.StatusOK}}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://%s:%d/users",
					cfg.ServerHost,
					cfg.ServerPort),
				nil)
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			require.Equal(t, tc.sc, resp.StatusCode)
			if resp.StatusCode != http.StatusOK {
				require.Equal(t, tc.resp, string(body))
				return
			}

			var list []shared.User
			require.Nil(t, json.Unmarshal(body, &list))
			allusers := func() []shared.User {
				var result []shared.User
				for _, u := range users {
					temp := *u
					temp.Contact = nil
					temp.Email = nil
					temp.Cell = nil
					result = append(result, temp)
				}
				return result
			}()
			require.Subset(t, allusers, list)
		})
	}
}

func Test_UserGet(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		userid shared.UUID
		resp   *shared.User
		sc     int
	}{
		// "happy_path": {}, // redundant
		"get_fails": {
			userid: "users.readonly.UUID",
			sc:     http.StatusBadRequest,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://%s:%d/user/%s",
					cfg.ServerHost,
					cfg.ServerPort,
					tc.userid),
				nil)
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.sc, resp.StatusCode)

			var got *shared.User
			if resp.StatusCode == http.StatusOK {
				err = json.Unmarshal(body, &got)
				require.Nil(t, err)
			}
			require.Equal(t, tc.resp, got)
		})
	}
}

func Test_UserPost(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		send     shared.User
		resp     string
		location string
		sc       int
	}{
		// "happy_path": {}, // covered in seed_test.go
		"user_exists": {
			send: func(u shared.User) shared.User {
				u.Name = "user_1"

				return u
			}(*users[readonly]),
			sc:   http.StatusBadRequest,
			resp: shared.UserExistsError.Error(),
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("http://%s:%d/user", cfg.ServerHost, cfg.ServerPort),
				userToReader(&tc.send))
			require.Nil(t, err)

			resp, err := (&http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}).Do(req)
			require.Nil(t, err)

			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			require.Equal(t, tc.sc, resp.StatusCode, "body: '%s'", body)
			require.Equal(t, tc.resp, string(body))
		})
	}
}

func Test_UserPatch(t *testing.T) {
	t.Parallel()

	addr := shared.Email("addr")

	tcs := map[string]struct {
		userid shared.UUID
		send   shared.User
		resp   string
		sc     int
	}{
		"happy_path": {
			userid: users[userpatch].UUID,
			send: shared.User{
				UUID:  users[userpatch].UUID,
				Name:  "test-user-random",
				Email: &addr,
			},
			sc: http.StatusNoContent,
		},
		"undeliverable": {
			userid: users[userpatch].UUID,
			send: shared.User{
				UUID: users[userpatch].UUID,
				Name: "test-user-random",
			},
			sc:   http.StatusBadRequest,
			resp: "no valid email or SMS provided",
		},
		"unique_key": {
			userid: users[userpatch].UUID,
			send:   *users[readonly],
			resp:   fmt.Sprintf("Error 1062 (23000): Duplicate entry '%s' for key 'users.name'", users[readonly].Name),
			sc:     http.StatusInternalServerError,
		},
		"not_found": {
			userid: "not-found",
			send: shared.User{
				UUID:  "somebody else",
				Email: &addr,
			},
			resp: shared.UserNotUpdatedError.Error(),
			sc:   http.StatusInternalServerError,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			// t.Parallel()

			req, err := http.NewRequest(
				http.MethodPatch,
				fmt.Sprintf("http://%s:%d/user/%s",
					cfg.ServerHost,
					cfg.ServerPort,
					tc.userid),
				userToReader(&tc.send))
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			assert.Equal(t, tc.sc, resp.StatusCode)
			require.Equal(t, tc.resp, string(body))
		})
	}
}

func Test_UserDelete(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		userid shared.UUID
		resp   string
		sc     int
	}{
		"happy_path": {
			userid: users[userdelete].UUID,
			sc:     http.StatusNoContent,
		},
		"not_found": {
			userid: "not-found",
			resp:   shared.UserNotDeletedError.Error(),
			sc:     http.StatusBadRequest,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodDelete,
				fmt.Sprintf("http://%s:%d/user/%s",
					cfg.ServerHost,
					cfg.ServerPort,
					tc.userid),
				nil)
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			assert.Equal(t, tc.sc, resp.StatusCode)
			require.Equal(t, tc.resp, string(body))
		})
	}
}

func Test_UserContactCreate(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		userid  shared.UUID
		contact shared.Contact
		resp    string
		sc      int
	}{
		// "happy_path": {}, // already in seed_test.go
		"not_found": {
			userid: "not-found",
			resp:   sql.ErrNoRows.Error(),
			sc:     http.StatusBadRequest,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("http://%s:%d/user/%s/contact",
					cfg.ServerHost,
					cfg.ServerPort,
					tc.userid),
				contactToReader(&tc.contact))
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)

			assert.Equal(t, tc.sc, resp.StatusCode)
			require.Equal(t, tc.resp, string(body))
		})
	}
}

func userToReader(u *shared.User) io.Reader {
	body, _ := json.Marshal(u)
	return bytes.NewReader(body)
}
