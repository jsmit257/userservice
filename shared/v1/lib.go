package shared

import (
	"fmt"
	"net/http"
)

// convenience method for getting the authentication state from
// a client token. errors are likely cookie-related so the client
// is responsible for what to do about bad input. unless the
// server isn't responding
func CheckValid(host string, port uint16, cookie *http.Cookie) (*http.Cookie, int) {
	if cookie == nil {
		return nil, http.StatusForbidden
	}

	url := fmt.Sprintf("http://%s:%d/valid", host, port)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, http.StatusInternalServerError
	}
	req.AddCookie(cookie)

	var result *http.Cookie
	if resp, err := http.DefaultClient.Do(req); err != nil {
		return nil, http.StatusInternalServerError
	} else if resp.StatusCode != http.StatusFound {
		return nil, http.StatusForbidden
	} else if header := resp.Header.Get("Set-Cookie"); len(header) == 0 {
		return nil, http.StatusInternalServerError
	} else if result, err = http.ParseSetCookie(header); err != nil {
		return nil, http.StatusInternalServerError
	}

	return result, http.StatusFound
}
