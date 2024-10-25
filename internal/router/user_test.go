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

	"github.com/jsmit257/userservice/internal/data"
	"github.com/jsmit257/userservice/shared/v1"

	"github.com/go-chi/chi/v5"

	"github.com/stretchr/testify/require"
)

type mockUser struct {
	authenticateResp  *shared.User
	authenticateErr   error
	getUserResp       *shared.User
	getUserErr        error
	postUserResp      *shared.User
	postUserErr       error
	patchUserErr      error
	createContactResp *shared.Contact
	createContactErr  error
}

func Test_userService_PatchUser(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		u       *mockUser
		r       *shared.User
		user    *shared.User
		userIDs []string
		sc      int
	}{
		"happy_path": {
			u: &mockUser{
				getUserErr: fmt.Errorf("some error"),
			},
			userIDs: []string{"1"},
			r:       &shared.User{UUID: "1"},
			sc:      http.StatusNoContent,
		},
		"unmarshal_fails": {
			u: &mockUser{
				getUserErr: fmt.Errorf("some error"),
			},
			userIDs: []string{"1"},
			r:       &shared.User{UUID: "1"},
			sc:      http.StatusInternalServerError,
		},
		"bad_userid": {
			u: &mockUser{
				getUserErr: fmt.Errorf("some error"),
			},
			userIDs: []string{"1"},
			r:       &shared.User{UUID: "2"},
			sc:      http.StatusBadRequest,
		},
		"update_fails": {
			u: &mockUser{
				patchUserErr: fmt.Errorf("some error"),
			},
			userIDs: []string{"1"},
			r:       &shared.User{UUID: "1"},
			sc:      http.StatusInternalServerError,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			us := &UserService{
				Userer: tc.u,
			}
			w := httptest.NewRecorder()
			rctx := chi.NewRouteContext()
			rctx.URLParams = chi.RouteParams{Keys: []string{"user_id"}, Values: tc.userIDs}
			body := userToBody(tc.r)
			if name == "unmarshal_fails" {
				body = body[1:]
			}
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					context.Background(),
					chi.RouteCtxKey,
					rctx),
				http.MethodPatch,
				"tc.url",
				bytes.NewReader([]byte(body)),
			)
			us.PatchUser(w, r)
			require.Equal(t, tc.sc, w.Code)
		})
	}
}

func Test_userService_GetUser(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		u        *mockUser
		userIDs  []string
		response string
		sc       int
	}{
		"happy_path": {
			u: &mockUser{
				getUserResp: &shared.User{UUID: "1"},
			},
			userIDs: []string{"1"},
			response: func() string {
				result, _ := json.Marshal(shared.User{UUID: "1"})
				return string(result)
			}(),
			sc: http.StatusOK,
		},
		"get_user_fails": {
			u: &mockUser{
				getUserErr: fmt.Errorf("some error"),
			},
			userIDs:  []string{"1"},
			sc:       http.StatusBadRequest,
			response: "some error",
		},
		"user_is_nil": {
			u:        &mockUser{},
			userIDs:  []string{""},
			sc:       http.StatusOK,
			response: "null",
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			us := &UserService{Userer: tc.u}
			w := httptest.NewRecorder()
			rctx := chi.NewRouteContext()
			rctx.URLParams = chi.RouteParams{Keys: []string{"user_id"}, Values: tc.userIDs}
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					context.Background(),
					chi.RouteCtxKey,
					rctx),
				http.MethodGet,
				"tc.url",
				nil,
			)
			// t.Errorf("%#v\n", tc.u)
			us.GetUser(w, r)
			resp, _ := io.ReadAll(w.Body)
			require.Equal(t, tc.sc, w.Code)
			require.Equal(t, tc.response, string(resp))

		})
	}
}

func Test_userService_PostUser(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		u        *mockUser
		r        *shared.User
		sc       int
		response string
	}{
		"happy_path": {
			u: &mockUser{
				postUserResp: &shared.User{UUID: "1"},
			},
			r:        &shared.User{UUID: "1"},
			sc:       http.StatusMovedPermanently,
			response: "1",
		},
		"unmarshal_fails": {
			u:  &mockUser{},
			r:  &shared.User{},
			sc: http.StatusBadRequest,
		},
		"adduser_fails": {
			u: &mockUser{
				postUserResp: &shared.User{},
				postUserErr:  fmt.Errorf("some error"),
			},
			r:  &shared.User{},
			sc: http.StatusInternalServerError,
		},
		"user_exists": {
			u: &mockUser{
				postUserResp: &shared.User{},
				postUserErr:  data.UserExistsError,
			},
			r:  &shared.User{UUID: "1"},
			sc: http.StatusBadRequest,
		},
		"user_not_added": {
			u: &mockUser{
				postUserResp: &shared.User{},
				postUserErr:  data.UserNotAddedError,
			},
			r:  &shared.User{UUID: "1"},
			sc: http.StatusInternalServerError,
		},
		"user_not_found": {
			u: &mockUser{
				postUserResp: &shared.User{},
			},
			r:  &shared.User{},
			sc: http.StatusInternalServerError,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			us := &UserService{
				Userer: tc.u,
			}
			w := httptest.NewRecorder()
			body := userToBody(tc.r)
			if name == "unmarshal_fails" {
				body = body[1:]
			}
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					context.Background(),
					chi.RouteCtxKey,
					chi.NewRouteContext()),
				http.MethodPost,
				"tc.url",
				bytes.NewReader([]byte(body)),
			)
			us.PostUser(w, r)
			resp, _ := io.ReadAll(w.Body)
			require.Equal(t, tc.sc, w.Code)
			require.Equal(t, tc.response, string(resp))
		})
	}
}

func userToBody(u *shared.User) string {
	result, _ := json.Marshal(u)
	return string(result)
}

func (mu *mockUser) BasicAuth(ctx context.Context, login *shared.BasicAuth, cid shared.CID) (*shared.User, error) {
	return mu.authenticateResp, mu.authenticateErr
}
func (mu *mockUser) GetUser(ctx context.Context, id shared.UUID, cid shared.CID) (*shared.User, error) {
	return mu.getUserResp, mu.getUserErr
}
func (mu *mockUser) AddUser(ctx context.Context, u *shared.User, cid shared.CID) (shared.UUID, error) {
	return mu.postUserResp.UUID, mu.postUserErr
}
func (mu *mockUser) UpdateUser(ctx context.Context, u *shared.User, cid shared.CID) error {
	return mu.patchUserErr
}
func (mu *mockUser) CreateContact(ctx context.Context, u *shared.User, c *shared.Contact, cid shared.CID) (*shared.Contact, error) {
	return mu.createContactResp, mu.createContactErr
}
func (mu *mockUser) DeleteUser(ctx context.Context, id shared.UUID, cid shared.CID) error { // unused
	return nil
}
