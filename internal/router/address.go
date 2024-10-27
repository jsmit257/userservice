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

func (us UserService) GetAllAddresses(w http.ResponseWriter, r *http.Request) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "GetAllAddresses", "method": http.MethodGet})

	if addresses, err := us.Addresser.GetAllAddresses(r.Context(), cid()); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
		_, _ = w.Write([]byte(err.Error()))
	} else {
		sc(http.StatusOK).success(m, w)
		_, _ = w.Write([]byte(mustJSON(addresses)))
	}
}

func (us UserService) GetAddress(w http.ResponseWriter, r *http.Request) {
	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "GetAddress", "method": http.MethodGet})

	uuid := shared.UUID(chi.URLParam(r, "address_id"))
	if address, err := us.Addresser.GetAddress(r.Context(), uuid, cid()); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
		_, _ = w.Write([]byte(err.Error()))
	} else {
		sc(http.StatusOK).success(m, w)
		_, _ = w.Write([]byte(mustJSON(address)))
	}
}

func (us *UserService) PostAddress(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var Address shared.Address

	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "PostAddress", "method": http.MethodPost})

	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
		return
	} else if err = json.Unmarshal(body, &Address); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
		return
	}

	id, err := us.Addresser.AddAddress(r.Context(), &Address, cid())
	switch {
	case errors.Is(err, shared.AddressNotAddedError):
		sc(http.StatusInternalServerError).send(m, w, err)
		return
	case err != nil:
		sc(http.StatusInternalServerError).send(m, w, err)
		return
	}

	if id == "" {
		sc(http.StatusInternalServerError).send(m, w, fmt.Errorf("addressid_nil"))
	} else {
		sc(http.StatusOK).success(m, w)
		_, _ = w.Write([]byte(id))
	}
}

func (us *UserService) PatchAddress(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var address shared.Address

	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "PatchAddress", "method": http.MethodPatch})

	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
	} else if err = json.Unmarshal(body, &address); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
		_, _ = w.Write([]byte(fmt.Sprintf("couldn't unmarshal: '%s'", body)))
	} else if err = us.Addresser.UpdateAddress(r.Context(), &address, cid()); err != nil {
		sc(http.StatusInternalServerError).send(m, w, err)
		_, _ = w.Write([]byte(err.Error()))
	} else {
		sc(http.StatusNoContent).success(m, w)
	}
}
