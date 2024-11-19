package seed

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/jsmit257/userservice/shared/v1"
	"github.com/stretchr/testify/require"
)

func checkStatusCode(t *testing.T, expected int, label string, resp *http.Response) []byte {
	body, err := io.ReadAll(resp.Body)
	require.Nil(t, err, "reading %s response", label)
	require.Equal(t, expected, resp.StatusCode, "response: %s is '%s'", label, body)
	return body
}

func getToken(t *testing.T, cookies []*http.Cookie) string {
	j := len(cookies)
	require.Greater(t, j, 0, "counting login cookies")
	for i, c := range cookies {
		if c.Name == cfg.CookieName {
			return c.Value
		} else if i == j {
			require.Fail(t, "auth token not found", c)
		}
	}
	return "" // require panics before this can happen
}
func Test_Validation(t *testing.T) {
	t.Parallel()

	var token string

	// log the user in
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://%s:%d/auth", cfg.ServerHost, cfg.ServerPort),
		authToReader(&shared.BasicAuth{UUID: users[logouttest].UUID}))
	require.Nil(t, err, "creating new login")

	resp, err := http.DefaultClient.Do(req)
	require.Nil(t, err, "logging in")
	checkStatusCode(t, http.StatusOK, "login", resp)

	token = getToken(t, resp.Cookies())
	require.NotEmpty(t, token, "token after cookies")

	// check the user is valid
	cookie, sc := shared.CheckValid(cfg.ServerHost, cfg.ServerPort, &http.Cookie{
		Name:    cfg.CookieName,
		Value:   token,
		Expires: time.Now().UTC().Add(time.Hour),
	})
	require.Equal(t, http.StatusFound, sc, "%v", &http.Cookie{
		Name:    cfg.CookieName,
		Value:   token,
		Expires: time.Now().UTC().Add(time.Hour),
	})
	require.NotNil(t, cookie)

	// checking an invalid user
	cookie, sc = shared.CheckValid(cfg.ServerHost, cfg.ServerPort, &http.Cookie{
		Name:    cfg.CookieName,
		Value:   "token",
		Expires: time.Now().UTC().Add(time.Hour),
	})
	require.Equal(t, http.StatusForbidden, sc)
	require.Nil(t, cookie)

	// logout the logged-in user
	url := fmt.Sprintf("http://%s:%d/logout", cfg.ServerHost, cfg.ServerPort)
	req, err = http.NewRequest(http.MethodPost, url, nil)
	require.Nil(t, err, "creating new logout")

	req.AddCookie(&http.Cookie{
		Name:    cfg.CookieName,
		Value:   token,
		Expires: time.Now().UTC().Add(time.Hour),
	})

	resp, err = http.DefaultClient.Do(req)
	require.Nil(t, err, "logging out")
	checkStatusCode(t, http.StatusNoContent, "logout", resp)

	getToken(t, resp.Cookies())
}
