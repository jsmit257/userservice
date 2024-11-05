package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/go-chi/chi/v5"

	log "github.com/sirupsen/logrus"

	"github.com/jsmit257/userservice/internal/config"
	"github.com/jsmit257/userservice/internal/metrics"
	valid "github.com/jsmit257/userservice/internal/validation"
	"github.com/jsmit257/userservice/shared/v1"
)

type (
	UserService struct {
		shared.Addresser
		shared.Auther
		shared.Contacter
		shared.Userer
		valid.Validator
	}

	sc int
)

var mtrcs = metrics.ServiceMetrics.MustCurryWith(prometheus.Labels{
	"pkg": "router",
	"app": "userservice",
})

func (sc sc) send(m *prometheus.CounterVec, w http.ResponseWriter, err error, messages ...string) {
	m.WithLabelValues(strconv.Itoa(int(sc)), err.Error()).Inc()
	w.WriteHeader(int(sc))
	for _, m := range messages {
		_, _ = w.Write([]byte(m))
	}
}

func (sc sc) success(m *prometheus.CounterVec, w http.ResponseWriter, messages ...string) {
	w.Header().Add("Content-Type", "application/json")
	sc.send(m, w, fmt.Errorf("none"), messages...)
}

func middleware(http.Handler) http.Handler {
	return nil
}

func NewInstance(us *UserService, cfg *config.Config, logger *log.Entry) *http.Server {
	r := chi.NewRouter()

	// r.Use(middleware)

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

	r.Post("/auth/{token}/logout", us.Logout)
	r.Get("/auth/{token}/valid", us.Valid)

	r.Get("/hc", hc)

	r.Get("/metrics", metrics.NewHandler(prometheus.NewRegistry()))

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

func mustJSON(a any) string {
	response, _ := json.Marshal(a)
	return string(response)
}

func cid() shared.CID {
	return shared.CID(uuid.NewString())
}
