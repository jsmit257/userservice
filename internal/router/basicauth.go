package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-gomail/gomail"

	"github.com/jsmit257/userservice/shared/v1"
)

var emailTmpl string = `<a "href=https://%s/otp/%s">Change Password</a>`

func (us UserService) GetAuth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var uuid *shared.UUID
	if name := chi.URLParam(r, "username"); name == "" {
		sc(http.StatusBadRequest).send(ctx, w, fmt.Errorf("missing username"), "missing parameter")
	} else if auth, err := us.Auther.GetAuthByAttrs(ctx, uuid, &name); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else {
		sc(http.StatusFound).success(ctx, w, mustJSON(auth.Redact()))
		// XXX: should get the path from the request?
		w.Header().Set("Location", fmt.Sprintf("/auth/%s", auth.UUID))
	}
}

func (us UserService) PostLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var login shared.BasicAuth
	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if err = json.Unmarshal(body, &login); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error(), string(body))
	} else if auth, err := us.Auther.Login(ctx, &login); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if user, err := us.Userer.GetUser(ctx, auth.UUID); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err, err.Error())
	} else if cookie, code := us.Validator.Login(ctx, user.UUID, r.RemoteAddr); code != http.StatusOK {
		sc(code).send(ctx, w, fmt.Errorf("failed redis login"))
	} else {
		http.SetCookie(w, cookie)
		sc(http.StatusOK).success(ctx, w, mustJSON(user))
	}
}

func (us UserService) PatchLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var login struct{ Old, New *shared.BasicAuth }
	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
	} else if err = json.Unmarshal(body, &login); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
	} else if login.Old == nil || login.New == nil {
		sc(http.StatusBadRequest).send(ctx, w, fmt.Errorf("incomplete input"))
	} else if err := us.Auther.ChangePassword(ctx, login.Old, login.New); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if cookie, code := us.Validator.Login(r.Context(), login.Old.UUID, r.RemoteAddr); code != http.StatusOK {
		sc(code).send(ctx, w, fmt.Errorf("failed redis login"))
	} else {
		http.SetCookie(w, cookie)
		sc(http.StatusNoContent).success(ctx, w)
	}
}

func genResetEmail(u *shared.User, host, token string) *gomail.Message {
	if u.Email == nil {
		return nil
	}

	m := gomail.NewMessage()
	m.SetHeader("From", "no-reply@cffc.io")
	m.SetHeader("To", *u.Email)
	m.SetBody("text/html", fmt.Sprintf(
		emailTmpl,
		host,
		token,
	))

	return m
}

func genResetTxt(u *shared.User, _ string) (func(), error) {
	if u.Cell == nil {
		return nil, fmt.Errorf("no SMS enabled number avalilable")
	}

	return func() {}, nil
}

func (us UserService) DeleteLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var input, user *shared.User
	var redirect struct {
		Location string `json:"redirect"`
	}
	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if err = json.Unmarshal(body, &input); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error(), string(body))
	} else if _ = json.Unmarshal(body, &redirect); redirect.Location == "" {
		sc(http.StatusBadRequest).send(ctx, w, err, "redirect required", string(body))
	} else if user, err = us.Userer.GetUser(ctx, input.UUID); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err, err.Error())
	} else if user.Email == nil && user.Cell == nil {
		sc(http.StatusBadRequest).send(ctx, w, shared.Undeliverable)
	} else if input.Email != nil && user.Email != nil && *input.Email != *user.Email {
		sc(http.StatusBadRequest).send(ctx, w, fmt.Errorf("email doesn't match records"))
	} else if input.Cell != nil && user.Cell != nil && *input.Cell != *user.Cell {
		sc(http.StatusBadRequest).send(ctx, w, fmt.Errorf("cell number doesn't match records"))
	} else if token, code := us.OTP(ctx, user.UUID, r.RemoteAddr, redirect.Location); token == "" {
		sc(code).send(ctx, w, fmt.Errorf("couldn't generate token"))
	} else if err := us.Auther.ResetPassword(ctx, &input.UUID); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if err = us.Sender.Send(genResetEmail(user, r.Host, token)); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err, err.Error())
	} else if txt, err := genResetTxt(user, token); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err, err.Error())
	} else {
		txt()
		sc(http.StatusNoContent).success(ctx, w)
	}
}
