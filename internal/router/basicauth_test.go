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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jsmit257/userservice/shared/v1"
	"github.com/stretchr/testify/require"
)

type mockAuther struct {
	get    *shared.BasicAuth
	getErr error

	login    *shared.BasicAuth
	loginErr error

	change error

	reset error
}

func Test_GetAuth(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		u        *mockAuther
		header   http.Header
		username string
		response string
		sc       int
	}{
		"happy_path": {
			u: &mockAuther{get: &shared.BasicAuth{
				UUID: "uuid",
				Name: "happy_path",
			}},
			username: "happy_path",
			response: mustJSON(&shared.BasicAuth{
				UUID:  "uuid",
				Name:  "happy_path",
				MTime: time.Time{},
				CTime: time.Time{},
			}),
			header: http.Header{
				"Content-Type": []string{"application/json"},
				"Location":     []string{"/auth/uuid"},
			},
			sc: http.StatusFound,
		},
		"missing_username": {
			u:        &mockAuther{},
			sc:       http.StatusBadRequest,
			response: "missing parameter",
			header:   http.Header{},
		},
		"get_user_fails": {
			u:        &mockAuther{getErr: fmt.Errorf("some error")},
			sc:       http.StatusBadRequest,
			username: "get_user_fails",
			response: "some error",
			header:   http.Header{},
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{Auther: tc.u}
			rctx := chi.NewRouteContext()
			rctx.URLParams = chi.RouteParams{Keys: []string{"username"}, Values: []string{tc.username}}
			w := httptest.NewRecorder()
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					context.Background(),
					chi.RouteCtxKey,
					rctx),
				http.MethodGet,
				"tc.url",
				nil,
			)

			us.GetAuth(w, r)

			resp, _ := io.ReadAll(w.Body)

			require.Equal(t, tc.sc, w.Code)
			require.Equal(t, tc.response, string(resp))
			require.Equal(t, tc.header, w.Header())
		})
	}
}

func Test_PostLogin(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		a        *mockAuther
		u        *mockUserer
		v        *mockValidator
		login    shared.BasicAuth
		response *shared.User
		sc       int
	}{
		"happy_path": {
			a: &mockAuther{login: &shared.BasicAuth{UUID: "uuid"}},
			u: &mockUserer{user: &shared.User{UUID: "uuid"}},
			v: &mockValidator{
				login:   &testCookie,
				loginsc: http.StatusOK,
			},
			login:    shared.BasicAuth{},
			response: &shared.User{UUID: "uuid"},
			sc:       http.StatusOK,
		},
		"validation_rails": {
			a: &mockAuther{login: &shared.BasicAuth{UUID: "uuid"}},
			u: &mockUserer{user: &shared.User{UUID: "uuid"}},
			v: &mockValidator{
				login:   &testCookie,
				loginsc: http.StatusNotAcceptable,
			},
			login: shared.BasicAuth{},
			sc:    http.StatusNotAcceptable,
		},
		"read_fails": {
			a:  &mockAuther{},
			sc: http.StatusBadRequest,
		},
		"unmarshal_fails": {
			a:  &mockAuther{getErr: fmt.Errorf("some error")},
			sc: http.StatusBadRequest,
		},
		"login_fails": {
			a:  &mockAuther{loginErr: fmt.Errorf("some error")},
			sc: http.StatusBadRequest,
		},
		"getuser_fails": {
			a:  &mockAuther{login: &shared.BasicAuth{UUID: "uuid"}},
			u:  &mockUserer{userErr: fmt.Errorf("some error")},
			sc: http.StatusInternalServerError,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{
				Auther:    tc.a,
				Userer:    tc.u,
				Validator: tc.v,
			}

			body := authToBody(&tc.login)
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
				http.MethodGet,
				"tc.url",
				bodyreader,
			)

			us.PostLogin(w, r)

			resp, _ := io.ReadAll(w.Body)
			require.Equal(t, tc.sc, w.Code)
			var temp *shared.User
			_ = json.Unmarshal(resp, &temp)
			require.Equal(t, tc.response, temp)
			if w.Code == http.StatusOK {
				require.Subset(t, w.Result().Cookies(), []*http.Cookie{&testCookie})
			} else {
				require.NotSubset(t, w.Result().Cookies(), []*http.Cookie{&testCookie})
			}
		})
	}
}

