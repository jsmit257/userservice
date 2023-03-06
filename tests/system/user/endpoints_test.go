package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	sharedv1 "github.com/jsmit257/userservice/shared/v1"

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
