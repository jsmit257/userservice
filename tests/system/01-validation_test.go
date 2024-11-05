package seed

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/jsmit257/userservice/shared/v1"
	"github.com/stretchr/testify/require"
)

func Test_Validation(t *testing.T) {
	t.Parallel()

	// log the user in
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://%s:%d/auth",
			cfg.ServerHost,
			cfg.ServerPort),
		authToReader(&shared.BasicAuth{UUID: users[logouttest].UUID}))
	require.Nil(t, err, "creating new login")

	resp, err := http.DefaultClient.Do(req)
	require.Nil(t, err, "logging in")
	body, err := io.ReadAll(resp.Body)
	require.Nil(t, err, "reading login response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "login response: %s", body)

	var token string

	j := len(resp.Cookies())
	require.Greater(t, j, 0, "counting login cookies")
	for i, c := range resp.Cookies() {
		if c.Name == cfg.CookieName {
			token = c.Value
			break
		} else if i == j {
			require.Fail(t, "auth token not found", c)
		}
	}
	require.NotEmpty(t, token, "token after cookies")

	// check the user is valid
	url := fmt.Sprintf("http://%s:%d/auth/%s/valid",
		cfg.ServerHost,
		cfg.ServerPort,
		token)
	req, err = http.NewRequest(http.MethodGet, url, nil)
	require.Nil(t, err, "creating valid")

	resp, err = http.DefaultClient.Do(req)
	require.Nil(t, err, "checking validity")
	require.Equal(t, http.StatusFound, resp.StatusCode, url)

	// checking an invalid user
	url = fmt.Sprintf("http://%s:%d/auth/%s/valid",
		cfg.ServerHost,
		cfg.ServerPort,
		"token")
	req, err = http.NewRequest(http.MethodGet, url, nil)
	require.Nil(t, err, "creating invalid")

	resp, err = http.DefaultClient.Do(req)
	require.Nil(t, err, "checking invalidity")
	require.Equal(t, http.StatusForbidden, resp.StatusCode, url)

	// logout the logged-in user
	url = fmt.Sprintf("http://%s:%d/auth/%s/logout",
		cfg.ServerHost,
		cfg.ServerPort,
		token)
	req, err = http.NewRequest(http.MethodPost, url, nil)
	require.Nil(t, err, "creating new logout")

	resp, err = http.DefaultClient.Do(req)
	require.Nil(t, err, "logging out")
	require.Equal(t, http.StatusFound, resp.StatusCode)

	j = len(resp.Cookies())
	for i, c := range resp.Cookies() {
		if c.Name == cfg.CookieName {
			require.Empty(t, c.Value, "cookie after logout")
			break
		} else if i == j {
			require.Fail(t, "auth token not found", c)
		}
	}
}