func Test_PatchLogin(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		a        *mockAuther
		v        *mockValidator
		new, old *shared.BasicAuth
		sc       int
	}{
		"happy_path": {
			a:   &mockAuther{login: &shared.BasicAuth{UUID: "uuid"}},
			v:   &mockValidator{login: &http.Cookie{}, loginsc: http.StatusOK},
			old: &shared.BasicAuth{},
			new: &shared.BasicAuth{},
			sc:  http.StatusNoContent,
		},
		"read_fails": {
			a:  &mockAuther{},
			sc: http.StatusBadRequest,
		},
		"unmarshal_fails": {
			a:  &mockAuther{getErr: fmt.Errorf("some error")},
			sc: http.StatusBadRequest,
		},
		"nil_old": {
			a:   &mockAuther{login: &shared.BasicAuth{UUID: "uuid"}},
			new: &shared.BasicAuth{},
			sc:  http.StatusBadRequest,
		},
		"nil_new": {
			a:   &mockAuther{login: &shared.BasicAuth{UUID: "uuid"}},
			old: &shared.BasicAuth{},
			sc:  http.StatusBadRequest,
		},
		"change_fails": {
			a:   &mockAuther{change: fmt.Errorf("some error")},
			old: &shared.BasicAuth{},
			new: &shared.BasicAuth{},
			sc:  http.StatusBadRequest,
		},
		"authn_login_fails": {
			a:   &mockAuther{login: &shared.BasicAuth{UUID: "uuid"}},
			v:   &mockValidator{login: nil, loginsc: http.StatusForbidden},
			old: &shared.BasicAuth{},
			new: &shared.BasicAuth{},
			sc:  http.StatusForbidden,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{
				Auther:    tc.a,
				Validator: tc.v,
			}

			body := mustJSON(map[string]interface{}{
				"Old": tc.old,
				"New": tc.new,
			})
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
				http.MethodGet,
				"tc.url",
				bodyreader,
			)

			us.PatchLogin(w, r)

			require.Equal(t, tc.sc, w.Code)
		})
	}
}

func Test_DeleteLogin(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		a     *mockAuther
		login shared.BasicAuth
		sc    int
	}{
		"happy_path": {
			a:     &mockAuther{login: &shared.BasicAuth{UUID: "uuid"}},
			login: shared.BasicAuth{UUID: "uuid"},
			sc:    http.StatusNoContent,
		},
		"read_fails": {
			a:  &mockAuther{},
			sc: http.StatusBadRequest,
		},
		"unmarshal_fails": {
			a:  &mockAuther{getErr: fmt.Errorf("some error")},
			sc: http.StatusBadRequest,
		},
		"reset_fails": {
			a:  &mockAuther{reset: fmt.Errorf("some error")},
			sc: http.StatusBadRequest,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{Auther: tc.a}

			rctx := chi.NewRouteContext()
			rctx.URLParams = chi.RouteParams{Keys: []string{"user_id"}, Values: []string{string(tc.login.UUID)}}
			w := httptest.NewRecorder()
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					context.Background(),
					chi.RouteCtxKey,
					rctx),
				http.MethodGet,
				"tc.url",
				nil,
			)

			us.DeleteLogin(w, r)

			require.Equal(t, tc.sc, w.Code)
		})
	}
}

func authToBody(a *shared.BasicAuth) string {
	result, _ := json.Marshal(a)
	return string(result)
}

func (ma *mockAuther) GetAuthByAttrs(context.Context, *shared.UUID, *string, shared.CID) (*shared.BasicAuth, error) {
	return ma.get, ma.getErr
}
func (ma *mockAuther) ChangePassword(context.Context, *shared.BasicAuth, *shared.BasicAuth, shared.CID) error {
	return ma.change
}
func (ma *mockAuther) Login(context.Context, *shared.BasicAuth, shared.CID) (*shared.BasicAuth, error) {
	return ma.login, ma.loginErr
}
func (ma *mockAuther) ResetPassword(context.Context, *shared.UUID, shared.CID) error {
	return ma.reset
}
