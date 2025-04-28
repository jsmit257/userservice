package shared

import (
	"fmt"
	"net/http"

	"github.com/go-gomail/gomail"
	"github.com/sirupsen/logrus"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// convenience method for getting the authentication state from
// a client token. errors are likely cookie-related so the client
// is responsible for what to do about bad input. unless the
// server isn't responding
func CheckValid(host string, port uint16, cookie *http.Cookie) (*http.Cookie, http.Header, int) {
	url := fmt.Sprintf("http://%s:%d/valid", host, port)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, http.StatusInternalServerError
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, http.StatusInternalServerError
	} else if header := resp.Header.Get("Set-Cookie"); len(header) == 0 {
		return nil, nil, http.StatusInternalServerError
	} else if cookie, err = http.ParseSetCookie(header); err != nil {
		return nil, nil, http.StatusInternalServerError
	}

	return cookie, resp.Header, resp.StatusCode
}

// func CheckOTP(host string, port uint16, cookie *http.Cookie, pad string) (UUID, int) {
// 	var result UUID

// 	if cookie == nil {
// 		return result, http.StatusBadRequest
// 	}

// 	url := fmt.Sprintf("http://%s:%d/validateotp/%s", host, port, cookie.Value)
// 	req, err := http.NewRequest(http.MethodPost, url, nil)
// 	if err != nil {
// 		return result, http.StatusInternalServerError
// 	}

// 	resp, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		return result, http.StatusInternalServerError
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return result, http.StatusForbidden
// 	} else if body, err := io.ReadAll(resp.Body); err != nil {
// 		return result, http.StatusInternalServerError
// 	} else if len(body) == 0 {
// 		return result, http.StatusForbidden
// 	} else {
// 		result = UUID(body)
// 	}

// 	return result, resp.StatusCode
// }

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

	// there's more to the message than this; where does it go?
	var emailTmpl string = `<a "href=https://%s/otp/%s">Change Password</a>`

	m := gomail.NewMessage()
	m.SetHeader("From", "no-reply@cffc.io")
	m.SetHeader("To", string(*u.Email))
	m.SetBody("text/html", fmt.Sprintf(
		emailTmpl,
		host,
		token,
	))
	logrus.WithField("emailTmpl", fmt.Sprintf(
		emailTmpl,
		host,
		token,
	)).Error("PasswordResetEmail log message")

	return m
}

func (u *User) PasswordResetSMS(host, token string) *twilioApi.CreateMessageParams {
	if u.Cell == nil {
		return nil
	}

	return (&twilioApi.CreateMessageParams{}).
		SetValidityPeriod(0).
		SetTrafficType("").
		SetContentRetention("").
		SetTo(string(*u.Cell)).
		SetBody(fmt.Sprintf(
			`<a "href=https://%s/otp/%s">Change Password</a>`,
			host,
			token,
		))
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
