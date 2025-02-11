package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/jsmit257/userservice/shared/v1"
)

func (us UserService) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if user, err := us.Userer.GetAllUsers(ctx); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else {
		sc(http.StatusOK).success(ctx, w, mustJSON(user))
	}
}

func (us UserService) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userid := shared.UUID(chi.URLParam(r, "user_id"))
	if user, err := us.Userer.GetUser(ctx, userid); err != nil {
		// TODO: differentiate between a missing userID (NotFound) and an http/service error
		//       (InternalServerError/BadRequest) and log some info
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else {
		sc(http.StatusOK).success(ctx, w, mustJSON(user))
	}
}

func (us *UserService) PostUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx := r.Context()

	var user shared.User
	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
	} else if err = json.Unmarshal(body, &user); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
	} else if !user.Email.Valid() && !user.Cell.Valid() {
		sc(http.StatusBadRequest).send(ctx, w, shared.Undeliverable, "no valid email or SMS provided")
	} else if id, err := us.Userer.AddUser(ctx, &user); errors.Is(err, shared.UserExistsError) {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if errors.Is(err, shared.UserNotAddedError) {
		sc(http.StatusConflict).send(ctx, w, err)
	} else if err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err)
	} else if id == "" {
		sc(http.StatusInternalServerError).send(ctx, w, fmt.Errorf("userid_nil"))
	} else {
		sc(http.StatusCreated).success(ctx, w)
		_, _ = w.Write([]byte(id))
	}
}

func (us *UserService) PatchUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx := r.Context()

	// short := true
	var user shared.User
	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
	} else if err = json.Unmarshal(body, &user); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, fmt.Sprintf("couldn't unmarshal: '%s'", body))
	} else if user.UUID = shared.UUID(chi.URLParam(r, "user_id")); user.UUID == "" {
		sc(http.StatusBadRequest).send(ctx, w, shared.MissingParams, "missing user id")
	} else if !user.Email.Valid() && !user.Cell.Valid() {
		sc(http.StatusBadRequest).send(ctx, w, shared.Undeliverable, "no valid email or SMS provided")
	} else if err = us.Userer.UpdateUser(ctx, &user); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err, err.Error())
	} else {
		sc(http.StatusNoContent).success(ctx, w)
	}
}

func (us *UserService) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	uuid := shared.UUID(chi.URLParam(r, "user_id"))
	if err := us.Userer.DeleteUser(ctx, uuid); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else {
		sc(http.StatusNoContent).success(ctx, w)
	}
}

func (us *UserService) CreateContact(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx := r.Context()

	var contact shared.Contact
	if uuid := shared.UUID(chi.URLParam(r, "user_id")); uuid == "" {
		sc(http.StatusBadRequest).send(ctx, w, shared.MissingParams)
	} else if user, err := us.Userer.GetUser(r.Context(), uuid); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err, err.Error())
	} else if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
	} else if err = json.Unmarshal(body, &contact); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
	} else if _, err = us.Userer.CreateContact(r.Context(), user, contact); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err, err.Error())
	} else {
		sc(http.StatusOK).success(ctx, w, mustJSON(user))
	}
}
