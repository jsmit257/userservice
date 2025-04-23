package router

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/internal/maild"
	"github.com/jsmit257/userservice/internal/metrics"
	valid "github.com/jsmit257/userservice/internal/validation"
	"github.com/jsmit257/userservice/shared/v1"
)

type (
	UserService struct {
		maild.Sender
		shared.Addresser
		shared.Auther
		shared.Contacter
		shared.Userer
		valid.Validator
	}

	sc int
)

func NewInstance(us *UserService, cfg *config.Config, log *logrus.Entry) *http.Server {
	r := chi.NewRouter()

	r.Use(wrapContext(log))

	r.Get("/users", us.GetAllUsers)
	r.Get("/user/{user_id}", us.GetUser)
	r.Post("/user", us.PostUser)
	r.Patch("/user/{user_id}", us.PatchUser)
	r.Delete("/user/{user_id}", us.DeleteUser)
	r.Post("/user/{user_id}/contact", us.CreateContact)

	r.Patch("/contact/{user_id}", us.PatchContact)

	r.Get("/addresses", us.GetAllAddresses)
	r.Get("/address/{address_id}", us.GetAddress)
	r.Post("/address", us.PostAddress)
	r.Patch("/address/{address_id}", us.PatchAddress)

	r.Get("/auth/{username}", us.GetAuth)
	r.Post("/auth", us.PostLogin)
	r.Patch("/auth/{user_id}", us.PatchLogin)
	r.Delete("/auth", us.DeleteLogin)

	r.Post("/logout", us.PostLogout)
	r.Get("/valid", us.GetValid)
	r.Get("/otp/{pad}", us.GetLoginOTP)
	r.Get("/validateotp", us.PostValidateOTP)

	r.Get("/hc", hc)

	r.Get("/metrics", metrics.NewHandler())

	return &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort),
		Handler: r,
	}
}

// not much of a healthcheck, for now
func hc(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// shamelessly copied from https://github.com/go-chi/chi/issues/270#issuecomment-479184559
func getRoutePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if pattern := rctx.RoutePattern(); pattern != "" {
		return "!" + pattern // leaving the bang to see if this ever happens (and how?)
	}

	routePath := r.URL.Path
	if r.URL.RawPath != "" {
		routePath = r.URL.RawPath
	}

	tctx := chi.NewRouteContext()
	if rctx.Routes.Match(tctx, r.Method, routePath) {
		return tctx.RoutePattern()
	}

	// better than logging or panicing, as long as it never happens
	return "!!" + routePath
}

func wrapContext(log *logrus.Entry) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			cid := shared.CID(uuid.NewString())

			log = log.WithFields(logrus.Fields{
				"method": r.Method,
				"remote": r.RemoteAddr, // is this necessary?
				"url":    r.RequestURI, // this, or getRoutePattern and fill in the blanks with more fields?
				"cid":    cid,
			})

			m := metrics.ServiceMetrics.MustCurryWith(prometheus.Labels{
				"proto":  r.Proto,
				"method": r.Method,
				"url":    getRoutePattern(r),
			})

			w.Header().Set("Cid", string(cid))
			r = r.WithContext(context.WithValue(r.Context(), shared.CTXKey("cid"), cid))
			r = r.WithContext(context.WithValue(r.Context(), shared.CTXKey("log"), log))
			r = r.WithContext(context.WithValue(r.Context(), shared.CTXKey("metrics"), m))

			log.Info("started request")

			next.ServeHTTP(w, r)

			log.WithField("duration", time.Since(start).String()).Info("finished request")
		})
	}
}

func (sc sc) send(ctx context.Context, w http.ResponseWriter, err error, messages ...string) {
	l := ctx.Value(shared.CTXKey("log")).(*logrus.Entry)
	if err != nil {
		l = l.WithError(err)
		// if len(messages) == 0 {
		// 	messages = append(messages, err.Error())
		// }
	}
	l.WithField("status-code", sc).Info("sending status code and messages")

	ctx.
		Value(shared.CTXKey("metrics")).(*prometheus.CounterVec).
		WithLabelValues(strconv.Itoa(int(sc))).
		Inc()

	w.WriteHeader(int(sc))
	for _, m := range messages {
		_, _ = w.Write([]byte(html.EscapeString(m)))
	}
}

func (sc sc) success(ctx context.Context, w http.ResponseWriter, messages ...string) {
	w.Header().Add("Content-Type", "application/json")
	sc.send(ctx, w, nil, messages...)
}

func mustJSON(a any) string {
	response, _ := json.Marshal(a)
	return string(response)
}
