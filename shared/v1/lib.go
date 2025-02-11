package shared

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-gomail/gomail"
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

func (u *User) Undeliverable() bool {
	if u.Email != nil && len(*u.Email) != 0 {
		return false
	} else if u.Cell != nil && len(*u.Cell) != 0 {
		return false
	}

	return true
}

func (u *User) PasswordResetEmail(host, token string) *gomail.Message {
	if u.Email == nil {
		return nil
	}

	var emailTmpl string = `<a "href=https://%s/otp/%s">Change Password</a>` // where does this thing go?

	m := gomail.NewMessage()
	m.SetHeader("From", "no-reply@cffc.io")
	m.SetHeader("To", string(*u.Email))
	m.SetBody("text/html", fmt.Sprintf(
		emailTmpl,
		host,
		token,
	))

	return m
}

func (u *User) PasswordResetSMS(string) any {
	if u.Cell == nil {
		return nil
	}

	return func() {}
}

func (p Password) Valid() bool {
	if len(p) < 8 {
		return false
	}
	return true
}

func (e *Email) Valid() bool {
	if e == nil {
		return false
	} else if len(*e) == 0 {
		return false
	}
	return true
}

func (c *Cell) Valid() bool {
	if c == nil {
		return false
	} else if len(*c) == 0 {
		return false
	}
	return true
}

func (a BasicAuth) Redact() BasicAuth {
	a.Pass = ""
	a.Salt = ""

	return a
}
