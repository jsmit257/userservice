package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jsmit257/userservice/shared/v1"
)

func (us UserService) PostLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token, err := r.Cookie("us-authn")
	if err != nil { // FIXME? do some kind of redirect?
		sc(http.StatusForbidden).send(ctx, w, shared.MissingAuthToken, shared.MissingAuthToken.Error())
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
		sc(http.StatusForbidden).send(ctx, w, shared.MissingAuthToken)
		return
	}

	cookie, code := us.Validator.Valid(ctx, token.Value)

	http.SetCookie(w, cookie)

	sc(code).success(ctx, w)
}

func (us UserService) GetValidOTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pad := chi.URLParam(r, "pad")
	if pad == "" {
		sc(http.StatusBadRequest).send(ctx, w, shared.MissingParams)
		return
	}

	cookie, code := us.Validator.ValidOTP(ctx, pad, r.RemoteAddr)

	http.SetCookie(w, cookie)

	sc(code).success(ctx, w)
}
