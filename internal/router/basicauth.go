package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/jsmit257/userservice/shared/v1"
)

func (us UserService) GetAuth(w http.ResponseWriter, r *http.Request) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "GetAuth", "method": http.MethodGet})

	var uuid *shared.UUID
	if name := chi.URLParam(r, "username"); name == "" {
		sc(http.StatusBadRequest).send(m, w, fmt.Errorf("missing username"), "missing parameter")
	} else if auth, err := us.Auther.GetAuthByAttrs(r.Context(), uuid, &name, cid()); err != nil {
		sc(http.StatusBadRequest).send(m, w, err, err.Error())
	} else {
		sc(http.StatusFound).success(m, w, mustJSON(auth.Redact()))
		// XXX: should get the path from the request?
		w.Header().Set("Location", fmt.Sprintf("/auth/%s", auth.UUID))
	}
}

func (us UserService) PostLogin(w http.ResponseWriter, r *http.Request) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "PostLogin", "method": http.MethodPost})

	cid := cid()

	var login shared.BasicAuth
	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(m, w, err, err.Error())
	} else if err = json.Unmarshal(body, &login); err != nil {
		sc(http.StatusBadRequest).send(m, w, err, err.Error(), string(body))
	} else if auth, err := us.Auther.Login(r.Context(), &login, cid); err != nil {
		sc(http.StatusBadRequest).send(m, w, err, err.Error())
	} else if user, err := us.Userer.GetUser(r.Context(), auth.UUID, cid); err != nil {
		sc(http.StatusInternalServerError).send(m, w, err, err.Error())
	} else {
		sc(http.StatusOK).success(m, w, mustJSON(user))
	}
}

func (us UserService) PatchLogin(w http.ResponseWriter, r *http.Request) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "PatchLogin", "method": http.MethodPatch})

	var login struct{ Old, New *shared.BasicAuth }

	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
	} else if err = json.Unmarshal(body, &login); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
	} else if err := us.Auther.ResetPassword(r.Context(), login.Old, login.New, cid()); err != nil {
		sc(http.StatusBadRequest).send(m, w, err, err.Error())
	} else {
		sc(http.StatusNoContent).success(m, w)
	}
}
