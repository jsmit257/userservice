package router

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-gomail/gomail"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"

	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/internal/metrics"
	"github.com/jsmit257/userservice/shared/v1"
)

type (
	errReader string

	mockContextHandler struct {
		sc int
	}

	mockRoutes struct {
		routes  []chi.Route
		middles chi.Middlewares
		matched bool
		found   string
	}

	mockMailSender struct {
		msgs int
		err  error
	}

	mockSmsSender struct {
		msgs int
		err  error
	}
)

func Test_NewInstance(t *testing.T) {
	t.Parallel()

	os.Setenv("MYSQL_HOST", "localhost")
	os.Setenv("MYSQL_PORT", "1234")
	os.Setenv("MYSQL_USER", "fake")
	os.Setenv("MYSQL_PASSWORD", "snakeoil")

	_ = NewInstance(&UserService{
		Addresser: &mockAddresser{},
		Auther:    &mockAuther{},
		Contacter: &mockContacter{},
		Userer:    &mockUserer{},
		Validator: &mockValidator{},
	}, config.NewConfig(), nil)
}

func Test_WrapContext(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		mr *mockRoutes
		sc int
	}{
		"happy_path": {
			sc: http.StatusTeapot,
			mr: &mockRoutes{},
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			wrap := wrapContext(logrus.WithField("test", name))
			handle := wrap(&mockContextHandler{sc: tc.sc})
			w := httptest.NewRecorder()
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					context.TODO(),
					chi.RouteCtxKey,
					&chi.Context{
						URLParams: chi.RouteParams{Keys: []string{"user_id"}, Values: []string{name}},
						Routes:    tc.mr,
					}),
				http.MethodGet,
				"tc.url",
				nil,
			)
			r.URL.RawPath = "not-empty"

			handle.ServeHTTP(w, r)
			require.Equal(t, tc.sc, w.Result().StatusCode)
		})
	}
}

func Test_Settings(t *testing.T) {
	t.Skip()
	t.Parallel()

	handler := settings(&config.Config{})

	tcs := map[string]struct {
		name  string
		value map[string]interface{}
		sc    int
	}{
		"get_all": {
			name: "*",
			value: map[string]interface{}{
				"authn_path":   "foobar",
				"authn_port":   float64(1234),
				"huautla_host": "quux",
			},
			sc: http.StatusOK,
		},
		"get_string": {
			name:  "authn_path",
			value: map[string]interface{}{"value": "foobar"},
			sc:    http.StatusOK,
		},
		"get_number": {
			name:  "authn_port",
			value: map[string]interface{}{"value": float64(1234)},
			sc:    http.StatusOK,
		},
		"not_found": {
			name:  "bad_key",
			value: map[string]interface{}{"error": "no value for key: bad_key"},
			sc:    http.StatusNotFound,
		},
	}

	for name, tc := range tcs {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			defer w.Result().Body.Close()
			rctx := chi.NewRouteContext()
			rctx.URLParams = chi.RouteParams{Keys: []string{"name"}, Values: []string{tc.name}}
			r, _ := http.NewRequestWithContext(
				context.WithValue(
					context.Background(), // metrics.MockServiceContext,
					chi.RouteCtxKey,
					rctx),
				http.MethodGet,
				"url",
				nil)

			handler(w, r)

			body, err := io.ReadAll(w.Body)
			require.Nil(t, err)
			t.Log(string(body))
			var result any
			err = json.Unmarshal(body, &result)
			require.Nil(t, err, string(body))

			require.Equal(t, tc.sc, w.Code)
			require.Equal(t, tc.value, result)
		})
	}
}

func mockContext() context.Context {
	return context.WithValue(
		context.WithValue(
			context.WithValue(
				context.Background(),
				shared.CTXKey("log"),
				logrus.WithField("app", "test"),
			),
			shared.CTXKey("metrics"),
			metrics.ServiceMetrics.MustCurryWith(prometheus.Labels{
				"proto":  "test",
				"method": "test",
				"url":    "test",
			}),
		),
		shared.CTXKey("cid"),
		shared.CID("test"),
	)
}

func (ms *mockMailSender) Send(*gomail.Message) error {
	ms.msgs++
	return ms.err
}
func (ms *mockMailSender) Close() {}

func (ss *mockSmsSender) Send(*twilioApi.CreateMessageParams) error {
	ss.msgs++
	return ss.err
}
func (ss *mockSmsSender) Close() {}

func (er errReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("%s", er)
}

func (f *mockContextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(f.sc)
}

func (mr *mockRoutes) Routes() []chi.Route {
	return mr.routes
}
func (mr *mockRoutes) Middlewares() chi.Middlewares {
	return mr.middles
}
func (mr *mockRoutes) Match(rctx *chi.Context, method, path string) bool {
	return mr.matched
}
func (mr *mockRoutes) Find(rctx *chi.Context, method, path string) string {
	return mr.found
}
