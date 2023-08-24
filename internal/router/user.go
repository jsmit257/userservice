package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/jsmit257/userservice/internal/data/mysql"
	sharedv1 "github.com/jsmit257/userservice/shared/v1"
)

type User interface {
	Authenticate(w http.ResponseWriter, r *http.Request)
	GetUser(w http.ResponseWriter, r *http.Request)
	PostUser(w http.ResponseWriter, r *http.Request)
	PatchUser(w http.ResponseWriter, r *http.Request)
	// DeleteUser(w http.ResponseWriter, r *http.Request)
}

func (us *UserService) GetUser(w http.ResponseWriter, r *http.Request) {
	cid := cid()
	user, err := us.User.GetUser(r.Context(), chi.URLParam(r, "user_id"), cid)
	if err != nil {
		// TODO: differentiate between a missing userID (NotFound) and an http/service error
		//       (InternalServerError/BadRequest) and log some info
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	response, _ := json.Marshal(user) // nolint: errcheck // the error case can't really happen
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}

func (us *UserService) PatchUser(w http.ResponseWriter, r *http.Request) {
	cid := cid()
	var user sharedv1.User
	userID := chi.URLParam(r, "user_id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	err = json.Unmarshal(body, &user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("couldn't unmarshal: '%s'", body)))
		return
	} else if userID != user.ID {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf("user '%s' can't change attributes for user '%s'", userID, user.ID)))
		return
	} else if err = us.User.UpdateUser(r.Context(), &user, cid); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (us *UserService) PostUser(w http.ResponseWriter, r *http.Request) {
	cid := cid()
	// TODO: integrate logging/metrics and responsewrites into a helper function, probably in routes.go
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "PostUser", "method": http.MethodPost})
	var user sharedv1.User
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		m.WithLabelValues(strconv.Itoa(http.StatusInternalServerError), err.Error()).Inc()
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	err = json.Unmarshal(body, &user)
	if err != nil {
		m.WithLabelValues(strconv.Itoa(http.StatusBadRequest), err.Error()).Inc()
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id, err := us.User.AddUser(r.Context(), &user, cid)
	switch {
	case errors.Is(err, mysql.UserExistsError):
		m.WithLabelValues(strconv.Itoa(http.StatusBadRequest), fmt.Sprintf("%q", err)).Inc()
		w.WriteHeader(http.StatusBadRequest)
		return
	case errors.Is(err, mysql.UserNotAddedError):
		m.WithLabelValues(strconv.Itoa(http.StatusInternalServerError), fmt.Sprintf("%q", err)).Inc()
		w.WriteHeader(http.StatusInternalServerError)
		return
	case err != nil:
		m.WithLabelValues(strconv.Itoa(http.StatusInternalServerError), fmt.Sprintf("%q", err)).Inc()
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if id == "" {
		m.WithLabelValues(strconv.Itoa(http.StatusInternalServerError), "userid_nil").Inc()
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusMovedPermanently)
	m.WithLabelValues(strconv.Itoa(http.StatusMovedPermanently), "none").Inc()
	w.Header().Add("location", "/resetpassword") // FIXME: url isn't so simple
	w.Header().Add("otc", "one time code")       // FIXME: need to figure out codes
	_, _ = w.Write([]byte(id))
}
