package router

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"

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
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	user, err := us.User.GetUser(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	response, _ := json.Marshal(user) // nolint: errcheck // the error case can't really happen
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}

func (us *UserService) PatchUser(w http.ResponseWriter, r *http.Request) {
	var user sharedv1.User
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	err = json.Unmarshal(body, &user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if userID != user.ID {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if err = us.User.UpdateUser(r.Context(), &user); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (us *UserService) PostUser(w http.ResponseWriter, r *http.Request) {
	// TODO: integrate logging/metrics and responsewrites into a helper function, probably in routes.go
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "PostUser", "method": http.MethodPost})
	var user sharedv1.User
	body, err := io.ReadAll(r.Body)
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
	id, err := us.User.AddUser(r.Context(), &user)
	if err != nil {
		m.WithLabelValues(strconv.Itoa(http.StatusInternalServerError), err.Error()).Inc()
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if id == "" {
		m.WithLabelValues(strconv.Itoa(http.StatusInternalServerError), "userid_nil").Inc()
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(id))
	m.WithLabelValues(strconv.Itoa(http.StatusOK), "none").Inc()
}
