package shared

import (
	"fmt"
	"io"
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

func CheckOTP(host string, port uint16, cookie *http.Cookie, pad string) (UUID, int) {
	var result UUID

	if cookie == nil {
		return result, http.StatusBadRequest
	}

	url := fmt.Sprintf("http://%s:%d/validateotp/%s", host, port, cookie.Value)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return result, http.StatusInternalServerError
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, http.StatusInternalServerError
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result, http.StatusForbidden
	} else if body, err := io.ReadAll(resp.Body); err != nil {
		return result, http.StatusInternalServerError
	} else if len(body) == 0 {
		return result, http.StatusForbidden
	} else {
		result = UUID(body)
	}

	return result, resp.StatusCode
}
