package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jsmit257/userservice/shared/v1"
)

func (us UserService) PostLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token, err := r.Cookie("us-authn")
	if err != nil {
		w.Header().Set("Location", us.logon)
		sc(http.StatusMovedPermanently).success(ctx, w)

		return
	}

	cookie, code := us.Validator.Logout(ctx, token.Value)

	http.SetCookie(w, cookie)

	sc(code).success(ctx, w)
}

func (us UserService) GetValid(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token, err := r.Cookie("us-authn")
	if err != nil {
		w.Header().Set("Location", us.logon)
		sc(http.StatusTemporaryRedirect).send(ctx, w, shared.MissingAuthToken)
		return
	}

	cookie, code := us.Validator.Valid(ctx, token.Value)

	http.SetCookie(w, cookie)
	if code == http.StatusTemporaryRedirect {
		w.Header().Set("Location", us.logon)
	}

	sc(code).success(ctx, w)
}

func (us UserService) GetLoginOTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pad := chi.URLParam(r, "pad")
	if pad == "" {
		sc(http.StatusBadRequest).send(ctx, w, shared.MissingParams)
		return
	}

	loc, code := us.Validator.LoginOTP(ctx, pad)

	w.Header().Set("Location", loc)

	http.SetCookie(w, &http.Cookie{
		Name:     "authn-pad",
		Value:    pad,
		Path:     "/",
		Expires:  time.Now().UTC().Add(2 * time.Minute),
		MaxAge:   int(2 * 60),
		HttpOnly: true,
	})

	sc(code).success(ctx, w)
}
