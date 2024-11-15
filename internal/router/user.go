package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/jsmit257/userservice/shared/v1"
)

func (us UserService) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "GetAllUsers", "method": http.MethodGet})

	if user, err := us.Userer.GetAllUsers(r.Context(), cid()); err != nil {
		sc(http.StatusBadRequest).send(m, w, err, err.Error())
	} else {
		sc(http.StatusOK).success(m, w, mustJSON(user))
	}
}

func (us UserService) GetUser(w http.ResponseWriter, r *http.Request) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "GetUser", "method": http.MethodGet})

	userid := shared.UUID(chi.URLParam(r, "user_id"))
	if user, err := us.Userer.GetUser(r.Context(), userid, cid()); err != nil {
		// TODO: differentiate between a missing userID (NotFound) and an http/service error
		//       (InternalServerError/BadRequest) and log some info
		sc(http.StatusBadRequest).send(m, w, err, err.Error())
	} else {
		sc(http.StatusOK).success(m, w, mustJSON(user))
	}
}

func (us *UserService) PostUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var user shared.User

	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "PostUser", "method": http.MethodPost})

	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
	} else if err = json.Unmarshal(body, &user); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
	} else if id, err := us.Userer.AddUser(r.Context(), &user, cid()); errors.Is(err, shared.UserExistsError) {
		sc(http.StatusBadRequest).send(m, w, err, err.Error())
	} else if errors.Is(err, shared.UserNotAddedError) {
		sc(http.StatusConflict).send(m, w, err)
	} else if err != nil {
		sc(http.StatusInternalServerError).send(m, w, err)
	} else if id == "" {
		sc(http.StatusInternalServerError).send(m, w, fmt.Errorf("userid_nil"))
	} else {
		sc(http.StatusMovedPermanently).success(m, w)
		w.Header().Add("Location", "/resetpassword") // FIXME: url isn't so simple
		w.Header().Add("OTC", "one time code")       // FIXME: need to figure out codes
		_, _ = w.Write([]byte(id))
	}
}

func (us *UserService) PatchUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var user shared.User

	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "PatchUser", "method": http.MethodPatch})

	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
	} else if err = json.Unmarshal(body, &user); err != nil {
		sc(http.StatusBadRequest).send(m, w, err, fmt.Sprintf("couldn't unmarshal: '%s'", body))
	} else if err = us.Userer.UpdateUser(r.Context(), &user, cid()); err != nil {
		sc(http.StatusInternalServerError).send(m, w, err, err.Error())
	} else {
		sc(http.StatusNoContent).success(m, w)
	}
}

func (us *UserService) DeleteUser(w http.ResponseWriter, r *http.Request) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "DeleteUser", "method": http.MethodDelete})

	uuid := shared.UUID(chi.URLParam(r, "user_id"))
	if err := us.Userer.DeleteUser(r.Context(), uuid, cid()); err != nil {
		sc(http.StatusBadRequest).send(m, w, err, err.Error())
	} else {
		sc(http.StatusNoContent).success(m, w)
	}
}

func (us *UserService) CreateContact(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var contact shared.Contact

	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "CreateContact", "method": http.MethodPost})

	uuid := shared.UUID(chi.URLParam(r, "user_id"))
	if user, err := us.Userer.GetUser(r.Context(), uuid, cid()); err != nil {
		sc(http.StatusBadRequest).send(m, w, err, err.Error())
	} else if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
	} else if err = json.Unmarshal(body, &contact); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
	} else if _, err = us.Userer.CreateContact(r.Context(), user, contact, cid()); err != nil {
		sc(http.StatusInternalServerError).send(m, w, err, err.Error())
	} else {
		sc(http.StatusOK).success(m, w, mustJSON(user))
	}
}
