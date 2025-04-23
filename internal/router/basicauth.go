package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	data "github.com/jsmit257/userservice/internal/relational"
	"github.com/jsmit257/userservice/shared/v1"
)

func (us UserService) GetAuth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var uuid *shared.UUID
	if name := chi.URLParam(r, "username"); name == "" {
		sc(http.StatusBadRequest).send(ctx, w, fmt.Errorf("missing username"), "missing parameter")
	} else if auth, err := us.Auther.GetAuthByAttrs(ctx, uuid, &name); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else {
		sc(http.StatusOK).success(ctx, w, mustJSON(auth.Redact()))
		// w.Header().Set("Location", fmt.Sprintf("/user/%s", auth.UUID))
	}
}

func (us UserService) PostLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	short := true
	var login shared.BasicAuth
	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if err = json.Unmarshal(body, &login); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error(), string(body))
	} else if pad := r.Header.Get("Authn-Pad"); pad == "" {
		short = false
	} else if cookie, err := r.Cookie("us-authn"); err != nil {
		sc(http.StatusForbidden).send(ctx, w, err, err.Error())
	} else if uid, code := us.ValidateOTP(ctx, cookie.Value, pad); code != http.StatusOK {
		sc(code).send(ctx, w, fmt.Errorf("bad status"))
	} else if uid != login.UUID {
		sc(http.StatusForbidden).send(ctx, w, shared.BadUserOrPassError)
	} else {
		login.Pass = shared.Password(data.Obfuscate(string(uid)))
		short = false
	}

	if short {
		return
	} else if auth, err := us.Auther.Login(ctx, &login); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if user, err := us.Userer.GetUser(ctx, auth.UUID); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err, err.Error())
	} else if cookie, code := us.Validator.Login(ctx, user.UUID, r.RemoteAddr); code != http.StatusOK {
		sc(code).send(ctx, w, fmt.Errorf("failed redis login"))
	} else {
		http.SetCookie(w, cookie)
		w.Header().Set("Location", "/") // FIXME: get location from `us`
		// sc(http.StatusMovedPermanently).success(ctx, w, nil)
		sc(http.StatusOK).success(ctx, w, mustJSON(user))
	}
}

func (us UserService) PatchLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var pair struct{ Old, New shared.Password }
	if uid := shared.UUID(chi.URLParam(r, "user_id")); uid == "" {
		sc(http.StatusBadRequest).send(ctx, w, shared.MissingParams, "missing uid")
	} else if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, "couldn't read body")
	} else if err = json.Unmarshal(body, &pair); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, "couldn't unmarshal form")
	} else if err := us.Auther.ChangePassword(ctx, uid, pair.Old, pair.New); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if cookie, code := us.Validator.Login(r.Context(), uid, r.RemoteAddr); code != http.StatusOK {
		sc(code).send(ctx, w, fmt.Errorf("failed redis login"), "failed redis login")
	} else {
		http.SetCookie(w, cookie)
		w.Header().Set("Location", "/") // FIXME: get location from `us`
		// sc(http.StatusMovedPermanently).success(ctx, w, nil)
		sc(http.StatusNoContent).success(ctx, w)
	}
}

func (us UserService) DeleteLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var login, user *shared.User
	location := map[string]string{}
	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if err = json.Unmarshal(body, &login); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, string(body))
	} else if login.Undeliverable() {
		ctx.Value(shared.CTXKey("log")).(*logrus.Entry).Error("from login")
		sc(http.StatusBadRequest).send(ctx, w, shared.Undeliverable)
	} else if _ = json.Unmarshal(body, &location); location["redirect"] == "" {
		sc(http.StatusBadRequest).send(ctx, w, err, "redirect required", string(body))
	} else if user, err = us.Userer.GetUser(ctx, login.UUID); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err, fmt.Sprintf("%v", *login))
	} else if user.Undeliverable() {
		ctx.Value(shared.CTXKey("log")).(*logrus.Entry).Error("from login")
		sc(http.StatusBadRequest).send(ctx, w, shared.Undeliverable)
	} else if login.Email != nil && user.Email != nil && *login.Email != *user.Email {
		sc(http.StatusBadRequest).send(ctx, w, fmt.Errorf("email doesn't match records"))
	} else if login.Cell != nil && user.Cell != nil && *login.Cell != *user.Cell {
		sc(http.StatusBadRequest).send(ctx, w, fmt.Errorf("cell number doesn't match records"))
	} else if token, code := us.OTP(ctx, user.UUID, r.RemoteAddr, location["redirect"]); token == "" {
		sc(code).send(ctx, w, fmt.Errorf("couldn't generate token"), "couldn't generate token")
	} else if err := us.Auther.ResetPassword(ctx, &login.UUID); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if err = us.Sender.Send(user.PasswordResetEmail(r.Host, token)); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err, err.Error())
	} else if txt := user.PasswordResetSMS(token); txt == nil {
		sc(http.StatusInternalServerError).send(ctx, w, err)
	} else {
		sc(http.StatusNoContent).success(ctx, w)
	}
}
