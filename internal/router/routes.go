package router

import (
	"net/http"

	"github.com/jsmit257/userservice/internal/data"
	"github.com/jsmit257/userservice/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/go-chi/chi/v5"
)

type UserService struct {
	data.Address
	data.Contact
	data.User
}

var mtrcs = metrics.ServiceMetrics.MustCurryWith(prometheus.Labels{})

func NewInstance(us *UserService) error {
	r := chi.NewRouter()

	// r.Use(middleware.Logger)

	r.Get("/user/{user_id}", us.GetUser)
	r.Patch("/user", us.PatchUser)
	r.Post("/user", us.PostUser)
	// r.Post("/user/{user_id}/contact", us.PostUser)

	return http.ListenAndServe(":3000", r)
}
