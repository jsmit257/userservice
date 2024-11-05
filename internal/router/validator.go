package router

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

func (us UserService) Logout(w http.ResponseWriter, r *http.Request) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "Logout", "method": http.MethodPost})

	token := chi.URLParam(r, "token")
	if token == "" {
		sc(http.StatusBadRequest).send(m, w, fmt.Errorf("missing token"), "missing parameter")
		return
	}

	cookie, code := us.Validator.Logout(r.Context(), token, cid())

	http.SetCookie(w, cookie)

	sc(code).success(m, w)
}

func (us UserService) Valid(w http.ResponseWriter, r *http.Request) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "Valid", "method": http.MethodPost})

	token := chi.URLParam(r, "token")
	if token == "" {
		sc(http.StatusBadRequest).send(m, w, fmt.Errorf("missing token"), "missing parameter")
		return
	}

	code := us.Validator.Valid(r.Context(), token, cid())

	sc(code).success(m, w)
}
