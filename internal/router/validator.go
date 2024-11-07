package router

import (
	"net/http"

	"github.com/jsmit257/userservice/shared/v1"
	"github.com/prometheus/client_golang/prometheus"
)

func (us UserService) PostLogout(w http.ResponseWriter, r *http.Request) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "PostLogout", "method": http.MethodPost})

	token, err := r.Cookie("us-authn")
	if err != nil { // FIXME? do some kind of redirect?
		sc(http.StatusForbidden).send(m, w, shared.MissingAuthToken, shared.MissingAuthToken.Error())
		return
	}

	cookie, code := us.Validator.Logout(r.Context(), token.Value, cid())

	http.SetCookie(w, cookie)

	sc(code).success(m, w)
}

func (us UserService) GetValid(w http.ResponseWriter, r *http.Request) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "GetValid", "method": http.MethodPost})

	token, err := r.Cookie("us-authn")
	if err != nil {
		sc(http.StatusForbidden).send(m, w, shared.MissingAuthToken, shared.MissingAuthToken.Error())
		return
	}

	code := us.Validator.Valid(r.Context(), token.Value, cid())

	sc(code).success(m, w)
}
