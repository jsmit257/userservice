package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	sharedv1 "github.com/jsmit257/userservice/shared/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_UserGet(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		want sharedv1.User
		resp string
		sc   int
	}{
		"happy_path": {
			want: sharedv1.User{
				ID:   "00000000-0000-0000-0000-000000000001",
				Name: "testuser1",
			},
			sc: http.StatusOK,
		},
		"get_fails": {
			want: sharedv1.User{ID: "10000000-0000-0000-0000-000000000001"},
			resp: "sql: no rows in result set",
			sc:   http.StatusBadRequest,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:%s/user/%s", "localhost", "3000", tc.want.ID), nil)
			require.Nil(t, err)
			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)
			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)
			defer resp.Body.Close()
			require.Equal(t, tc.sc, resp.StatusCode)
			if tc.sc != http.StatusOK {
				require.Equal(t, tc.resp, string(body))
				return
			}
			got := sharedv1.User{}
			err = json.Unmarshal(body, &got)
			require.Nil(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func Test_UserPost(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		send     sharedv1.User
		resp     string
		location string
		sc       int
	}{
		"happy_path": {
			send: sharedv1.User{
				Name: "testuser-post-1",
			},
			resp:     "testuser-post-1",
			location: "/resetpassword",
			sc:       http.StatusMovedPermanently,
		},
		"user_exists": {
			send: sharedv1.User{
				Name: "testuser1",
			},
			sc: http.StatusBadRequest,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("http://%s:%s/user", "localhost", "3000"),
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
			require.Equal(t, tc.sc, resp.StatusCode, "body: '%s'", body)
			require.Nil(t, err)
			require.Equal(t, tc.resp, string(body))
		})
	}
}

func Test_UserPatch(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		userID string
		send   sharedv1.User
		resp   string
		sc     int
	}{
		"happy_path": {
			userID: "00000000-0000-0000-0000-000000000001",
			send: sharedv1.User{
				ID:   "00000000-0000-0000-0000-000000000001",
				Name: "test-user-random",
			},
			sc: http.StatusNoContent,
		},
		"not_me": {
			userID: "00000000-0000-0000-0000-000000000001",
			send:   sharedv1.User{ID: "somebody else"},
			resp:   "user '00000000-0000-0000-0000-000000000001' can't change attributes for user 'somebody else'",
			sc:     http.StatusBadRequest,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequest(
				http.MethodPatch,
				fmt.Sprintf("http://%s:%s/user/%s", "localhost", "3000", tc.userID),
				userToReader(&tc.send))
			require.Nil(t, err)
			resp, err := http.DefaultClient.Do(req)
			require.Nil(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tc.sc, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)
			require.Equal(t, tc.resp, string(body))
		})
	}
}

func userToReader(u *sharedv1.User) io.Reader {
	body, _ := json.Marshal(u)
	return bytes.NewReader(body)
}

func mustReader(a any) io.Reader {
	body, _ := json.Marshal(a)
	return bytes.NewReader(body)
}
