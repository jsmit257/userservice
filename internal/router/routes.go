package router

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/go-chi/chi/v5"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"

	"github.com/jsmit257/userservice/internal/metrics"
	shared "github.com/jsmit257/userservice/shared/v1"
)

type UserService struct {
	shared.Addresser
	shared.Contacter
	shared.Userer
}

var mtrcs = metrics.ServiceMetrics.MustCurryWith(prometheus.Labels{})

func middleware(http.Handler) http.Handler {
	return nil
}

func NewInstance(us *UserService, hostAddr string, hostPort uint16, mtrcs http.HandlerFunc, logger *log.Entry) *http.Server {
	r := chi.NewRouter()

	// r.Use(middleware)

	r.Get("/user/{user_id}", us.GetUser)
	r.Patch("/user/{user_id}", us.PatchUser)
	r.Post("/user", us.PostUser)
	// r.Post("/user/{user_id}/contact", us.PostUser)

	r.Get("/hc", hc)

	r.Get("/metrics", mtrcs)

	return &http.Server{
		Addr:    fmt.Sprintf("%s:%d", hostAddr, hostPort),
		Handler: r,
	}
}

// not much of a healthcheck, for now
func hc(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func cid() shared.CID {
	return shared.CID(uuid.NewString())
}
