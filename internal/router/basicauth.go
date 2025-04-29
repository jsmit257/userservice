package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

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

	var login shared.BasicAuth
	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if err = json.Unmarshal(body, &login); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error(), string(body))
	} else if auth, err := us.Auther.Login(ctx, &login); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if cookie, code := us.Validator.Login(ctx, auth.UUID, r.RemoteAddr); code != http.StatusOK {
		sc(code).send(ctx, w, fmt.Errorf("failed redis login"))
	} else {
		http.SetCookie(w, cookie)
		w.Header().Set("Location", us.success)
		sc(http.StatusMovedPermanently).success(ctx, w)
	}
}

func authzPad(us UserService, w http.ResponseWriter, r *http.Request, id shared.UUID, old shared.Password) (shared.Password, int) {
	ctx := r.Context()

	otp, err := r.Cookie("authn-pad")
	if err == http.ErrNoCookie {
		return old, http.StatusOK
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "authn-pad",
		Value:    "",
		Path:     "/",
		Expires:  time.Time{},
		MaxAge:   -1,
		HttpOnly: true,
	})

	if otpID, code := us.Validator.CompleteOTP(ctx, otp.Value); code != http.StatusOK {
		return old, code
	} else if otpID != id {
		return old, http.StatusBadRequest // enter the wrong username?
	}

	pwd, err := us.Auther.ResetPassword(ctx, &id)
	if err != nil {
		return old, http.StatusInternalServerError
	}

	// mysql has a problem updating the same row twice in close succession; we've
	// seen this in integration testing, too, when creating a user, then immediately
	// setting the password; for whatever reason, it seems to need some time for the
	// driver to settle before it can handle a new request; in this case, the
	// password field is getting clobbered first, then set with the user imput
	//
	// FIXME: FIXME: FIXME: this should *mot* be here
	time.Sleep(500 * time.Millisecond)

	return pwd, http.StatusOK
}

func (us UserService) PatchLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var code int
	var pair struct{ Old, New shared.Password }
	if id := shared.UUID(chi.URLParam(r, "user_id")); id == "" {
		sc(http.StatusBadRequest).send(ctx, w, shared.MissingParams, "missing uid")
	} else if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, "couldn't read body")
	} else if err = json.Unmarshal(body, &pair); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, "couldn't unmarshal form")
	} else if pair.Old, code = authzPad(us, w, r, id, pair.Old); code != http.StatusOK {
		sc(code).send(ctx, w, err)
	} else if err := us.Auther.ChangePassword(ctx, id, pair.Old, pair.New); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err, err.Error())
	} else if cookie, code := us.Validator.Login(r.Context(), id, r.RemoteAddr); code != http.StatusOK {
		sc(code).send(ctx, w, fmt.Errorf("failed redis login"), "failed redis login")
	} else {
		http.SetCookie(w, cookie)
		w.Header().Set("Location", us.success)
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
	} else if pad, code := us.OTP(ctx, user.UUID, r.RemoteAddr, location["redirect"]); pad == "" {
		sc(code).send(ctx, w, fmt.Errorf("couldn't generate token"), "couldn't generate token")
	} else if err = us.MailSender.Send(user.PasswordResetEmail(r.Host, pad)); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err, err.Error())
	} else if err := us.SmsSender.Send(user.PasswordResetSMS(r.Host, pad)); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err)
	} else {
		sc(http.StatusNoContent).success(ctx, w)
	}
}
