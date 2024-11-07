package shared

import (
	"fmt"
	"net/http"
)

// convenience method for getting the authorization state from a client
// token;
func CheckValid(host string, port uint16, cookie *http.Cookie) (bool, error) {
	url := fmt.Sprintf("http://%s:%d/valid", host, port)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	req.AddCookie(cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}

	return http.StatusFound == resp.StatusCode, nil
}
