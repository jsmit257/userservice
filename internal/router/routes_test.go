package router

import (
	"context"
	"fmt"
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
