package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/jsmit257/userservice/shared/v1"
)

func (us UserService) GetAllAddresses(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if addresses, err := us.Addresser.GetAllAddresses(ctx); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
		_, _ = w.Write([]byte(err.Error()))
	} else {
		sc(http.StatusOK).success(ctx, w)
		_, _ = w.Write([]byte(mustJSON(addresses)))
	}
}

func (us UserService) GetAddress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	uuid := shared.UUID(chi.URLParam(r, "address_id"))
	if address, err := us.Addresser.GetAddress(ctx, uuid); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
		_, _ = w.Write([]byte(err.Error()))
	} else {
		sc(http.StatusOK).success(ctx, w)
		_, _ = w.Write([]byte(mustJSON(address)))
	}
}

func (us *UserService) PostAddress(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var Address shared.Address

	ctx := r.Context()

	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
		return
	} else if err = json.Unmarshal(body, &Address); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
		return
	}

	id, err := us.Addresser.AddAddress(ctx, &Address)
	switch {
	case errors.Is(err, shared.AddressNotAddedError):
		sc(http.StatusInternalServerError).send(ctx, w, err)
		return
	case err != nil:
		sc(http.StatusInternalServerError).send(ctx, w, err)
		return
	}

	if id == "" {
		sc(http.StatusInternalServerError).send(ctx, w, fmt.Errorf("addressid_nil"))
	} else {
		sc(http.StatusOK).success(ctx, w)
		_, _ = w.Write([]byte(html.EscapeString(string(id))))
	}
}

func (us *UserService) PatchAddress(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var address shared.Address

	ctx := r.Context()

	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
	} else if err = json.Unmarshal(body, &address); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
		_, _ = w.Write([]byte(fmt.Sprintf("couldn't unmarshal: '%s'", html.EscapeString(string(body)))))
	} else if err = us.Addresser.UpdateAddress(ctx, &address); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err)
		_, _ = w.Write([]byte(err.Error()))
	} else {
		sc(http.StatusNoContent).success(ctx, w)
	}
}
