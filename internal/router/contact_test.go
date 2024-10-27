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

type mockContacter struct {
	updResp error
}

func Test_PatchContact(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		c       *mockContacter
		userid  shared.UUID
		contact shared.Contact
		sc      int
	}{
		"happy_path": {
			c:  &mockContacter{},
			sc: http.StatusOK,
		},
		"unmarshal_fails": {
			c:  &mockContacter{updResp: fmt.Errorf("some error")},
			sc: http.StatusBadRequest,
		},
		"read_fails": {
			c:  &mockContacter{},
			sc: http.StatusBadRequest,
		},
		"update_fails": {
			c:  &mockContacter{updResp: fmt.Errorf("some error")},
			sc: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			body := contactToBody(&tc.contact)
			if name == "unmarshal_fails" {
				body = body[1:]
			}
			bodyreader := io.Reader(bytes.NewReader([]byte(body)))
			if name == "read_fails" {
				bodyreader = errReader(name)
			}

			us := &UserService{Contacter: tc.c}
			w := httptest.NewRecorder()
			rctx := chi.NewRouteContext()
			rctx.URLParams = chi.RouteParams{Keys: []string{"user_id"}, Values: []string{string(tc.userid)}}
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					context.Background(),
					chi.RouteCtxKey,
					rctx),
				http.MethodGet,
				"tc.url",
				bodyreader)

			us.PatchContact(w, r)

			require.Equal(t, tc.sc, w.Code)
		})
	}
}

func contactToBody(c *shared.Contact) string {
	result, _ := json.Marshal(c)
	return string(result)
}

func (mc *mockContacter) UpdateContact(context.Context, shared.UUID, *shared.Contact, shared.CID) error {
	return mc.updResp
}
