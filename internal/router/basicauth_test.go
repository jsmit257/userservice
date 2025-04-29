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
	"github.com/google/uuid"
	"github.com/jsmit257/userservice/shared/v1"
	"github.com/stretchr/testify/require"
)

type mockAuther struct {
	get    *shared.BasicAuth
	getErr error

	login    *shared.BasicAuth
	loginErr error

	change error

	reset    shared.Password
	resetErr error
}

func Test_GetAuth(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		u        *mockAuther
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
			sc: http.StatusOK,
		},
		"missing_username": {
			u:        &mockAuther{},
			sc:       http.StatusBadRequest,
			response: "missing parameter",
		},
		"get_user_fails": {
			u:        &mockAuther{getErr: fmt.Errorf("some error")},
			sc:       http.StatusBadRequest,
			username: "get_user_fails",
			response: "some error",
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
					mockContext(),
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
		})
	}
}

func Test_PostLogin(t *testing.T) {
	t.Parallel()

	uid := shared.UUID(uuid.NewString()[:5])

	tcs := map[string]struct {
		a     *mockAuther
		v     *mockValidator
		login shared.BasicAuth
		sc    int
	}{
		"happy_path": {
			a: &mockAuther{login: &shared.BasicAuth{UUID: uid}},
			v: &mockValidator{
				login:   &testCookie,
				loginsc: http.StatusOK,
			},
			sc: http.StatusMovedPermanently,
		},
		"read_fails": {
			a:  &mockAuther{},
			sc: http.StatusBadRequest,
		},
		"unmarshal_fails": {
			a:  &mockAuther{getErr: fmt.Errorf("some error")},
			sc: http.StatusBadRequest,
		},
		"auth_login_fails": {
			a:  &mockAuther{loginErr: fmt.Errorf("auth_login_fails")},
			sc: http.StatusBadRequest,
		},
		"valid_login_fails": {
			a: &mockAuther{login: &shared.BasicAuth{UUID: uid}},
			v: &mockValidator{
				login:   &testCookie,
				loginsc: http.StatusBadRequest,
			},
			sc: http.StatusBadRequest,
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
					mockContext(),
					chi.RouteCtxKey,
					chi.NewRouteContext()),
				http.MethodGet,
				"tc.url",
				bodyreader,
			)

			us.PostLogin(w, r)

			require.Equal(t, tc.sc, w.Code)
			if w.Code == http.StatusMovedPermanently {
				require.Subset(t, w.Result().Cookies(), []*http.Cookie{&testCookie})
			} else {
				require.NotSubset(t, w.Result().Cookies(), []*http.Cookie{&testCookie})
			}
		})
	}
}

