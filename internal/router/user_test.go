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

type mockUserer struct {
	users             []shared.User
	getUsersErr       error
	user              *shared.User
	userErr           error
	postUserResp      *shared.User
	postUserErr       error
	patchUserErr      error
	createContactResp *shared.Contact
	createContactErr  error
	rmUserErr         error
}

func Test_GetAllUsers(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		u        *mockUserer
		response string
		sc       int
	}{
		"happy_path": {
			u: &mockUserer{
				users: []shared.User{{UUID: "1"}},
			},
			response: func() string {
				result, _ := json.Marshal([]shared.User{{UUID: "1"}})
				return string(result)
			}(),
			sc: http.StatusOK,
		},
		"get_user_fails": {
			u: &mockUserer{
				getUsersErr: fmt.Errorf("some error"),
			},
			sc:       http.StatusBadRequest,
			response: "some error",
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{Userer: tc.u}
			w := httptest.NewRecorder()
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					mockContext(),
					chi.RouteCtxKey,
					chi.NewRouteContext()),
				http.MethodGet,
				"tc.url",
				nil,
			)

			us.GetAllUsers(w, r)

			resp, _ := io.ReadAll(w.Body)
			require.Equal(t, tc.sc, w.Code)
			require.Equal(t, tc.response, string(resp))
		})
	}
}

func Test_GetUser(t *testing.T) {
	t.Parallel()
	tcs := map[string]struct {
		u        *mockUserer
		userIDs  []string
		response string
		sc       int
	}{
		"happy_path": {
			u: &mockUserer{
				user: &shared.User{UUID: "1"},
			},
			userIDs: []string{"1"},
			response: func() string {
				result, _ := json.Marshal(shared.User{UUID: "1"})
				return string(result)
			}(),
			sc: http.StatusOK,
		},
		"get_user_fails": {
			u: &mockUserer{
				userErr: fmt.Errorf("some error"),
			},
			userIDs:  []string{"1"},
			sc:       http.StatusBadRequest,
			response: "some error",
		},
		"user_is_nil": {
			u:        &mockUserer{},
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
					mockContext(),
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

func Test_PostUser(t *testing.T) {
	t.Parallel()

	addr := shared.Email("email")

	tcs := map[string]struct {
		u        *mockUserer
		r        *shared.User
		sc       int
		response string
	}{
		"happy_path": {
			u: &mockUserer{
				postUserResp: &shared.User{UUID: "1"},
			},
			r: &shared.User{
				UUID:  "1",
				Email: &addr,
			},
			sc:       http.StatusCreated,
			response: "1",
		},
		"unmarshal_fails": {
			u:  &mockUserer{},
			r:  &shared.User{},
			sc: http.StatusBadRequest,
		},
		"read_fails": {
			u:  &mockUserer{},
			r:  &shared.User{},
			sc: http.StatusBadRequest,
		},
		"undeliverable": {
			r:        &shared.User{},
			sc:       http.StatusBadRequest,
			response: "no valid email or SMS provided",
		},
		"adduser_fails": {
			u: &mockUserer{
				postUserResp: &shared.User{},
				postUserErr:  fmt.Errorf("some error"),
			},
			r:  &shared.User{Email: &addr},
			sc: http.StatusInternalServerError,
		},
		"user_exists": {
			u: &mockUserer{
				postUserResp: &shared.User{},
				postUserErr:  shared.UserExistsError,
			},
			r: &shared.User{
				UUID:  "1",
				Email: &addr,
			},
			sc:       http.StatusBadRequest,
			response: shared.UserExistsError.Error(),
		},
		"user_not_added": {
			u: &mockUserer{
				postUserResp: &shared.User{},
				postUserErr:  shared.UserNotAddedError,
			},
			r: &shared.User{
				UUID:  "1",
				Email: &addr,
			},
			sc: http.StatusConflict,
		},
		"user_not_found": {
			u: &mockUserer{
				postUserResp: &shared.User{},
			},
			r: &shared.User{
				Email: &addr,
			},
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
			bodyreader := io.Reader(bytes.NewReader([]byte(body)))
			if name == "read_fails" {
				bodyreader = errReader(name)
			}

			r, _ := http.NewRequestWithContext(
				context.WithValue(
					mockContext(),
					chi.RouteCtxKey,
					chi.NewRouteContext()),
				http.MethodPost,
				"tc.url",
				bodyreader)

			us.PostUser(w, r)
			resp, _ := io.ReadAll(w.Body)
			require.Equal(t, tc.sc, w.Code)
			require.Equal(t, tc.response, string(resp))
		})
	}
}

func Test_PatchUser(t *testing.T) {
	t.Parallel()

	addr := shared.Email("addr")

	tcs := map[string]struct {
		u       *mockUserer
		r       *shared.User
		user    *shared.User
		userIDs []string
		sc      int
	}{
		"happy_path": {
			u:       &mockUserer{},
			userIDs: []string{"1"},
			r: &shared.User{
				UUID:  "1",
				Email: &addr,
			},
			sc: http.StatusNoContent,
		},
		"missing_param": {
			u:       &mockUserer{},
			userIDs: []string{""},
			r:       &shared.User{},
			sc:      http.StatusBadRequest,
		},
		"unmarshal_fails": {
			u:       &mockUserer{},
			userIDs: []string{"1"},
			r:       &shared.User{UUID: "1"},
			sc:      http.StatusBadRequest,
		},
		"read_fails": {
			u:       &mockUserer{},
			userIDs: []string{"1"},
			r:       &shared.User{UUID: "1"},
			sc:      http.StatusBadRequest,
		},
		"undeliverable": {
			r:       &shared.User{UUID: "1"},
			userIDs: []string{"1"},
			sc:      http.StatusBadRequest,
		},
		"update_fails": {
			u:       &mockUserer{patchUserErr: fmt.Errorf("some error")},
			userIDs: []string{"1"},
			r: &shared.User{
				UUID:  "1",
				Email: &addr,
			},
			sc: http.StatusInternalServerError,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			us := &UserService{Userer: tc.u}

			body := userToBody(tc.r)
			if name == "unmarshal_fails" {
				body = body[1:]
			}
			bodyreader := io.Reader(bytes.NewReader([]byte(body)))
			if name == "read_fails" {
				bodyreader = errReader(name)
			}

			w := httptest.NewRecorder()
			rctx := chi.NewRouteContext()
			rctx.URLParams = chi.RouteParams{Keys: []string{"user_id"}, Values: tc.userIDs}
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					mockContext(),
					chi.RouteCtxKey,
					rctx),
				http.MethodPatch,
				"tc.url",
				bodyreader)

			us.PatchUser(w, r)

			require.Equal(t, tc.sc, w.Code)
		})
	}
}

