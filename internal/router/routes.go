package router

import (
	"net/http"

	"github.com/jsmit257/userservice/internal/data"
	"github.com/jsmit257/userservice/internal/metrics"

	"github.com/go-chi/chi/v5"

	"github.com/prometheus/client_golang/prometheus"
)

type UserService struct {
	data.Address
	data.Contact
	data.User
}

var mtrcs = metrics.ServiceMetrics.MustCurryWith(prometheus.Labels{})

func NewInstance(us *UserService) *http.Server {
	r := chi.NewRouter()

	// r.Use(middleware.Logger)

	r.Get("/user/{user_id}", us.GetUser)
	r.Patch("/user", us.PatchUser)
	r.Post("/user", us.PostUser)
	// r.Post("/user/{user_id}/contact", us.PostUser)

	r.Get("/hc", hc)

	return &http.Server{Addr: ":3000", Handler: r}
}

// not much of a healthcheck, for now
func hc(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