func Test_PatchLogin(t *testing.T) {
	t.Parallel()

	pass := shared.Password("password")

	tcs := map[string]struct {
		a        *mockAuther
		v        *mockValidator
		id       shared.UUID
		cookie   *http.Cookie
		new, old *shared.Password
		sc       int
	}{
		"happy_path": {
			a: &mockAuther{login: &shared.BasicAuth{UUID: "uuid"}},
			v: &mockValidator{
				completeotp:   "uuid",
				completeotpsc: http.StatusOK,
				login:         &http.Cookie{},
				loginsc:       http.StatusOK,
			},
			id:     "uuid",
			cookie: &http.Cookie{Name: "authn-pad"},
			old:    &pass,
			new:    &pass,
			sc:     http.StatusNoContent,
		},
		"param_missing": {
			// a:  &mockAuther{},
			sc: http.StatusBadRequest,
		},
		"read_fails": {
			// a:   &mockAuther{},
			id:  "user_id",
			old: &pass,
			sc:  http.StatusBadRequest,
		},
		"unmarshal_fails": {
			// a:   &mockAuther{getErr: fmt.Errorf("some error")},
			id:  "uuid",
			old: &pass,
			sc:  http.StatusBadRequest,
		},
		"complete_fails": {
			a:      &mockAuther{resetErr: fmt.Errorf("some error")},
			v:      &mockValidator{completeotpsc: http.StatusBadRequest},
			id:     "uuid",
			cookie: &http.Cookie{Name: "authn-pad"},
			sc:     http.StatusBadRequest,
		},
		"wrong_id": {
			a: &mockAuther{resetErr: fmt.Errorf("some error")},
			v: &mockValidator{
				completeotp:   "wrong_id",
				completeotpsc: http.StatusOK,
			},
			id:     "uuid",
			cookie: &http.Cookie{Name: "authn-pad"},
			sc:     http.StatusBadRequest,
		},
		"reset_fails": {
			a: &mockAuther{resetErr: fmt.Errorf("some error")},
			v: &mockValidator{
				completeotp:   "uuid",
				completeotpsc: http.StatusOK,
			},
			id:     "uuid",
			cookie: &http.Cookie{Name: "authn-pad"},
			sc:     http.StatusInternalServerError,
		},
		"change_fails": {
			a:   &mockAuther{change: fmt.Errorf("some error")},
			id:  "uuid",
			old: &pass,
			new: &pass,
			sc:  http.StatusInternalServerError,
		},
		"authn_login_fails": {
			a:   &mockAuther{login: &shared.BasicAuth{UUID: "uuid"}},
			v:   &mockValidator{login: nil, loginsc: http.StatusForbidden},
			id:  "uuid",
			old: &pass,
			new: &pass,
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

			body := mustJSON(map[string]*shared.Password{
				"old": tc.old,
				"new": tc.new,
			})
			if name == "unmarshal_fails" {
				body = body[1:]
			}

			bodyreader := io.Reader(bytes.NewReader([]byte(body)))
			if name == "read_fails" {
				bodyreader = errReader(name)
			}

			rctx := chi.NewRouteContext()
			rctx.URLParams = chi.RouteParams{Keys: []string{"user_id"}, Values: []string{string(tc.id)}}

			w := httptest.NewRecorder()
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					mockContext(),
					chi.RouteCtxKey,
					rctx),
				http.MethodGet,
				"tc.url",
				bodyreader,
			)

			if tc.cookie != nil {
				r.AddCookie(tc.cookie)
			}

			us.PatchLogin(w, r)

			require.Equal(t, tc.sc, w.Code)
		})
	}
}