func Test_DeleteUser(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		u  *mockUserer
		sc int
	}{
		"happy_path": {
			u:  &mockUserer{},
			sc: http.StatusNoContent,
		},
		"rm_user_fails": {
			u: &mockUserer{
				rmUserErr: fmt.Errorf("some error"),
			},
			sc: http.StatusBadRequest,
		},
	}
	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{Userer: tc.u}
			w := httptest.NewRecorder()
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					mockContext(),
					chi.RouteCtxKey,
					chi.NewRouteContext()),
				http.MethodGet,
				"tc.url",
				nil,
			)

			us.DeleteUser(w, r)

			require.Equal(t, tc.sc, w.Code)
		})
	}
}

func Test_CreateContact(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		u       *mockUserer
		userid  shared.UUID
		contact *shared.Contact
		sc      int
	}{
		"happy_path": {
			userid: "create_contact",
			u: &mockUserer{
				user:              &shared.User{},
				createContactResp: &shared.Contact{},
			},
			contact: &shared.Contact{},
			sc:      http.StatusOK,
		},
		"get_user_fails": {
			userid: "create_contact",
			u:      &mockUserer{userErr: fmt.Errorf("some error")},
			sc:     http.StatusBadRequest,
		},
		"param_fails": {
			u:  &mockUserer{user: &shared.User{}},
			sc: http.StatusBadRequest,
		},
		"read_fails": {
			userid: "create_contact",
			u:      &mockUserer{user: &shared.User{}},
			sc:     http.StatusBadRequest,
		},
		"unmarshal_fails": {
			userid:  "create_contact",
			u:       &mockUserer{user: &shared.User{}},
			contact: &shared.Contact{},
			sc:      http.StatusBadRequest,
		},
		"create_contact_fails": {
			userid: "create_contact",
			u:      &mockUserer{createContactErr: fmt.Errorf("some error")},
			sc:     http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			us := &UserService{Userer: tc.u}

			body := contactToBody(tc.contact)
			if name == "unmarshal_fails" {
				body = body[1:]
			}
			bodyreader := io.Reader(bytes.NewReader([]byte(body)))
			if name == "read_fails" {
				bodyreader = errReader(name)
			}

			w := httptest.NewRecorder()
			rctx := chi.NewRouteContext()
			rctx.URLParams = chi.RouteParams{Keys: []string{"user_id"}, Values: []string{string(tc.userid)}}
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					mockContext(),
					chi.RouteCtxKey,
					rctx),
				http.MethodGet,
				"tc.url",
				bodyreader)

			us.CreateContact(w, r)

			require.Equal(t, tc.sc, w.Code)
		})
	}
}

func userToBody(u *shared.User) string {
	result, _ := json.Marshal(u)
	return string(result)
}
func userToDelete(u *shared.User, redirect string) string {
	result, _ := json.Marshal(u)
	temp := map[string]interface{}{}
	_ = json.Unmarshal(result, &temp)
	temp["redirect"] = redirect
	result, _ = json.Marshal(temp)
	return string(result)
}

func (mu *mockUserer) GetAllUsers(context.Context) ([]shared.User, error) {
	return mu.users, mu.getUsersErr
}
func (mu *mockUserer) GetUser(context.Context, shared.UUID) (*shared.User, error) {
	return mu.user, mu.userErr
}
func (mu *mockUserer) AddUser(context.Context, *shared.User) (shared.UUID, error) {
	return mu.postUserResp.UUID, mu.postUserErr
}
func (mu *mockUserer) UpdateUser(context.Context, *shared.User) error {
	return mu.patchUserErr
}
func (mu *mockUserer) CreateContact(context.Context, *shared.User, shared.Contact) (*shared.Contact, error) {
	return mu.createContactResp, mu.createContactErr
}
func (mu *mockUserer) DeleteUser(context.Context, shared.UUID) error { // unused
	return mu.rmUserErr
}
