package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jsmit257/userservice/shared/v1"
	"github.com/stretchr/testify/require"
)

type mockValidator struct {
	login   *http.Cookie
	loginsc int

	logoutsc,
	validsc int

	token   string
	tokensc int

	validotp   *http.Cookie
	validotpsc int
}

var testCookie = http.Cookie{
	Name:     "us-authn",
	Path:     "/",
	Expires:  time.Time{},
	MaxAge:   -1,
	HttpOnly: true,
	Raw:      "us-authn=; Path=/; Max-Age=0; HttpOnly",
}

func Test_PostLogout(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		token string
		mv    *mockValidator
		sc    int
	}{
		"pass_through": {
			token: "foobar",
			mv:    &mockValidator{logoutsc: http.StatusFound},
			sc:    http.StatusFound,
		},
		"missing_token": {
			sc: http.StatusForbidden,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{Validator: tc.mv}
			w := httptest.NewRecorder()
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					mockContext(),
					chi.RouteCtxKey,
					chi.NewRouteContext()),
				http.MethodPost,
				"tc.url",
				nil,
			)
			if tc.token != "" {
				r.AddCookie(&http.Cookie{
					Name:    "us-authn",
					Value:   tc.token,
					Expires: time.Now().UTC().Add(time.Hour),
				})
			}

			us.PostLogout(w, r)

			require.Equal(t, tc.sc, w.Code)
			if w.Code == http.StatusFound {
				require.Subset(t, w.Result().Cookies(), []*http.Cookie{&testCookie})
			} else {
				require.NotSubset(t, w.Result().Cookies(), []*http.Cookie{&testCookie})
			}
		})
	}
}

func Test_GetValid(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		token string
		mv    *mockValidator
		sc    int
	}{
		"pass_through": {
			token: "foobar",
			mv:    &mockValidator{validsc: http.StatusFound},
			sc:    http.StatusFound,
		},
		"missing_token": {
			sc: http.StatusForbidden,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{Validator: tc.mv}
			w := httptest.NewRecorder()
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					mockContext(),
					chi.RouteCtxKey,
					chi.NewRouteContext()),
				http.MethodPost,
				"tc.url",
				nil,
			)
			if tc.token != "" {
				r.AddCookie(&http.Cookie{
					Name:    "us-authn",
					Value:   tc.token,
					Expires: time.Now().UTC().Add(time.Hour),
				})
			}

			us.GetValid(w, r)

			require.Equal(t, tc.sc, w.Code)
		})
	}
}

func Test_GetValidOTP(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		pad string
		mv  *mockValidator
		sc  int
	}{
		"happy_path": {
			pad: "foobar",
			mv:  &mockValidator{validotpsc: http.StatusFound},
			sc:  http.StatusFound,
		},
		"missing_token": {
			sc: http.StatusBadRequest,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{Validator: tc.mv}

			rctx := chi.NewRouteContext()
			rctx.URLParams = chi.RouteParams{Keys: []string{"pad"}, Values: []string{tc.pad}}
			w := httptest.NewRecorder()
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					mockContext(),
					chi.RouteCtxKey,
					rctx),
				http.MethodPost,
				"tc.url",
				nil,
			)
			if tc.pad != "" {
				r.AddCookie(&http.Cookie{
					Name:    "us-authn",
					Value:   tc.pad,
					Expires: time.Now().UTC().Add(time.Hour),
				})
			}

			us.GetValidOTP(w, r)

			require.Equal(t, tc.sc, w.Code)
		})
	}
}

func (mv *mockValidator) Login(context.Context, shared.UUID, string) (*http.Cookie, int) {
	return mv.login, mv.loginsc
}
func (mv *mockValidator) Logout(context.Context, string) (*http.Cookie, int) {
	return &testCookie, mv.logoutsc
}
func (mv *mockValidator) Valid(context.Context, string) (*http.Cookie, int) {
	return &testCookie, mv.validsc
}
func (mv *mockValidator) OTP(context.Context, shared.UUID, string) (string, int) {
	return mv.token, mv.tokensc
}
func (mv *mockValidator) ValidOTP(context.Context, string, string) (*http.Cookie, int) {
	return mv.validotp, mv.validotpsc
}
