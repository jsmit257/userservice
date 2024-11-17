package shared

import (
	"fmt"
	"net/http"
	"os"
)

// convenience method for getting the authentication state from
// a client token. errors are likely cookie-related so the client
// is responsible for what to do about bad input. unless the
// server isn't responding
func CheckValid(host string, port uint16, cookie *http.Cookie) (*http.Cookie, error) {
	if cookie == nil {
		return nil, fmt.Errorf("nil cookie")
	}

	url := fmt.Sprintf("http://%s:%d/valid", host, port)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.AddCookie(cookie)

	var result *http.Cookie
	if resp, err := http.DefaultClient.Do(req); err != nil {
		return nil, err
	} else if resp.StatusCode != http.StatusFound {
		return nil, MissingAuthToken
	} else if header := resp.Header.Get("Set-Cookie"); len(header) == 0 {
		fmt.Fprintf(os.Stderr, "%v/n", resp.Header)
		return nil, MissingAuthToken
	} else if result, err = http.ParseSetCookie(header); err != nil {
		return nil, err
	}

	return result, nil
}
