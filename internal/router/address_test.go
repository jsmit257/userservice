package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/jsmit257/userservice/shared/v1"
)

type mockAddresser struct {
	allResp []shared.Address
	allErr  error
	getResp *shared.Address
	getErr  error
	addResp shared.UUID
	addErr  error
	updErr  error
}

func Test_GetAllGetAllAddresses(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		a        *mockAddresser
		response string
		sc       int
	}{
		"happy_path": {
			a: &mockAddresser{
				allResp: []shared.Address{{UUID: "1"}},
			},
			response: func() string {
				result, _ := json.Marshal([]shared.Address{{UUID: "1"}})
				return string(result)
			}(),
			sc: http.StatusOK,
		},
		"get_user_fails": {
			a: &mockAddresser{
				allErr: fmt.Errorf("some error"),
			},
			sc:       http.StatusBadRequest,
			response: "some error",
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{Addresser: tc.a}
			w := httptest.NewRecorder()
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					context.Background(),
					chi.RouteCtxKey,
					chi.NewRouteContext()),
				http.MethodGet,
				"tc.url",
				nil,
			)

			us.GetAllAddresses(w, r)

			resp, _ := io.ReadAll(w.Body)
			require.Equal(t, tc.sc, w.Code)
			require.Equal(t, tc.response, string(resp))
		})
	}
}

func Test_GetAddress(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		a        *mockAddresser
		userid   []string
		response string
		sc       int
	}{
		"happy_path": {
			a: &mockAddresser{
				getResp: &shared.Address{UUID: "1"},
			},
			userid: []string{"1"},
			response: func() string {
				result, _ := json.Marshal(shared.Address{UUID: "1"})
				return string(result)
			}(),
			sc: http.StatusOK,
		},
		"get_user_fails": {
			a: &mockAddresser{
				getErr: fmt.Errorf("some error"),
			},
			userid:   []string{"1"},
			sc:       http.StatusBadRequest,
			response: "some error",
		},
		"user_is_nil": {
			a:        &mockAddresser{},
			userid:   []string{""},
			sc:       http.StatusOK,
			response: "null",
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			us := &UserService{Addresser: tc.a}
			w := httptest.NewRecorder()
			rctx := chi.NewRouteContext()
			rctx.URLParams = chi.RouteParams{Keys: []string{"user_id"}, Values: tc.userid}
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					context.Background(),
					chi.RouteCtxKey,
					rctx),
				http.MethodGet,
				"tc.url",
				nil,
			)

			us.GetAddress(w, r)

			resp, _ := io.ReadAll(w.Body)
			require.Equal(t, tc.sc, w.Code)
			require.Equal(t, tc.response, string(resp))

		})
	}
}

func Test_PostAddress(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		a        *mockAddresser
		address  shared.Address
		sc       int
		response string
	}{
		"happy_path": {
			a:        &mockAddresser{addResp: shared.UUID("1")},
			address:  shared.Address{UUID: "1"},
			sc:       http.StatusOK,
			response: "1",
		},
		"unmarshal_fails": {
			a:  &mockAddresser{},
			sc: http.StatusBadRequest,
		},
		"read_fails": {
			a:  &mockAddresser{},
			sc: http.StatusBadRequest,
		},
		"addaddress_fails": {
			a:  &mockAddresser{addErr: fmt.Errorf("some error")},
			sc: http.StatusInternalServerError,
		},
		"addaddress_fails2": {
			a:  &mockAddresser{addErr: shared.AddressNotAddedError},
			sc: http.StatusInternalServerError,
		},
		"address_not_added": {
			a:  &mockAddresser{addErr: shared.UserNotAddedError},
			sc: http.StatusInternalServerError,
		},
		"address_not_found": {
			a:  &mockAddresser{},
			sc: http.StatusInternalServerError,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{Addresser: tc.a}

			body := addressToBody(&tc.address)
			if name == "unmarshal_fails" {
				body = body[1:]
			}
			bodyreader := io.Reader(bytes.NewReader([]byte(body)))
			if name == "read_fails" {
				bodyreader = errReader(name)
			}

			w := httptest.NewRecorder()
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					context.Background(),
					chi.RouteCtxKey,
					chi.NewRouteContext()),
				http.MethodPost,
				"tc.url",
				bodyreader)

			us.PostAddress(w, r)

			resp, _ := io.ReadAll(w.Body)
			require.Equal(t, tc.sc, w.Code)
			require.Equal(t, tc.response, string(resp))
		})
	}
}

func Test_PatchAddress(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		u       *mockAddresser
		address *shared.Address
		userid  []string
		sc      int
	}{
		"happy_path": {
			u:      &mockAddresser{},
			userid: []string{"1"},
			sc:     http.StatusNoContent,
		},
		"unmarshal_fails": {
			u:      &mockAddresser{},
			userid: []string{"1"},
			sc:     http.StatusBadRequest,
		},
		"read_fails": {
			u:      &mockAddresser{},
			userid: []string{"1"},
			sc:     http.StatusBadRequest,
		},
		"update_fails": {
			u: &mockAddresser{
				updErr: fmt.Errorf("some error"),
			},
			userid: []string{"1"},
			sc:     http.StatusInternalServerError,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{Addresser: tc.u}

			body := addressToBody(tc.address)
			if name == "unmarshal_fails" {
				body = body[1:]
			}
			bodyreader := io.Reader(bytes.NewReader([]byte(body)))
			if name == "read_fails" {
				bodyreader = errReader(name)
			}

			w := httptest.NewRecorder()
			rctx := chi.NewRouteContext()
			rctx.URLParams = chi.RouteParams{Keys: []string{"user_id"}, Values: tc.userid}

			r, _ := http.NewRequestWithContext(
				context.WithValue(
					context.Background(),
					chi.RouteCtxKey,
					rctx),
				http.MethodPatch,
				"tc.url",
				bodyreader)

			us.PatchAddress(w, r)

			require.Equal(t, tc.sc, w.Code)
		})
	}
}

func addressToBody(a *shared.Address) string {
	result, _ := json.Marshal(a)
	return string(result)
}

func (ma *mockAddresser) GetAllAddresses(context.Context, shared.CID) ([]shared.Address, error) {
	return ma.allResp, ma.allErr
}
func (ma *mockAddresser) GetAddress(context.Context, shared.UUID, shared.CID) (*shared.Address, error) {
	return ma.getResp, ma.getErr
}
func (ma *mockAddresser) AddAddress(context.Context, *shared.Address, shared.CID) (shared.UUID, error) {
	return ma.addResp, ma.addErr
}
func (ma *mockAddresser) UpdateAddress(context.Context, *shared.Address, shared.CID) error {
	return ma.updErr
}