func Test_DeleteLogin(t *testing.T) {
	var (
		addr    shared.Email = "addr"
		badaddr shared.Email = "badaddr"
		cell    shared.Cell  = "cell"
		badcell shared.Cell  = "badcell"
	)

	t.Parallel()

	tcs := map[string]struct {
		a     mockAuther
		u     mockUserer
		ms    mockMailSender
		ss    mockSmsSender
		v     mockValidator
		login shared.User
		input,
		user *shared.User
		loc string
		sc  int
	}{
		"happy_path": {
			a:  mockAuther{login: &shared.BasicAuth{UUID: "uuid"}},
			v:  mockValidator{token: "token"},
			ms: mockMailSender{},
			u: mockUserer{
				user: &shared.User{
					UUID:  "uuid",
					Email: &addr,
					Cell:  &cell,
				}},
			login: shared.User{
				UUID:  "uuid",
				Email: &addr,
				Cell:  &cell,
			},
			sc:  http.StatusNoContent,
			loc: "redirect",
		},
		"sms_only": {
			a:  mockAuther{login: &shared.BasicAuth{UUID: "uuid"}},
			v:  mockValidator{token: "token"},
			ms: mockMailSender{},
			u: mockUserer{
				user: &shared.User{
					UUID: "uuid",
					Cell: &cell,
				}},
			login: shared.User{
				UUID: "uuid",
				Cell: &cell,
			},
			sc:  http.StatusNoContent,
			loc: "redirect",
		},
		"read_fails": {
			sc: http.StatusBadRequest,
		},
		"unmarshal_fails": {
			sc: http.StatusBadRequest,
		},
		"login_undeliverable": {
			sc: http.StatusBadRequest,
		},
		"missing_redirect": {
			u:     mockUserer{userErr: fmt.Errorf("some error")},
			login: shared.User{Email: &addr},
			sc:    http.StatusBadRequest,
		},
		"get_user_fails": {
			u:     mockUserer{userErr: fmt.Errorf("some error")},
			login: shared.User{UUID: "uuid", Email: &addr},
			sc:    http.StatusInternalServerError,
			loc:   "redirect",
		},
		"undeliverable": {
			u:     mockUserer{user: &shared.User{UUID: "uuid"}},
			login: shared.User{UUID: "uuid", Email: &addr},
			sc:    http.StatusBadRequest,
			loc:   "redirect",
		},
		"email_mismatch": {
			u: mockUserer{user: &shared.User{
				UUID:  "uuid",
				Email: &addr,
				Cell:  &cell,
			}},
			login: shared.User{
				UUID:  "uuid",
				Email: &badaddr,
			},
			sc:  http.StatusBadRequest,
			loc: "redirect",
		},
		"cell_mismatch": {
			u: mockUserer{user: &shared.User{
				UUID:  "uuid",
				Email: &addr,
				Cell:  &cell,
			}},
			login: shared.User{
				UUID: "uuid",
				Cell: &badcell,
			},
			sc:  http.StatusBadRequest,
			loc: "redirect",
		},
		"gen_token_fails": {
			u: mockUserer{user: &shared.User{
				UUID:  "uuid",
				Email: &addr,
				Cell:  &cell,
			}},
			v: mockValidator{tokensc: http.StatusConflict},
			login: shared.User{
				UUID: "uuid",
				Cell: &cell,
			},
			sc:  http.StatusConflict,
			loc: "redirect",
		},
		"send_email_fails": {
			u: mockUserer{user: &shared.User{
				UUID:  "uuid",
				Email: &addr,
				Cell:  &cell,
			}},
			v:  mockValidator{token: "token"},
			ms: mockMailSender{err: fmt.Errorf("some error")},
			login: shared.User{
				UUID: "uuid",
				Cell: &cell,
			},
			sc:  http.StatusInternalServerError,
			loc: "redirect",
		},
		"send_sms_fails": {
			u: mockUserer{user: &shared.User{
				UUID:  "uuid",
				Email: &addr,
			}},
			v:  mockValidator{token: "token"},
			ss: mockSmsSender{err: fmt.Errorf("some error")},
			login: shared.User{
				UUID:  "uuid",
				Email: &addr,
			},
			sc:  http.StatusInternalServerError,
			loc: "redirect",
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{
				Auther:     &tc.a,
				MailSender: &tc.ms,
				SmsSender:  &tc.ss,
				Userer:     &tc.u,
				Validator:  &tc.v,
			}

			body := userToDelete(&tc.login, tc.loc)
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
					mockContext(),
					chi.RouteCtxKey,
					chi.NewRouteContext()),
				http.MethodDelete,
				"tc.url",
				bodyreader,
			)

			us.DeleteLogin(w, r)

			resp, err := io.ReadAll(w.Body)
			require.Nil(t, err)

			require.Equal(t, tc.sc, w.Code, string(resp))
		})
	}
}

func authToBody(a *shared.BasicAuth) string {
	result, _ := json.Marshal(a)
	return string(result)
}

func (ma *mockAuther) GetAuthByAttrs(context.Context, *shared.UUID, *string) (*shared.BasicAuth, error) {
	return ma.get, ma.getErr
}
func (ma *mockAuther) ChangePassword(context.Context, shared.UUID, shared.Password, shared.Password) error {
	return ma.change
}
func (ma *mockAuther) Login(context.Context, *shared.BasicAuth) (*shared.BasicAuth, error) {
	return ma.login, ma.loginErr
}
func (ma *mockAuther) ResetPassword(context.Context, *shared.UUID) (shared.Password, error) {
	return ma.reset, ma.resetErr
}
